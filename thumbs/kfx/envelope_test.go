package kfx_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amazon-ion/ion-go/ion"
	"github.com/ulikunitz/xz/lzma"
	"go.uber.org/zap/zaptest"

	"s2k/thumbs/kfx"
	"s2k/thumbs/kfx/internal/envelope"
)

func TestReaderEnvelopeThumbnails(t *testing.T) {
	inner := syntheticBook(t, 0, 0, 0, 0)
	wantName, wantJPEG := readThumbnail(t, inner)
	if wantName != "thumbnail_B000000001_EBOK_portrait.jpg" {
		t.Fatalf("unexpected thumbnail name: %s", wantName)
	}
	if !bytes.HasPrefix(wantJPEG, []byte{0xff, 0xd8, 0xff, 0xe0}) {
		t.Fatal("thumbnail is not a JFIF JPEG")
	}
	img, err := jpeg.Decode(bytes.NewReader(wantJPEG))
	if err != nil || img.Bounds().Dx() != 330 || img.Bounds().Dy() != 470 {
		t.Fatalf("invalid thumbnail dimensions or JPEG: %v", err)
	}
	for _, compressed := range []bool{false, true} {
		for _, pageSize := range []int{4096, 16384, 32768, 65536} {
			t.Run(fmt.Sprintf("compressed=%t/page=%d", compressed, pageSize), func(t *testing.T) {
				wrapped := wrapBook(t, inner, compressed, pageSize)
				e, err := envelope.Inspect(t.Context(), wrapped)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := e.Decode(t.Context())
				if err != nil || !bytes.Equal(decoded, inner) {
					t.Fatalf("envelope changed inner book: %v", err)
				}
				gotName, gotJPEG := readThumbnail(t, wrapped)
				if gotName != wantName || !bytes.Equal(gotJPEG, wantJPEG) {
					t.Fatal("wrapped book produced different thumbnail name or JPEG")
				}
			})
		}
	}
}

