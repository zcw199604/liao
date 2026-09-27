package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"mime/multipart"
	"sort"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"liao/internal/database"
)

// ErrPHashUnsupported 表示该文件无法计算 pHash（非图片或解码失败等）。
var ErrPHashUnsupported = errors.New("无法计算pHash（仅支持可解码的图片格式）")

const (
	phashSize      = 32
	phashLowFreq   = 8
	phashBitLength = 64
)

var (
	phashDCTFactors [phashSize]float64
)

var (
	resizeToGrayFn    = resizeToGray
	dctLowFreq8x8Fn   = dctLowFreq8x8
	lanczosKernelFunc = lanczos
)

func init() {
	for n := 2; n <= phashSize; n *= 2 {
		for i := 0; i < n/2; i++ {
			phashDCTFactors[n/2+i] = 2 * math.Cos(math.Pi*(float64(i)+0.5)/float64(n))
		}
	}
}

// ImageHashItem 对应 image_hash 表的一条记录（用于接口返回）。
type ImageHashItem struct {
	ID        int64  `json:"id"`
	FilePath  string `json:"filePath"`
	FileName  string `json:"fileName"`
	FileDir   string `json:"fileDir,omitempty"`
	MD5Hash   string `json:"md5Hash"`
	PHash     int64  `json:"pHash,string"`
	FileSize  int64  `json:"fileSize,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// ImageHashMatch 为查重匹配结果：包含距离与相似度。
type ImageHashMatch struct {
	ImageHashItem
	Distance   int     `json:"distance"`
	Similarity float64 `json:"similarity"`
}

// ImageHashService 提供 image_hash 表查询与 pHash 计算能力（仅用于读/比对）。
type ImageHashService struct {
	db *database.DB
}

func NewImageHashService(db *database.DB) *ImageHashService {
	return &ImageHashService{db: db}
}

// FindByMD5Hash 按 md5_hash 精确查询（命中即视为重复）。
func (s *ImageHashService) FindByMD5Hash(ctx context.Context, md5Hash string, limit int) ([]ImageHashMatch, error) {
	md5Hash = strings.TrimSpace(md5Hash)
	if md5Hash == "" {
		return nil, nil
	}
	limit = clampInt(limit, 1, 500)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_path, file_name, file_dir, md5_hash, phash, file_size, created_at
		FROM image_hash
		WHERE md5_hash = ?
		ORDER BY id DESC
		LIMIT ?`, md5Hash, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ImageHashMatch
	for rows.Next() {
		item, err := scanImageHashRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ImageHashMatch{
			ImageHashItem: item,
			Distance:      0,
			Similarity:    1,
		})
	}
	return out, rows.Err()
}

