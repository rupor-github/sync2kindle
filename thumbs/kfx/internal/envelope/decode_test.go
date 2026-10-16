package envelope

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/amazon-ion/ion-go/ion"
)

func referenceFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	text, err := os.ReadFile("testdata/" + name + ".hex")
	if err != nil {
		tb.Fatal(err)
	}
	data, err := hex.DecodeString(strings.TrimSpace(string(text)))
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

func TestReferenceFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name        string
		repetitions int
		pageSize    uint64
	}{
		{"plaintext", 0, 4096}, {"lzma", 210, 4096}, {"lzma64", 3000, 65536},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			data := referenceFixture(t, fixture.name)
			if !IsEnvelope(data) {
				t.Fatal("reference envelope not detected")
			}
			e, err := Inspect(t.Context(), data)
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.Decode(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			want := append(bytes.Repeat([]byte("Привет, 世界!\x00"), fixture.repetitions), []byte("CONT\x00reference\xff\x80")...)
			if e.pageSize != fixture.pageSize || e.compressed != (fixture.repetitions != 0) {
				t.Fatal("incorrect reference metadata")
			}
			if !bytes.Equal(got, want) {
				t.Fatal("reference fixture did not decode byte-exactly")
			}
		})
	}
}

func TestRejectMalformed(t *testing.T) {
	valid := referenceFixture(t, "lzma")
	for j := range len(valid) {
		if _, err := Inspect(t.Context(), valid[:j]); err == nil {
			t.Fatalf("accepted truncated input at %d", j)
		}
	}
	for _, test := range []struct {
		name    string
		pattern []byte
		replace []byte
	}{
		{"frame-prolog", []byte{0xe0, 1, 0, 0xea}, []byte{0, 1, 0, 0xea}},
		{"filter", []byte{0, 0x5d, 0, 0, 0x80, 0}, []byte{1, 0x5d, 0, 0, 0x80, 0}},
		{"properties", []byte{0, 0x5d, 0, 0, 0x80, 0}, []byte{0, 0x5c, 0, 0, 0x80, 0}},
		{"dictionary", []byte{0, 0x5d, 0, 0, 0x80, 0}, []byte{0, 0x5d, 0, 0, 0x40, 0}},
		{"unknown-size", []byte{0, 0x5d, 0, 0, 0x80, 0, 0, 0x10}, []byte{0, 0x5d, 0, 0, 0x80, 0, 0xff, 0xff}},
		{"encrypted-page", []byte{0x81, 0xc6}, []byte{0x81, 0xc5}},
		{"wrong-page-size", []byte{0x8d, 0x22, 0x10, 0}, []byte{0x8d, 0x22, 0x20, 0}},
		{"algorithm", []byte("LZMA"), []byte("XZMA")},
		{"compression-annotation", []byte{0x81, 0xc8}, []byte{0x81, 0xc7}},
		{"symbols", []byte("ProtectedData"), []byte("XrotectedData")},
		{"trailer", valid[len(valid)-9:], append([]byte{0x21}, valid[len(valid)-8:]...)},
		{"index-overflow", valid[len(valid)-8:], bytes.Repeat([]byte{0xff}, 8)},
		{"decoded-size-mismatch", []byte{0x22, 0x12, 0x1c}, []byte{0x22, 0xff, 0xff}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !bytes.Contains(valid, test.pattern) {
				t.Fatal("mutation pattern not found")
			}
			b := bytes.ReplaceAll(valid, test.pattern, test.replace)
			if _, err := Inspect(t.Context(), b); err == nil {
				t.Fatal("accepted malformed envelope")
			}
		})
	}
}

func TestRejectMalformedPages(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Envelope)
	}{
		{"truncated", func(e *Envelope) { e.pages[0] = e.pages[0][:14] }},
		{"trailing-bytes", func(e *Envelope) { e.pages[0] = append(e.pages[0], 0) }},
		{"corrupt-stream", func(e *Envelope) { e.pages[0][14] ^= 0xff }},
		{"missing-eos", func(e *Envelope) { e.pages[0] = e.pages[0][:len(e.pages[0])-1] }},
		{"decoded-size", func(e *Envelope) { e.size-- }},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, err := Inspect(t.Context(), referenceFixture(t, "lzma"))
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(e)
			if _, err := e.Decode(t.Context()); err == nil {
				t.Fatal("accepted malformed page")
			}
		})
	}
}

