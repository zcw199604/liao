package app

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestMotionPhotoXMPAndOffset(t *testing.T) {
	jpeg := []byte{
		0xFF, 0xD8, // SOI
		0xFF, 0xE0, 0x00, 0x10, // APP0 length=16 (includes length bytes)
		'J', 'F', 'I', 'F', 0x00, 0x01, 0x02, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, // payload 14 bytes
		0xFF, 0xD9, // EOI
	}

	mp4 := []byte{
		0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2',
		0x00, 0x00, 0x00, 0x00, 'm', 'p', '4', '2', 'i', 's', 'o', 'm',
	}

	seg, err := buildMotionPhotoXMPAPP1Segment(int64(len(mp4)))
	if err != nil {
		t.Fatalf("buildMotionPhotoXMPAPP1Segment err=%v", err)
	}
	jpegWithXMP, err := injectJPEGSegmentAfterAPP0(jpeg, seg)
	if err != nil {
		t.Fatalf("injectJPEGSegmentAfterAPP0 err=%v", err)
	}

	if !bytes.Contains(jpegWithXMP, []byte("http://ns.adobe.com/xap/1.0/\x00")) {
		t.Fatalf("missing XMP header")
	}

	combined := append(append([]byte{}, jpegWithXMP...), mp4...)
	offset, ok := findMicroVideoOffset(combined)
	if !ok {
		t.Fatalf("missing MicroVideoOffset in XMP")
	}
	if offset != int64(len(mp4)) {
		t.Fatalf("MicroVideoOffset=%d, want %d", offset, len(mp4))
	}

	mp4Start := int64(len(combined)) - offset
	if mp4Start < 0 || mp4Start+8 > int64(len(combined)) {
		t.Fatalf("mp4Start out of range: %d", mp4Start)
	}
	if got := string(combined[mp4Start+4 : mp4Start+8]); got != "ftyp" {
		t.Fatalf("mp4 header=%q, want %q", got, "ftyp")
	}
}

func TestInjectJPEGSegmentAfterAPP0_InsertsAfterAPP0(t *testing.T) {
	jpeg := []byte{
		0xFF, 0xD8,
		0xFF, 0xE0, 0x00, 0x10,
		'J', 'F', 'I', 'F', 0x00, 0x01, 0x02, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
		0xFF, 0xD9,
	}
	seg := []byte{0xFF, 0xE1, 0x00, 0x04, 0x00, 0x00} // minimal APP1 payload=2

	out, err := injectJPEGSegmentAfterAPP0(jpeg, seg)
	if err != nil {
		t.Fatalf("inject err=%v", err)
	}

	wantInsertAt := 2 + 2 + 0x10
	if wantInsertAt+2 > len(out) {
		t.Fatalf("out too short")
	}
	if out[wantInsertAt] != 0xFF || out[wantInsertAt+1] != 0xE1 {
		t.Fatalf("segment not inserted after APP0, got marker=%02x%02x at %d", out[wantInsertAt], out[wantInsertAt+1], wantInsertAt)
	}
}

func findMicroVideoOffset(data []byte) (int64, bool) {
	const key = "GCamera:MicroVideoOffset=\""
	s := string(data)
	i := strings.Index(s, key)
	if i < 0 {
		return 0, false
	}
	i += len(key)
	j := strings.IndexByte(s[i:], '"')
	if j < 0 {
		return 0, false
	}
	raw := s[i : i+j]
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func TestBuildMotionPhotoJPG_MiuiShape(t *testing.T) {
	dir := t.TempDir()
	stillPath := filepath.Join(dir, "still.jpg")
	mp4Path := filepath.Join(dir, "motion.mp4")
	outPath := filepath.Join(dir, "out.jpg")

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode jpeg err=%v", err)
	}
	if err := os.WriteFile(stillPath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write still err=%v", err)
	}

	mp4 := []byte{
		0x00, 0x00, 0x00, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm',
		0x00, 0x00, 0x02, 0x00, 'i', 's', 'o', 'm', 'i', 's', 'o', '2', 'a', 'v', 'c', '1', 'm', 'p', '4', '1',
	}
	if err := os.WriteFile(mp4Path, mp4, 0o644); err != nil {
		t.Fatalf("write mp4 err=%v", err)
	}

	if err := buildMotionPhotoJPG(stillPath, mp4Path, outPath); err != nil {
		t.Fatalf("buildMotionPhotoJPG err=%v", err)
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out err=%v", err)
	}

	offset, ok := findMicroVideoOffset(out)
	if !ok {
		t.Fatalf("missing MicroVideoOffset in XMP")
	}
	if offset != int64(len(mp4)) {
		t.Fatalf("MicroVideoOffset=%d, want %d", offset, len(mp4))
	}

	mp4Start := int64(len(out)) - offset
	if mp4Start < 0 || mp4Start+8 > int64(len(out)) {
		t.Fatalf("mp4Start out of range: %d", mp4Start)
	}
	if got := string(out[mp4Start+4 : mp4Start+8]); got != "ftyp" {
		t.Fatalf("mp4 header=%q, want %q", got, "ftyp")
	}

	// EOI -> gap(24) -> MP4
	eoiPos := bytes.LastIndex(out[:mp4Start], []byte{0xFF, 0xD9})
	if eoiPos < 0 {
		t.Fatalf("missing EOI before mp4Start")
	}
	gap := out[eoiPos+2 : mp4Start]
	if !bytes.Equal(gap, motionPhotoEOIGapBytes) {
		t.Fatalf("gap mismatch: got len=%d", len(gap))
	}

	// Header segments order: APP1(Exif) -> APP1(XMP) -> APP0(JFIF)
	if len(out) < 2 || out[0] != 0xFF || out[1] != 0xD8 {
		t.Fatalf("out not jpeg")
	}
	i := 2
	readSeg := func() (marker byte, payload []byte) {
		if i+4 > len(out) || out[i] != 0xFF {
			t.Fatalf("bad marker at %d", i)
		}
		for i < len(out) && out[i] == 0xFF {
			i++
		}
		if i >= len(out) {
			t.Fatalf("truncated marker")
		}
		marker = out[i]
		i++
		if i+2 > len(out) {
			t.Fatalf("truncated length")
		}
		n := int(out[i])<<8 | int(out[i+1])
		i += 2
		if n < 2 || i+(n-2) > len(out) {
			t.Fatalf("bad seg length=%d", n)
		}
		payload = out[i : i+(n-2)]
		i += n - 2
		return marker, payload
	}

	m, p := readSeg()
	if m != 0xE1 || !bytes.HasPrefix(p, []byte("Exif\x00\x00")) {
		t.Fatalf("want first APP1(Exif), got marker=%02x payloadPrefix=%q", m, p[:min(12, len(p))])
	}
	m, p = readSeg()
	if m != 0xE1 || !bytes.HasPrefix(p, []byte("http://ns.adobe.com/xap/1.0/\x00")) {
		t.Fatalf("want second APP1(XMP), got marker=%02x", m)
	}
	m, p = readSeg()
	if m != 0xE0 || !bytes.HasPrefix(p, []byte("JFIF\x00")) {
		t.Fatalf("want third APP0(JFIF), got marker=%02x", m)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
