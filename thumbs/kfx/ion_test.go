package kfx

import (
	"bytes"
	"testing"

	"github.com/amazon-ion/ion-go/ion"
)

func TestDecodeIonPreservesInput(t *testing.T) {
	data, err := ion.MarshalBinary(entityInfo{}, sharedSymbolTable)
	if err != nil {
		t.Fatal(err)
	}
	prolog := createProlog()
	// Simulate a symbol table within a book, followed by unrelated container data.
	book := append(bytes.Clone(prolog), bytes.Repeat([]byte{0xa5}, len(data))...)
	want := bytes.Clone(book)
	var info entityInfo
	if err := decodeIon(book[:len(prolog)], data, &info); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(book, want) {
		t.Fatal("decoding Ion overwrote adjacent book data")
	}
}
