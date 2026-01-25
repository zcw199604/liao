package app

import (
	"bytes"
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