// FindSimilarByPHash 按 pHash 相似度查询：距离<=maxDistance 视为命中，并按距离升序返回。
func (s *ImageHashService) FindSimilarByPHash(ctx context.Context, phash int64, maxDistance int, limit int) ([]ImageHashMatch, error) {
	limit = clampInt(limit, 1, 500)
	maxDistance = clampInt(maxDistance, 0, phashBitLength)

	query := `
		SELECT
			id, file_path, file_name, file_dir, md5_hash, phash, file_size, created_at,
			BIT_COUNT(CAST(phash AS UNSIGNED) ^ CAST(? AS UNSIGNED)) AS distance
		FROM image_hash
		WHERE BIT_COUNT(CAST(phash AS UNSIGNED) ^ CAST(? AS UNSIGNED)) <= ?
		ORDER BY distance ASC, id DESC
		LIMIT ?`
	if s.db.Dialect().Name() == "postgres" {
		// Compute 64-bit Hamming distance using bitwise XOR on bigint and count '1's.
		// This avoids relying on non-core extensions (e.g. no custom popcount required).
		query = `
			SELECT
				id, file_path, file_name, file_dir, md5_hash, phash, file_size, created_at,
				length(replace(((phash # ?)::bit(64))::text, '0', '')) AS distance
			FROM image_hash
			WHERE length(replace(((phash # ?)::bit(64))::text, '0', '')) <= ?
			ORDER BY distance ASC, id DESC
			LIMIT ?`
	}

	rows, err := s.db.QueryContext(ctx, query, phash, phash, maxDistance, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ImageHashMatch
	for rows.Next() {
		var item ImageHashItem
		var distance int

		var fileDir sql.NullString
		var fileSize sql.NullInt64
		var createdAt time.Time

		if err := rows.Scan(
			&item.ID,
			&item.FilePath,
			&item.FileName,
			&fileDir,
			&item.MD5Hash,
			&item.PHash,
			&fileSize,
			&createdAt,
			&distance,
		); err != nil {
			return nil, err
		}

		if fileDir.Valid {
			item.FileDir = fileDir.String
		}
		if fileSize.Valid {
			item.FileSize = fileSize.Int64
		}
		item.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

		out = append(out, ImageHashMatch{
			ImageHashItem: item,
			Distance:      distance,
			Similarity:    similarityFromDistance(distance),
		})
	}
	return out, rows.Err()
}

// CalculatePHash 从上传文件计算 pHash（64位）。仅对可解码图片有效。
func (s *ImageHashService) CalculatePHash(file *multipart.FileHeader) (int64, error) {
	if file == nil {
		return 0, ErrPHashUnsupported
	}

	src, err := openMultipartFileHeaderFn(file)
	if err != nil {
		return 0, ErrPHashUnsupported
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return 0, ErrPHashUnsupported
	}

	gray, err := resizeToGrayFn(img, phashSize, phashSize)
	if err != nil {
		return 0, ErrPHashUnsupported
	}

	coeffs := dctLowFreq8x8Fn(gray)
	if len(coeffs) != phashBitLength {
		return 0, fmt.Errorf("pHash计算失败：DCT系数长度异常")
	}

	// 对齐 Python imagehash.phash：
	// - 使用 dctlowfreq 的整体 median（包含 DC 项）
	median := medianFloat64(coeffs)

	var hash uint64
	for _, v := range coeffs {
		hash <<= 1
		if v > median {
			hash |= 1
		}
	}
	return int64(hash), nil
}

func resizeToGray(src image.Image, width, height int) (*image.Gray, error) {
	if src == nil || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("无效图片")
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return nil, fmt.Errorf("无效图片尺寸")
	}

	// Pillow convert("L") rounds fixed-point RGB luminance and ignores alpha.
	// GrayModel instead truncates and composites translucent colors against black.
	gray := image.NewGray(image.Rect(0, 0, sw, sh))
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			pixel := src.At(b.Min.X+x, b.Min.Y+y)
			if c, ok := pixel.(color.Gray16); ok {
				// Pillow's I;16/I -> L conversion clips instead of dividing by 256.
				gray.Pix[y*gray.Stride+x] = uint8(clampInt(int(c.Y), 0, 255))
				continue
			}
			c := color.NRGBAModel.Convert(pixel).(color.NRGBA)
			gray.Pix[y*gray.Stride+x] = uint8((19595*uint32(c.R) + 38470*uint32(c.G) + 7471*uint32(c.B) + 32768) >> 16)
		}
	}

	// Pillow resamples horizontally, rounds/clips to 8 bits, then vertically.
	// Downsampling widens the Lanczos support by the scale factor to avoid aliasing.
	if sw != width {
		weights := pillowResizeWeights(sw, width)
		tmp := image.NewGray(image.Rect(0, 0, width, sh))
		for y := 0; y < sh; y++ {
			for x, w := range weights {
				sum := int64(1 << 21)
				for i, k := range w.values {
					sum += int64(gray.Pix[y*gray.Stride+w.start+i]) * int64(k)
				}
				tmp.Pix[y*tmp.Stride+x] = uint8(clampInt(int(sum>>22), 0, 255))
			}
		}
		gray = tmp
	}
	if sh != height {
		weights := pillowResizeWeights(sh, height)
		dst := image.NewGray(image.Rect(0, 0, width, height))
		for y, w := range weights {
			for x := 0; x < width; x++ {
				sum := int64(1 << 21)
				for i, k := range w.values {
					sum += int64(gray.Pix[(w.start+i)*gray.Stride+x]) * int64(k)
				}
				dst.Pix[y*dst.Stride+x] = uint8(clampInt(int(sum>>22), 0, 255))
			}
		}
		gray = dst
	}
	return gray, nil
}

type pillowResampleWeights struct {
	start  int
	values []int32
}

