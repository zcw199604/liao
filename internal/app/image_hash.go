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
)

// ErrPHashUnsupported 表示该文件无法计算 pHash（非图片或解码失败等）。
var ErrPHashUnsupported = errors.New("无法计算pHash（仅支持可解码的图片格式）")

const (
	phashSize      = 32
	phashLowFreq   = 8
	phashBitLength = 64
)

var (
	phashCosTable [phashLowFreq][phashSize]float64
)

func init() {
	n := float64(phashSize)
	for u := 0; u < phashLowFreq; u++ {
		for x := 0; x < phashSize; x++ {
			phashCosTable[u][x] = math.Cos(math.Pi * (float64(x) + 0.5) * float64(u) / n)
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
	db *sql.DB
}

func NewImageHashService(db *sql.DB) *ImageHashService {
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

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, file_path, file_name, file_dir, md5_hash, phash, file_size, created_at,
			BIT_COUNT(CAST(phash AS UNSIGNED) ^ CAST(? AS UNSIGNED)) AS distance
		FROM image_hash
		WHERE BIT_COUNT(CAST(phash AS UNSIGNED) ^ CAST(? AS UNSIGNED)) <= ?
		ORDER BY distance ASC, id DESC
		LIMIT ?`, phash, phash, maxDistance, limit)
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

	src, err := file.Open()
	if err != nil {
		return 0, ErrPHashUnsupported
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return 0, ErrPHashUnsupported
	}

	gray, err := resizeToGray(img, phashSize, phashSize)
	if err != nil {
		return 0, ErrPHashUnsupported
	}

	coeffs := dctLowFreq8x8(gray)
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

	dst := image.NewGray(image.Rect(0, 0, width, height))

	// 尽量对齐 PIL 的 ANTIALIAS(LANCZOS) 重采样，避免与 imagehash.phash 生成的 pHash 差异过大。
	// 说明：这里实现了灰度域的 Lanczos3 缩放（目标尺寸固定 32x32，开销可控）。
	const a = 3.0
	scaleX := float64(sw) / float64(width)
	scaleY := float64(sh) / float64(height)
	halfKernel := int(math.Ceil(a))

	for y := 0; y < height; y++ {
		srcY := (float64(y)+0.5)*scaleY - 0.5
		iy := int(math.Floor(srcY))
		rowStart := y * dst.Stride
		for x := 0; x < width; x++ {
			srcX := (float64(x)+0.5)*scaleX - 0.5
			ix := int(math.Floor(srcX))

			sum := 0.0
			sumW := 0.0

			for sy := iy - halfKernel + 1; sy <= iy+halfKernel; sy++ {
				wy := lanczos(srcY-float64(sy), a)
				if wy == 0 {
					continue
				}
				py := clampInt(sy, 0, sh-1)
				for sx := ix - halfKernel + 1; sx <= ix+halfKernel; sx++ {
					wx := lanczos(srcX-float64(sx), a)
					if wx == 0 {
						continue
					}
					px := clampInt(sx, 0, sw-1)

					gy := color.GrayModel.Convert(src.At(b.Min.X+px, b.Min.Y+py)).(color.Gray)
					w := wx * wy
					sum += float64(gy.Y) * w
					sumW += w
				}
			}

			if sumW == 0 {
				gy := color.GrayModel.Convert(src.At(b.Min.X+clampInt(ix, 0, sw-1), b.Min.Y+clampInt(iy, 0, sh-1))).(color.Gray)
				dst.Pix[rowStart+x] = gy.Y
				continue
			}

			v := sum / sumW
			if v < 0 {
				v = 0
			} else if v > 255 {
				v = 255
			}
			dst.Pix[rowStart+x] = uint8(math.Round(v))
		}
	}

	return dst, nil
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

	// 仅计算低频 8x8（其余不参与 pHash）
	// 对齐 Python imagehash.phash 的 DCT 顺序：
	// - scipy.fftpack.dct(..., axis=0) 再 axis=1
	// - 因此 lowfreq 的第一维是 y 方向频率，第二维是 x 方向频率
	out := make([]float64, 0, phashBitLength)
	for u := 0; u < phashLowFreq; u++ {
		for v := 0; v < phashLowFreq; v++ {
			sum := 0.0
			for y := 0; y < phashSize; y++ {
				cosUy := phashCosTable[u][y]
				rowStart := y * img.Stride
				for x := 0; x < phashSize; x++ {
					sum += float64(img.Pix[rowStart+x]) * cosUy * phashCosTable[v][x]
				}
			}
			out = append(out, sum)
		}
	}
	return out
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
