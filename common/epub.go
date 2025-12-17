package common

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"strings"
)

type epubContainer struct {
	Rootfile struct {
		Path string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubPackage struct {
	Metadata struct {
		Title string `xml:"http://purl.org/dc/elements/1.1/ title"`
	} `xml:"metadata"`
}

// GetEPUBTitle extracts the title from an EPUB file
// Returns the title if successful, empty string otherwise
func GetEPUBTitle(epubPath string) (string, error) {
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Find and read the container.xml to get the content.opf path
	var contentPath string
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", err
			}

			var container epubContainer
			if err := xml.Unmarshal(data, &container); err != nil {
				return "", err
			}
			contentPath = container.Rootfile.Path
			break
		}
	}

	if contentPath == "" {
		return "", nil
	}

	// Read the content.opf to get metadata
	// Need to handle both forward and backslashes since some EPUB files use backslashes
	contentPathAlt := strings.ReplaceAll(contentPath, "/", "\\")
	for _, f := range r.File {
		if f.Name == contentPath || f.Name == contentPathAlt {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", err
			}

			var pkg epubPackage
			if err := xml.Unmarshal(data, &pkg); err != nil {
				return "", err
			}
			return strings.TrimSpace(pkg.Metadata.Title), nil
		}
	}

	return "", nil
}
