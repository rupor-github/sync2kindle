package common

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestGetEPUBTitle(t *testing.T) {
	// Create a temporary EPUB file for testing
	tmpDir := t.TempDir()
	epubPath := filepath.Join(tmpDir, "test.epub")

	// Create a minimal EPUB structure
	zipFile, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("Failed to create test EPUB: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)

	// Add mimetype file
	mimetypeFile, _ := w.Create("mimetype")
	mimetypeFile.Write([]byte("application/epub+zip"))

	// Add container.xml
	containerFile, _ := w.Create("META-INF/container.xml")
	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	containerFile.Write([]byte(containerXML))

	// Add content.opf with metadata
	contentFile, _ := w.Create("content.opf")
	contentOPF := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book Title</dc:title>
    <dc:creator>Test Author</dc:creator>
  </metadata>
</package>`
	contentFile.Write([]byte(contentOPF))

	w.Close()

	// Test the function
	title, err := GetEPUBTitle(epubPath)
	if err != nil {
		t.Fatalf("GetEPUBTitle failed: %v", err)
	}

	expectedTitle := "Test Book Title"
	if title != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, title)
	}
}

func TestGetEPUBTitle_NonExistent(t *testing.T) {
	_, err := GetEPUBTitle("/nonexistent/file.epub")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGetEPUBTitle_InvalidZip(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.epub")

	// Create a non-zip file
	if err := os.WriteFile(invalidPath, []byte("not a zip file"), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	_, err := GetEPUBTitle(invalidPath)
	if err == nil {
		t.Error("Expected error for invalid zip file, got nil")
	}
}