// Match Pillow's normalized, signed 22-bit resampling coefficients. Pixels
// beyond the image are excluded before normalization, not edge-replicated.
func pillowResizeWeights(input, output int) []pillowResampleWeights {
	scale := float64(input) / float64(output)
	filterScale := math.Max(scale, 1)
	support := 3 * filterScale
	weights := make([]pillowResampleWeights, output)
	for i := range weights {
		center := (float64(i) + 0.5) * scale
		start := clampInt(int(center-support+0.5), 0, input)
		end := clampInt(int(center+support+0.5), 0, input)
		values := make([]float64, end-start)
		total := 0.0
		for j := range values {
			values[j] = lanczosKernelFunc((float64(start+j)-center+0.5)/filterScale, 3)
			total += values[j]
		}
		w := pillowResampleWeights{start: start, values: make([]int32, len(values))}
		for j, value := range values {
			if total != 0 {
				value /= total
			}
			w.values[j] = int32(math.Round(value * (1 << 22)))
		}
		weights[i] = w
	}
	return weights
}

func lanczos(x, a float64) float64 {
	ax := math.Abs(x)
	if ax < 1e-12 {
		return 1
	}
	if ax >= a {
		return 0
	}
	return sinc(x) * sinc(x/a)
}

func sinc(x float64) float64 {
	if math.Abs(x) < 1e-12 {
		return 1
	}
	px := math.Pi * x
	return math.Sin(px) / px
}

func dctLowFreq8x8(img *image.Gray) []float64 {
	if img == nil || img.Bounds().Dx() != phashSize || img.Bounds().Dy() != phashSize {
		return nil
	}

	// Match imagehash: DCT-II along axis 0, then axis 1. A butterfly
	// decomposition preserves exact zero AC coefficients for constant inputs;
	// direct cosine summation produces roundoff bits in otherwise flat images.
	var columns [phashLowFreq][phashSize]float64
	var values, scratch [phashSize]float64
	for x := 0; x < phashSize; x++ {
		for y := 0; y < phashSize; y++ {
			values[y] = float64(img.Pix[y*img.Stride+x])
		}
		dctII(values[:], scratch[:])
		for u := 0; u < phashLowFreq; u++ {
			columns[u][x] = values[u]
		}
	}
	out := make([]float64, 0, phashBitLength)
	for u := 0; u < phashLowFreq; u++ {
		dctII(columns[u][:], scratch[:])
		for v := 0; v < phashLowFreq; v++ {
			// scipy's unnormalized DCT-II includes a factor of 2 per axis.
			out = append(out, 4*columns[u][v])
		}
	}
	return out
}

// Lee's recursive DCT-II for power-of-two lengths up to phashSize.
func dctII(values, scratch []float64) {
	n := len(values)
	if n == 1 {
		return
	}
	half := n / 2
	for i := 0; i < half; i++ {
		a, b := values[i], values[n-1-i]
		scratch[i] = a + b
		scratch[half+i] = (a - b) / phashDCTFactors[half+i]
	}
	dctII(scratch[:half], values[:half])
	dctII(scratch[half:], values[half:])
	for i := 0; i < half; i++ {
		values[2*i] = scratch[i]
		values[2*i+1] = scratch[half+i]
		if i+1 < half {
			values[2*i+1] += scratch[half+i+1]
		}
	}
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := make([]float64, len(values))
	copy(cp, values)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

func similarityFromDistance(distance int) float64 {
	distance = clampInt(distance, 0, phashBitLength)
	return float64(phashBitLength-distance) / float64(phashBitLength)
}

func scanImageHashRow(rows *sql.Rows) (ImageHashItem, error) {
	var item ImageHashItem
	var fileDir sql.NullString
	var fileSize sql.NullInt64
	var createdAt time.Time

	if err := rows.Scan(
		&item.ID,
		&item.FilePath,
		&item.FileName,
		&fileDir,
		&item.MD5Hash,
		&item.PHash,
		&fileSize,
		&createdAt,
	); err != nil {
		return ImageHashItem{}, err
	}

	if fileDir.Valid {
		item.FileDir = fileDir.String
	}
	if fileSize.Valid {
		item.FileSize = fileSize.Int64
	}
	item.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

	return item, nil
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
