// Much of the knowledge of the KPF/KDF/KFX internals comes from Calibre's KFX
// Conversion Input/Output Plugins created by John Howell <jhowell@acm.org> and
// copyrighted under GPL v3. Visit https://www.mobileread.com/forums for more
// details.
package kfx

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/disintegration/imaging"
	"go.uber.org/zap"

	"s2k/thumbs/imgutils"
)

type Reader struct {
	width, height int
	fname         string
	//
	asin, cdetype, assetID, bookID string
	thumbnail                      []byte
}

func (r *Reader) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("asin: %s, cdetype: %s, asset_id: %s, book_id: %s", r.asin, r.cdetype, r.assetID, r.bookID)
}

func NewReader(fname string, w, h int, log *zap.Logger) (*Reader, error) {
	var r *Reader

	l := log.Named("kfx-reader")
	l.Debug("KFX parse starting",
		zap.String("fname", fname), zap.Int("width", w), zap.Int("height", h))
	defer func(start time.Time) {
		l.Debug("KFX parse finished", zap.Stringer("metadata", r), zap.Duration("elapsed", time.Since(start)))
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

	if len(r.thumbnail) == 0 || len(r.asin) == 0 {
		return "", nil
	}

	fileName := "thumbnail_" + r.asin + "_" + r.cdetype + "_portrait.jpg"
	fullName := filepath.Join(dir, fileName)

	if err := os.WriteFile(fullName, r.thumbnail, 0644); err != nil {
		return "", err
	}
	return fileName, nil
}

func (r *Reader) extractThumbnail(data []byte) error {

	var contHeader containerHeader
	if _, err := readData(data, &contHeader); err != nil {
		return err
	}

	var contInfo containerInfo
	if err := decodeIon(createProlog(), data[contHeader.InfoOffset:contHeader.InfoOffset+contHeader.InfoSize], &contInfo); err != nil {
		return err
	}

	if contInfo.DocSymLength == 0 {
		return errors.New("no document symbols found, unsupported KFX type")
	}

	lstProlog := data[contInfo.DocSymOffset : contInfo.DocSymOffset+contInfo.DocSymLength]
	docSymbols, err := decodeSymbolTable(lstProlog)
	if err != nil {
		return fmt.Errorf("unable to read KFX document symbols: %w", err)
	}

	entities := newEntitySet(docSymbols)

	var tableEntry indexTableEntry
	indexTableReader := bytes.NewReader(data[contInfo.IndexTabOffset : contInfo.IndexTabOffset+contInfo.IndexTabLength])
	for {
		if err := tableEntry.readFrom(indexTableReader, contHeader.Size, len(data), docSymbols); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("unable to read entity table: %w", err)
		}

		var entyHeader entityHeader
		enty := data[uint64(contHeader.Size)+tableEntry.Offset : uint64(contHeader.Size)+tableEntry.Offset+tableEntry.Size]
		count, err := readData(enty, &entyHeader)
		if err != nil {
			return err
		}

		var entyInfo entityInfo
		if err := decodeIon(lstProlog, enty[count:entyHeader.Size], &entyInfo); err != nil {
			return err
		}
		entities.addEntity(tableEntry.NumID, tableEntry.NumType, enty[entyHeader.Size:])
	}

	metaEntys := entities.getAllOfType(symBookMetadata)
	if len(metaEntys) == 0 {
		// NOTE: KFX input plugin expects case like this and attempts to get to
		// $258 (Metadata) directly, processing it differently in this case. I
		// am ignoring this for now as I never seen case like this.
		return errors.New("no book metadata found")
	} else if len(metaEntys) != 1 {
		return fmt.Errorf("ambiguous metadata, expected single book metadata entity, got %d", len(metaEntys))
	}

	var bmd bookMetadata
	if err := decodeIon(lstProlog, metaEntys[0].data, &bmd); err != nil {
		return err
	}

	// NOTE: we expect a single metadata entity with category "kindle_title_metadata"
	var coverResourceName string
	if index := slices.IndexFunc(bmd.CategorizedMetadata, func(p properties) bool {
		if p.Category == "kindle_title_metadata" {
			for _, prop := range p.Metadata {
				switch prop.Key {
				case "ASIN":
					r.asin = prop.Value.(string)
				case "cde_content_type":
					r.cdetype = prop.Value.(string)
				case "asset_id":
					r.assetID = prop.Value.(string)
				case "book_id":
					r.bookID = prop.Value.(string)
				case "cover_image":
					coverResourceName = prop.Value.(string)
				}
			}
			return true
		}
		return false
	}); index == -1 {
		return errors.New("no kindle book metadata found")
	}

	if r.cdetype != "EBOK" || len(r.asin) == 0 || len(coverResourceName) == 0 {
		// bail out early - we are only interested in non-personal books
		return nil
	}

	coverEnty, err := entities.getByName(coverResourceName, symExternalResource)
	if err != nil {
		return fmt.Errorf("unable to get cover image entity: %w", err)
	}

	var cr coverResource
	if err := decodeIon(lstProlog, coverEnty.data, &cr); err != nil {
		return err
	}

	coverImgEnty, err := entities.getByName(cr.Location, symRawMedia)
	if err != nil {
		return fmt.Errorf("unable to get cover image data: %w", err)
	}

	var img image.Image
	if img, _, err = image.Decode(bytes.NewReader(coverImgEnty.data)); err != nil {
		return fmt.Errorf("unable to decode extracted cover: %w", err)
	}
	imgthumb := imaging.Thumbnail(img, r.width, r.height, imaging.Lanczos)
	if imgthumb == nil {
		return errors.New("unable to resize extracted cover")
	}

	var buf = new(bytes.Buffer)
	if err := imaging.Encode(buf, imgthumb, imaging.JPEG, imaging.JPEGQuality(75)); err != nil {
		return fmt.Errorf("unable to encode produced thumbnail: %w", err)
	}
	buf, _ = imgutils.SetJpegDPI(buf, imgutils.DpiPxPerInch, 300, 300)
	r.thumbnail = buf.Bytes()

	return nil
}
