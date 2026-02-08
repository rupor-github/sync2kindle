// Much of the knowledge of the mobi internals comes from KindleUnpack project
// copyrighted under GPL v3. Visit https://github.com/kevinhendricks/KindleUnpack
// for more details.
package mobi

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"go.uber.org/zap"

	"s2k/thumbs/imgutils"
)

// Reader - mobi thumbnail extractor.
type Reader struct {
	width, height int
	fname         string
	//
	acr, asin, cdetype, cdekey,
	thumbnail []byte
}

func (r *Reader) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("asin: %s, cdekey: %s, cdetype: %s, acr: %s", string(r.asin), string(r.cdekey), string(r.cdetype), string(r.acr))
}

// NewReader returns pointer to Reader with parsed mobi file.
func NewReader(fname string, w, h int, log *zap.Logger) (*Reader, error) {
	var r *Reader

	l := log.Named("mobi-reader")
	l.Debug("MOBI parse starting",
		zap.String("fname", fname), zap.Int("width", w), zap.Int("height", h))
	defer func(start time.Time) {
		l.Debug("MOBI parse finished", zap.Stringer("metadata", r), zap.Duration("elapsed", time.Since(start)))
	}(time.Now())

	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	r = &Reader{fname: fname, width: w, height: h}
	return r, r.extractThumbnail(data)
}

// SaveResult saves extracted thumbnail if any to the requested location.
func (r *Reader) SaveResult(dir string) (string, error) {

	if len(r.thumbnail) == 0 {
		return "", nil
	}

	asin := string(r.asin)
	if len(r.cdekey) > 0 {
		asin = string(r.cdekey)
	}
	if len(asin) == 0 {
		return "", nil
	}

	fileName := "thumbnail_" + asin + "_" + string(r.cdetype) + "_portrait.jpg"
	fullName := filepath.Join(dir, fileName)

	if err := os.WriteFile(fullName, r.thumbnail, 0644); err != nil {
		return "", err
	}
	return fileName, nil
}