func readThumbnail(t *testing.T, data []byte) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	// Deliberately use the same extension: detection must use the file's bytes.
	fname := filepath.Join(dir, "book.kfx")
	if err := os.WriteFile(fname, data, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := kfx.NewReader(fname, 330, 470, zaptest.NewLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	name, err := r.SaveResult(dir)
	if err != nil || name == "" {
		t.Fatalf("missing thumbnail: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return name, data
}

func TestReaderRejectsUnsupportedInner(t *testing.T) {
	for _, test := range []struct {
		name                         string
		contCompression, contDRM     int
		entityCompression, entityDRM int
		want                         string
	}{
		{"container-compression", 1, 0, 0, 0, "container compression"},
		{"container-drm", 0, 1, 0, 0, "container DRM"},
		{"entity-compression", 0, 0, 1, 0, "entity compression"},
		{"entity-drm", 0, 0, 0, 1, "entity DRM"},
	} {
		t.Run(test.name, func(t *testing.T) {
			inner := syntheticBook(t, test.contCompression, test.contDRM, test.entityCompression, test.entityDRM)
			for _, wrapped := range []bool{false, true} {
				t.Run(fmt.Sprintf("wrapped=%t", wrapped), func(t *testing.T) {
					data := inner
					if wrapped {
						data = wrapBook(t, inner, true, 4096)
					}
					fname := filepath.Join(t.TempDir(), "book.kfx")
					if err := os.WriteFile(fname, data, 0600); err != nil {
						t.Fatal(err)
					}
					if _, err := kfx.NewReader(fname, 330, 470, zaptest.NewLogger(t)); err == nil || !strings.Contains(err.Error(), test.want) {
						t.Fatalf("expected %q rejection, got %v", test.want, err)
					}
				})
			}
		})
	}
}

func TestReaderRejectsMalformedEnvelope(t *testing.T) {
	valid := wrapBook(t, syntheticBook(t, 0, 0, 0, 0), true, 4096)
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"truncated", valid[:len(valid)-1]},
		{"encrypted-page", bytes.ReplaceAll(valid, []byte{0x81, 0xc6}, []byte{0x81, 0xc5})},
		{"corrupt-lzma", bytes.ReplaceAll(valid, []byte{0, 0x5d, 0, 0, 0x80, 0}, []byte{1, 0x5d, 0, 0, 0x80, 0})},
	} {
		t.Run(test.name, func(t *testing.T) {
			fname := filepath.Join(t.TempDir(), "book.kfx")
			if err := os.WriteFile(fname, test.data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := kfx.NewReader(fname, 330, 470, zaptest.NewLogger(t)); err == nil {
				t.Fatal("accepted malformed envelope")
			}
		})
	}
}

// syntheticBook builds a minimal, public-domain book with metadata, a JPEG cover,
// and an unused padding entity so all supported page sizes cross a page boundary.
func syntheticBook(tb testing.TB, contCompression, contDRM, entityCompression, entityDRM int) []byte {
	tb.Helper()
	symbols := make([]string, 851)
	for j := range symbols {
		symbols[j] = fmt.Sprintf("$%d", j+10)
	}
	shared := ion.NewSharedSymbolTable("YJ_symbols", 10, symbols)
	local := ion.NewLocalSymbolTable([]ion.SharedSymbolTable{shared}, []string{"cover", "cover-data", "padding"})
	marshal := func(v any) []byte {
		tb.Helper()
		data, err := ion.MarshalBinaryLST(v, local)
		if err != nil {
			tb.Fatal(err)
		}
		return data
	}
	// A fixed-LST writer emits its prolog on the first value, not on Finish.
	// Remove the one-byte integer zero to retain only the document symbol table.
	prolog := marshal(0)
	prolog = prolog[:len(prolog)-1]
	var cover bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 33, 47))
	for y := range 47 {
		for x := range 33 {
			img.SetRGBA(x, y, color.RGBA{R: byte(x * 7), G: byte(y * 5), B: 120, A: 255})
		}
	}
	if err := jpeg.Encode(&cover, img, nil); err != nil {
		tb.Fatal(err)
	}
	metadata := map[string]any{"$491": []any{map[string]any{
		"$495": "kindle_title_metadata",
		"$258": []any{
			map[string]any{"$492": "ASIN", "$307": "B000000001"},
			map[string]any{"$492": "cde_content_type", "$307": "EBOK"},
			map[string]any{"$492": "cover_image", "$307": "cover"},
		},
	}}}
	var entities, index bytes.Buffer
	for _, entity := range []struct {
		name string
		typ  uint32
		data []byte
	}{
		{"", 490, marshal(metadata)},
		{"cover", 164, marshal(map[string]any{"$165": "cover-data"})},
		{"cover-data", 417, cover.Bytes()},
		{"padding", 417, bytes.Repeat([]byte{0x5a}, 70000)},
	} {
		info := marshal(map[string]any{"$410": entityCompression, "$411": entityDRM})
		header := struct {
			Signature [4]byte
			Version   uint16
			Size      uint32
		}{[4]byte{'E', 'N', 'T', 'Y'}, 1, uint32(10 + len(info))}
		var id uint64
		if entity.name != "" {
			var ok bool
			id, ok = local.FindByName(entity.name)
			if !ok {
				tb.Fatalf("missing symbol %s", entity.name)
			}
			id -= ion.V1SystemSymbolTable.MaxID()
		}
		writeBinary(tb, &index, struct {
			ID, Type     uint32
			Offset, Size uint64
		}{uint32(id), entity.typ, uint64(entities.Len()), uint64(header.Size) + uint64(len(entity.data))})
		writeBinary(tb, &entities, header)
		entities.Write(info)
		entities.Write(entity.data)
	}
	const symbolsOffset = 18 + 512
	indexOffset := symbolsOffset + len(prolog)
	info := marshal(map[string]any{
		"$410": contCompression, "$411": contDRM,
		"$413": indexOffset, "$414": index.Len(), "$415": symbolsOffset, "$416": len(prolog),
	})
	if len(info) > 512 {
		tb.Fatal("container info exceeded reserved region")
	}
	var output bytes.Buffer
	writeBinary(tb, &output, struct {
		Signature                  [4]byte
		Version                    uint16
		Size, InfoOffset, InfoSize uint32
	}{[4]byte{'C', 'O', 'N', 'T'}, 2, uint32(indexOffset + index.Len()), 18, uint32(len(info))})
	output.Write(info)
	output.Write(make([]byte, symbolsOffset-output.Len()))
	output.Write(prolog)
	output.Write(index.Bytes())
	output.Write(entities.Bytes())
	return output.Bytes()
}

func writeBinary(tb testing.TB, out *bytes.Buffer, value any) {
	tb.Helper()
	if err := binary.Write(out, binary.LittleEndian, value); err != nil {
		tb.Fatal(err)
	}
}