func TestRejectIndexBounds(t *testing.T) {
	valid := referenceFixture(t, "plaintext")
	indexOffset := binary.BigEndian.Uint64(valid[len(valid)-8:])
	row, err := readIndex(t.Context(), valid[uint64(len(Magic))+indexOffset:len(valid)-9])
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		column int
		value  uint64
	}{
		{"zero-frame-offset", 0, 0},
		{"frame-offset-overflow", 0, 1 << 62},
		{"zero-frame-length", 1, 0},
		{"frame-length-overflow", 1, 1 << 62},
		{"nonzero-decoded-offset", 2, 1},
		{"empty-stream", 3, 0},
		{"decoded-size-limit", 3, maxDecodedSize + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := row
			mutated[test.column] = test.value
			var index bytes.Buffer
			w := ion.NewBinaryWriter(&index, symbolTable())
			// Finish checks retained writer errors.
			_ = w.Annotation(ion.NewSymbolTokenFromString(indexType))
			_ = w.BeginList()
			_ = w.BeginList()
			for _, n := range mutated {
				_ = w.WriteUint(n)
			}
			_ = w.EndList()
			_ = w.EndList()
			if err := w.Finish(); err != nil {
				t.Fatal(err)
			}
			b := bytes.Clone(valid[:uint64(len(Magic))+indexOffset])
			b = append(b, index.Bytes()...)
			b = append(b, valid[len(valid)-9:]...)
			if _, err := Inspect(t.Context(), b); err == nil {
				t.Fatal("accepted invalid index range or decoded size")
			}
		})
	}
}

func TestCancellation(t *testing.T) {
	data := referenceFixture(t, "lzma")
	e, err := Inspect(t.Context(), data)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Inspect(ctx, data); !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect: expected cancellation, got %v", err)
	}
	if _, err := e.Decode(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Decode: expected cancellation, got %v", err)
	}
}

func TestRejectNullImports(t *testing.T) {
	valid := referenceFixture(t, "plaintext")
	prologEnd := bytes.Index(valid[len(Magic):], []byte{0xe4, 0x81, 0xb2})
	if prologEnd < 0 {
		t.Fatal("missing doctype")
	}
	frameOffset := bytes.Index(valid[len(Magic)+4:], []byte{0xe0, 1, 0, 0xea}) + len(Magic) + 4
	indexOffset := int(binary.BigEndian.Uint64(valid[len(valid)-8:])) + len(Magic)
	for _, field := range []string{"name", "version", "max_id"} {
		for _, offset := range []int{len(Magic), frameOffset, indexOffset} {
			t.Run(fmt.Sprintf("%s/offset=%d", field, offset), func(t *testing.T) {
				var buf bytes.Buffer
				w := ion.NewBinaryWriter(&buf)
				_ = w.Annotation(ion.NewSymbolTokenFromString("$ion_symbol_table"))
				_ = w.BeginStruct()
				_ = w.FieldName(ion.NewSymbolTokenFromString("imports"))
				_ = w.BeginList()
				_ = w.BeginStruct()
				for _, name := range []string{"name", "version", "max_id"} {
					_ = w.FieldName(ion.NewSymbolTokenFromString(name))
					switch {
					case name == field:
						typ := ion.IntType
						if name == "name" {
							typ = ion.StringType
						}
						_ = w.WriteNullType(typ)
					case name == "name":
						_ = w.WriteString("ProtectedData")
					case name == "version":
						_ = w.WriteInt(1)
					default:
						_ = w.WriteInt(64)
					}
				}
				_ = w.EndStruct()
				_ = w.EndList()
				_ = w.EndStruct()
				if err := w.Finish(); err != nil {
					t.Fatal(err)
				}
				if buf.Len() > prologEnd {
					t.Fatal("replacement prolog exceeds original length")
				}
				b := bytes.Clone(valid)
				clear(b[offset : offset+prologEnd])
				copy(b[offset:], buf.Bytes())
				if _, err := Inspect(t.Context(), b); err == nil || !strings.Contains(err.Error(), "null") {
					t.Fatalf("expected null import error, got %v", err)
				}
			})
		}
	}
}

func TestIonBounds(t *testing.T) {
	deep := []byte{0x20}
	for range 20 {
		deep = append([]byte{0xbe, byte(len(deep)) | 0x80}, deep...)
	}
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"huge-scalar", []byte{0xe0, 1, 0, 0xea, 0xae, 0x7f, 0x7f, 0x7f, 0x7f, 0xff}},
		{"deep-nesting", append([]byte{0xe0, 1, 0, 0xea}, deep...)},
		{"varuint-overflow", append([]byte{0xe0, 1, 0, 0xea, 0xae}, bytes.Repeat([]byte{0x7f}, 12)...)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newReader(t.Context(), test.data); err == nil {
				t.Fatal("accepted malformed Ion")
			}
		})
	}
}

func FuzzInspect(f *testing.F) {
	for _, name := range []string{"plaintext", "lzma", "lzma64"} {
		data := referenceFixture(f, name)
		f.Add(data)
		f.Add(data[:len(data)-1])
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		const maxFuzzSize = 128 * 1024
		if len(data) > maxFuzzSize {
			t.Skip()
		}
		e, err := Inspect(t.Context(), data)
		if err != nil || e.size > maxFuzzSize {
			return
		}
		decoded, err := e.Decode(t.Context())
		if err == nil && uint64(len(decoded)) != e.size {
			t.Fatal("decoded length differs from declared size")
		}
	})
}