func (r *Reader) extractThumbnail(data []byte) error {
	rec0, err := readSection(data, 0)
	if err != nil {
		return fmt.Errorf("unable to read section 0: %w", err)
	}

	ct, err := getInt16(rec0, cryptoType)
	if err != nil {
		return fmt.Errorf("unable to read crypto type: %w", err)
	}
	if ct != 0 {
		return errors.New("encrypted books are not supported")
	}

	var (
		kf8    int
		kfrec0 []byte
	)

	kf8off, err := readExth(rec0, exthKF8Offset)
	if err != nil {
		return fmt.Errorf("unable to read KF8 offset: %w", err)
	}
	if len(kf8off) > 0 {
		// only pay attention to first KF8 offfset - there should only be one
		if kf8, err = getInt32(kf8off[0], 0); err == nil && kf8 >= 0 {
			kfrec0, err = readSection(data, kf8)
			if err != nil {
				return fmt.Errorf("unable to read KF8 section: %w", err)
			}
		}
	}
	combo := len(kfrec0) > 0 && kf8 >= 0

	// save ACR
	const alphabet = `- ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789`
	if len(data) >= 32 {
		r.acr = bytes.Map(func(sym rune) rune {
			if sym == 0 {
				return -1
			}
			if strings.ContainsRune(alphabet, sym) {
				return sym
			}
			return '_'
		}, data[0:32])
	}

	exth, err := readExth(rec0, exthASIN)
	if err != nil {
		return fmt.Errorf("unable to read ASIN: %w", err)
	}
	if len(exth) > 0 {
		r.asin = exth[0]
	}
	exth, err = readExth(rec0, exthCDEType)
	if err != nil {
		return fmt.Errorf("unable to read CDE type: %w", err)
	}
	if len(exth) > 0 {
		r.cdetype = exth[0]
	}
	exth, err = readExth(rec0, exthCDEContentKey)
	if err != nil {
		return fmt.Errorf("unable to read CDE content key: %w", err)
	}
	if len(exth) > 0 {
		r.cdekey = exth[0]
	}

	// always prefer data from KF8
	if combo {
		exth, err = readExth(kfrec0, exthASIN)
		if err != nil {
			return fmt.Errorf("unable to read KF8 ASIN: %w", err)
		}
		if len(exth) > 0 {
			r.asin = exth[0]
		}
		exth, err = readExth(kfrec0, exthCDEType)
		if err != nil {
			return fmt.Errorf("unable to read KF8 CDE type: %w", err)
		}
		if len(exth) > 0 {
			r.cdetype = exth[0]
		}
		exth, err = readExth(kfrec0, exthCDEContentKey)
		if err != nil {
			return fmt.Errorf("unable to read KF8 CDE content key: %w", err)
		}
		if len(exth) > 0 {
			r.cdekey = exth[0]
		}
	}

	if !bytes.Equal(r.cdetype, []byte("EBOK")) {
		// bail out early - we are only interested in non-personal books
		return nil
	}

	firstimage, err := getInt32(rec0, firstRescRecord)
	if err != nil {
		return fmt.Errorf("unable to read first resource record: %w", err)
	}
	exthCover, err := readExth(rec0, exthCoverOffset)
	if err != nil {
		return fmt.Errorf("unable to read cover offset: %w", err)
	}
	coverIndex := -1
	if len(exthCover) > 0 {
		ci, err := getInt32(exthCover[0], 0)
		if err != nil {
			return fmt.Errorf("unable to read cover index: %w", err)
		}
		coverIndex = ci + firstimage
	}

	exthThumb, err := readExth(rec0, exthThumbOffset)
	if err != nil {
		return fmt.Errorf("unable to read thumb offset: %w", err)
	}
	thumbIndex := -1
	if len(exthThumb) > 0 {
		ti, err := getInt32(exthThumb[0], 0)
		if err != nil {
			return fmt.Errorf("unable to read thumb index: %w", err)
		}
		thumbIndex = ti + firstimage
	}

	if coverIndex < 0 {
		return nil
	}

	var (
		img  image.Image
		w, h = 0, 0
	)
	if thumbIndex >= 0 {
		thumb, err := readSection(data, thumbIndex)
		if err != nil {
			return fmt.Errorf("unable to read thumbnail section: %w", err)
		}
		if img, _, err = image.Decode(bytes.NewReader(thumb)); err != nil {
			return fmt.Errorf("unable to decode extracted thumbnail: %w", err)
		}
		w, h = img.Bounds().Dx(), img.Bounds().Dy()
	}
	var buf = new(bytes.Buffer)
	if img != nil && w > r.width && h > r.height {
		// thumbnail is big enough, use it as is but always convert to JPEG
		if err := imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(75)); err != nil {
			// NOTE: old code ignored this and tried to extract from the cover instead
			return fmt.Errorf("unable to encode extracted thumbnail: %w", err)
		}
	} else {
		// recreate thumbnail from cover image if possible
		thumb, err := readSection(data, coverIndex)
		if err != nil {
			return fmt.Errorf("unable to read cover section: %w", err)
		}
		if img, _, err = image.Decode(bytes.NewReader(thumb)); err != nil {
			return fmt.Errorf("unable to decode extracted thumbnail: %w", err)
		}
		imgthumb := imaging.Thumbnail(img, r.width, r.height, imaging.Lanczos)
		if imgthumb == nil {
			return errors.New("unable to resize extracted cover")
		}
		if err := imaging.Encode(buf, imgthumb, imaging.JPEG, imaging.JPEGQuality(75)); err != nil {
			return fmt.Errorf("unable to encode produced thumbnail: %w", err)
		}
	}
	buf, _ = imgutils.SetJpegDPI(buf, imgutils.DpiPxPerInch, 300, 300)
	r.thumbnail = buf.Bytes()
	return nil
}
