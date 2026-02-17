package imgutils

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	"math"

	xdraw "golang.org/x/image/draw"
)

// Thumbnail scales src to fill the width x height bounding box (preserving
// aspect ratio) and then center-crops to exactly width x height.
// This replicates the behavior of imaging.Thumbnail with Lanczos resampling.
// Returns nil if width or height is non-positive.
func Thumbnail(src image.Image, width, height int) *image.NRGBA {
	if width <= 0 || height <= 0 {
		return nil
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil
	}

	// Determine scale factor so the image fills the target box.
	// We scale by the larger ratio so neither dimension is smaller than target.
	scaleW := float64(width) / float64(srcW)
	scaleH := float64(height) / float64(srcH)
	scale := math.Max(scaleW, scaleH)

	resW := int(math.Round(float64(srcW) * scale))
	resH := int(math.Round(float64(srcH) * scale))
	if resW < width {
		resW = width
	}
	if resH < height {
		resH = height
	}

	// Resize using CatmullRom (high quality, comparable to Lanczos).
	resized := image.NewNRGBA(image.Rect(0, 0, resW, resH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), src, srcBounds, xdraw.Over, nil)

	// Center-crop to exact target dimensions.
	if resW == width && resH == height {
		return resized
	}
	x0 := (resW - width) / 2
	y0 := (resH - height) / 2
	cropped := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(cropped, cropped.Bounds(), resized, image.Pt(x0, y0), draw.Src)
	return cropped
}

// EncodeJPEG encodes img as JPEG with the given quality into a buffer
// and applies the JFIF DPI fixup for Kindle compatibility.
func EncodeJPEG(img image.Image, quality int) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	buf, _ = setJpegDPI(buf, dpiPxPerInch, 300, 300)
	return buf, nil
}
