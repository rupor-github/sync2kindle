package imgutils

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// newTestImage creates an NRGBA image of the given size filled with c.
func newTestImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestThumbnail_ExactDimensions(t *testing.T) {
	tests := []struct {
		name         string
		srcW, srcH   int
		dstW, dstH   int
		wantW, wantH int
	}{
		{"landscape to portrait", 800, 400, 200, 300, 200, 300},
		{"portrait to landscape", 400, 800, 300, 200, 300, 200},
		{"square to portrait", 600, 600, 200, 300, 200, 300},
		{"square to landscape", 600, 600, 300, 200, 300, 200},
		{"square to square", 600, 600, 200, 200, 200, 200},
		{"exact match", 200, 300, 200, 300, 200, 300},
		{"upscale", 100, 100, 200, 300, 200, 300},
		{"large downscale", 4000, 3000, 200, 300, 200, 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := newTestImage(tt.srcW, tt.srcH, color.NRGBA{R: 128, G: 64, B: 32, A: 255})
			result := Thumbnail(src, tt.dstW, tt.dstH)
			if result == nil {
				t.Fatal("Thumbnail returned nil")
			}
			gotW := result.Bounds().Dx()
			gotH := result.Bounds().Dy()
			if gotW != tt.wantW || gotH != tt.wantH {
				t.Errorf("got %dx%d, want %dx%d", gotW, gotH, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestThumbnail_InvalidInputs(t *testing.T) {
	src := newTestImage(100, 100, color.NRGBA{A: 255})

	if result := Thumbnail(src, 0, 100); result != nil {
		t.Error("expected nil for zero width")
	}
	if result := Thumbnail(src, 100, 0); result != nil {
		t.Error("expected nil for zero height")
	}
	if result := Thumbnail(src, -1, 100); result != nil {
		t.Error("expected nil for negative width")
	}
	if result := Thumbnail(src, 100, -1); result != nil {
		t.Error("expected nil for negative height")
	}
}

func TestThumbnail_ZeroSizeSource(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if result := Thumbnail(src, 100, 100); result != nil {
		t.Error("expected nil for zero-size source image")
	}
}

func TestThumbnail_OriginAtZero(t *testing.T) {
	src := newTestImage(400, 600, color.NRGBA{R: 255, A: 255})
	result := Thumbnail(src, 100, 150)
	if result == nil {
		t.Fatal("Thumbnail returned nil")
	}
	b := result.Bounds()
	if b.Min.X != 0 || b.Min.Y != 0 {
		t.Errorf("expected origin at (0,0), got (%d,%d)", b.Min.X, b.Min.Y)
	}
}

func TestThumbnail_PixelsNotBlank(t *testing.T) {
	// Verify the output actually contains image data, not just a blank allocation.
	src := newTestImage(400, 600, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	result := Thumbnail(src, 100, 150)
	if result == nil {
		t.Fatal("Thumbnail returned nil")
	}
	// Sample the center pixel -- it should be close to the source color after resampling.
	cx, cy := 50, 75
	r, g, b, a := result.At(cx, cy).RGBA()
	if a == 0 {
		t.Error("center pixel is fully transparent")
	}
	// The source is (200, 100, 50) -- after resampling with a uniform color
	// the result should be essentially identical. Allow some tolerance.
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
	if diff(r8, 200) > 5 || diff(g8, 100) > 5 || diff(b8, 50) > 5 {
		t.Errorf("unexpected center pixel color: got (%d, %d, %d), want ~(200, 100, 50)", r8, g8, b8)
	}
}

func diff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestEncodeJPEG_ValidOutput(t *testing.T) {
	src := newTestImage(200, 300, color.NRGBA{R: 128, G: 64, B: 32, A: 255})

	buf, err := EncodeJPEG(src, 75)
	if err != nil {
		t.Fatalf("EncodeJPEG failed: %v", err)
	}

	data := buf.Bytes()

	// Must start with JPEG SOI marker.
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatal("output does not start with JPEG SOI marker")
	}

	// Must contain JFIF APP0 segment (inserted by setJpegDPI).
	if len(data) < 4 || data[2] != 0xFF || data[3] != 0xE0 {
		t.Fatal("output does not contain JFIF APP0 marker at expected position")
	}

	// Must be decodable as JPEG.
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not valid JPEG: %v", err)
	}

	// Decoded dimensions must match source.
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 300 {
		t.Errorf("decoded size %dx%d, want 200x300", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncodeJPEG_JFIF_DPI(t *testing.T) {
	src := newTestImage(50, 50, color.NRGBA{A: 255})

	buf, err := EncodeJPEG(src, 90)
	if err != nil {
		t.Fatalf("EncodeJPEG failed: %v", err)
	}

	data := buf.Bytes()

	// APP0 structure after SOI (FF D8):
	//   [02-03] FF E0          - APP0 marker
	//   [04-05] 00 10          - length (16 bytes)
	//   [06-10] 4A 46 49 46 00 - "JFIF\0"
	//   [11-12] 01 02          - version 1.2
	//   [13]    01             - density units = pixels per inch
	//   [14-15] 01 2C          - X density = 300
	//   [16-17] 01 2C          - Y density = 300
	//   [18-19] 00 00          - no thumbnail

	if len(data) < 20 {
		t.Fatal("output too short to contain JFIF header")
	}

	// Check JFIF identifier at offset 6.
	if !bytes.Equal(data[6:11], []byte("JFIF\x00")) {
		t.Error("JFIF identifier not found at expected offset")
	}

	// Check density units = 1 (pixels per inch) at offset 13.
	if data[13] != 0x01 {
		t.Errorf("density units = %d, want 1 (pixels per inch)", data[13])
	}

	// Check X density = 300 (0x012C) at offset 14-15.
	xdpi := int(data[14])<<8 | int(data[15])
	if xdpi != 300 {
		t.Errorf("X density = %d, want 300", xdpi)
	}

	// Check Y density = 300 (0x012C) at offset 16-17.
	ydpi := int(data[16])<<8 | int(data[17])
	if ydpi != 300 {
		t.Errorf("Y density = %d, want 300", ydpi)
	}
}
