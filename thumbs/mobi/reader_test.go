package mobi_test

import (
	"bytes"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"

	"s2k/thumbs/mobi"
)

const testFile = "../../testdata/_Test.azw3"

func TestNewReader(t *testing.T) {
	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("test fixture not available: %v", err)
	}

	log := zaptest.NewLogger(t)

	const thumbW, thumbH = 330, 470

	r, err := mobi.NewReader(testFile, thumbW, thumbH, log)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	dir := t.TempDir()
	name, err := r.SaveResult(dir)
	if err != nil {
		t.Fatalf("SaveResult failed: %v", err)
	}
	if name == "" {
		t.Fatal("SaveResult returned empty filename — no thumbnail was extracted")
	}

	t.Logf("saved thumbnail: %s", name)

	// Verify the file is a valid JPEG with expected dimensions.
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("unable to read saved thumbnail: %v", err)
	}

	// SOI marker.
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatal("saved file does not start with JPEG SOI marker")
	}

	// JFIF APP0 segment must be present (Kindle requirement).
	if len(data) < 4 || data[2] != 0xFF || data[3] != 0xE0 {
		t.Fatal("saved file does not contain JFIF APP0 segment")
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("saved file is not valid JPEG: %v", err)
	}

	gotW, gotH := img.Bounds().Dx(), img.Bounds().Dy()
	if gotW != thumbW || gotH != thumbH {
		t.Errorf("thumbnail dimensions %dx%d, want %dx%d", gotW, gotH, thumbW, thumbH)
	}
}
