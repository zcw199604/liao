package app

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math/bits"
	"os"
	"strconv"
	"testing"
)

// Golden values come from the same Python imagehash/Pillow implementation used
// by image_dedup.py. See testdata/phash/generate.py for reproducible inputs.
func TestImageHashService_PythonCompatibility(t *testing.T) {
	data, err := os.ReadFile("testdata/phash/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Mode, Pattern, PHash string
		Width, Height              int
		Gray                       []byte
		Encoded                    []byte
		MaxDistance                int
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			bounds := image.Rect(0, 0, tc.Width, tc.Height)
			rgb := image.NewNRGBA(bounds)
			gray := image.NewGray(bounds)
			gray16 := image.NewGray16(bounds)
			for y := 0; y < tc.Height; y++ {
				for x := 0; x < tc.Width; x++ {
					n := uint32(y*tc.Width+x+1) * 2654435761
					n = (n ^ (n >> 16)) * 2246822519
					r, g, b, a := uint8(n), uint8(n>>8), uint8(n>>16), uint8(n>>24)
					switch tc.Pattern {
					case "constant":
						r, g, b = 70, 130, 190
					case "gradient":
						r, g, b = uint8(x*3), uint8(y*3), uint8((x+y)*2)
					case "checkerboard":
						r = uint8(((x/8 + y/8) % 2) * 255)
						g, b = r, r
					}
					if tc.Mode != "RGBA" {
						a = 255
					}
					rgb.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
					gray.SetGray(x, y, color.Gray{Y: r})
					gray16.SetGray16(x, y, color.Gray16{Y: uint16(n & 1023)})
				}
			}
			var src image.Image = rgb
			if tc.Mode == "L" {
				src = gray
			}
			if tc.Mode == "I;16" {
				src = gray16
			}
			if len(tc.Encoded) > 0 {
				decoded, _, err := image.Decode(bytes.NewReader(tc.Encoded))
				if err != nil {
					t.Fatal(err)
				}
				src = decoded
			}
			resized, err := resizeToGray(src, 32, 32)
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.Gray) > 0 && !bytes.Equal(resized.Pix, tc.Gray) {
				count := 0
				for i, v := range resized.Pix {
					if v != tc.Gray[i] {
						count++
					}
				}
				t.Errorf("Pillow grayscale/resize mismatch: %d of 1024 pixels", count)
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, src); err != nil {
				t.Fatal(err)
			}
			content := buf.Bytes()
			if len(tc.Encoded) > 0 {
				content = tc.Encoded
			}
			_, fh := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "sample", "application/octet-stream", content, nil)
			got, err := (&ImageHashService{}).CalculatePHash(fh)
			if err != nil {
				t.Fatal(err)
			}
			want, err := strconv.ParseUint(tc.PHash, 16, 64)
			if err != nil {
				t.Fatal(err)
			}
			// Only the JPEG decoder fixture allows its measured 2-bit difference.
			// The companion decoded RGB fixture requires exact algorithm parity.
			if distance := bits.OnesCount64(uint64(got) ^ want); distance > tc.MaxDistance {
				t.Errorf("pHash=%016x, Python=%016x, distance=%d (max %d)", uint64(got), want, distance, tc.MaxDistance)
			}
		})
	}
}