// wrapBook is a test-only writer adapted from fb2cng (GPL-3.0). Production code
// deliberately contains no encoder. Independent reference fixtures test the decoder
// separately, so these integration tests do not rely only on a matching writer.
func wrapBook(tb testing.TB, inner []byte, compressed bool, pageSize int) []byte {
	tb.Helper()
	shared := ion.NewSharedSymbolTable("ProtectedData", 1, []string{
		"com.amazon.drm.Envelope@1.0", "com.amazon.drm.EnvelopeMetadata@1.0", "size", "page_size",
		"encryption_key", "encryption_transformation", "encryption_voucher", "signing_key",
		"signing_algorithm", "signing_voucher", "com.amazon.drm.EncryptedPage@1.0", "cipher_text",
		"cipher_iv", "com.amazon.drm.Signature@1.0", "data", "com.amazon.drm.EnvelopeIndexTable@1.0",
		"length", "offset", "algorithm", "encoded", "encryption_algorithm", "hashing_algorithm",
		"expires", "format", "id", "lock_parameters", "strategy", "com.amazon.drm.Key@1.0",
		"com.amazon.drm.KeySet@1.0", "com.amazon.drm.PIDv3@1.0", "com.amazon.drm.PlainTextPage@1.0",
		"com.amazon.drm.PlainText@1.0", "com.amazon.drm.PrivateKey@1.0", "com.amazon.drm.PublicKey@1.0",
		"com.amazon.drm.SecretKey@1.0", "com.amazon.drm.Voucher@1.0", "public_key", "private_key",
		"com.amazon.drm.KeyPair@1.0", "com.amazon.drm.ProtectedData@1.0", "doctype",
		"com.amazon.drm.EnvelopeIndexTableOffset@1.0", "enddoc", "license_type", "license", "watermark",
		"key", "value", "com.amazon.drm.License@1.0", "category", "metadata", "categorized_metadata",
		"com.amazon.drm.CategorizedMetadata@1.0", "com.amazon.drm.VoucherEnvelope@1.0", "mac", "voucher",
		"com.amazon.drm.ProtectedData@2.0", "com.amazon.drm.Envelope@2.0", "com.amazon.drm.EnvelopeMetadata@2.0",
		"com.amazon.drm.EncryptedPage@2.0", "com.amazon.drm.PlainText@2.0", "compression_algorithm",
		"com.amazon.drm.Compressed@1.0", "page_index_table",
	})
	datagram := func(write func(ion.Writer)) []byte {
		tb.Helper()
		var buf bytes.Buffer
		w := ion.NewBinaryWriter(&buf, shared)
		write(w)
		// Ion writers retain earlier errors; Finish checks all writes below.
		if err := w.Finish(); err != nil {
			tb.Fatal(err)
		}
		return buf.Bytes()
	}
	marker := func(annotation string) []byte {
		return datagram(func(w ion.Writer) {
			_ = w.Annotation(ion.NewSymbolTokenFromString(annotation))
			_ = w.WriteSymbolFromString("com.amazon.drm.ProtectedData@2.0")
		})
	}
	prefix := marker("doctype")
	frame := datagram(func(w ion.Writer) {
		_ = w.Annotation(ion.NewSymbolTokenFromString("com.amazon.drm.Envelope@2.0"))
		_ = w.BeginList()
		_ = w.Annotation(ion.NewSymbolTokenFromString("com.amazon.drm.EnvelopeMetadata@2.0"))
		_ = w.BeginStruct()
		_ = w.FieldName(ion.NewSymbolTokenFromString("size"))
		_ = w.WriteInt(int64(len(inner)))
		_ = w.FieldName(ion.NewSymbolTokenFromString("page_size"))
		_ = w.WriteInt(int64(pageSize))
		if compressed {
			_ = w.FieldName(ion.NewSymbolTokenFromString("compression_algorithm"))
			_ = w.WriteString("LZMA")
		}
		_ = w.EndStruct()
		for offset := 0; offset < len(inner); offset += pageSize {
			page := inner[offset:min(offset+pageSize, len(inner))]
			if compressed {
				var buf bytes.Buffer
				buf.WriteByte(0)
				encoder, err := (lzma.WriterConfig{
					Properties: &lzma.Properties{LC: 3, LP: 0, PB: 2},
					DictCap:    max(4096, len(page)), SizeInHeader: true, Size: int64(len(page)), EOSMarker: true,
				}).NewWriter(&buf)
				if err != nil {
					tb.Fatal(err)
				}
				_, writeErr := encoder.Write(page)
				if err := errors.Join(writeErr, encoder.Close()); err != nil {
					tb.Fatal(err)
				}
				page = buf.Bytes()
				binary.LittleEndian.PutUint32(page[2:6], 8*1024*1024)
			}
			_ = w.Annotation(ion.NewSymbolTokenFromString("com.amazon.drm.PlainText@2.0"))
			_ = w.BeginStruct()
			_ = w.FieldName(ion.NewSymbolTokenFromString("data"))
			if compressed {
				_ = w.Annotation(ion.NewSymbolTokenFromString("com.amazon.drm.Compressed@1.0"))
			}
			_ = w.WriteBlob(page)
			_ = w.EndStruct()
		}
		_ = w.EndList()
	})
	prolog := datagram(func(ion.Writer) {})
	end := marker("enddoc")[len(prolog):]
	indexOffset := uint64(len(prefix) + len(frame) + len(end))
	index := datagram(func(w ion.Writer) {
		_ = w.Annotation(ion.NewSymbolTokenFromString("com.amazon.drm.EnvelopeIndexTable@1.0"))
		_ = w.BeginList()
		_ = w.BeginList()
		for _, n := range []uint64{uint64(len(prefix)), uint64(len(frame)), 0, uint64(len(inner))} {
			_ = w.WriteUint(n)
		}
		_ = w.EndList()
		_ = w.EndList()
	})
	output := []byte(envelope.Magic)
	for _, section := range [][]byte{prefix, frame, end, index, {0x28}} {
		output = append(output, section...)
	}
	return binary.BigEndian.AppendUint64(output, indexOffset)
}
