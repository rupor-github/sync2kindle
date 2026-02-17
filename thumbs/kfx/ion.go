package kfx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/amazon-ion/ion-go/ion"
)

const (
	largestKnownSymbol = 851
)

var (
	ionBVM            = []byte{0xE0, 1, 0, 0xEA} // binary version marker
	sharedSymbolTable = createSST(largestKnownSymbol)
)

// Actual names for symbols could be obtained by looking at EpubToKFXConverter-4.0.jar from Kindle Previewer 3
// with enum of interest in class file "com.amazon.kaf.c/B.class" presently.
func createSST(maxID uint64) ion.SharedSymbolTable {
	symbols := make([]string, 0, maxID)
	for i := len(ion.V1SystemSymbolTable.Symbols()) + 1; i <= len(ion.V1SystemSymbolTable.Symbols())+int(maxID); i++ {
		symbols = append(symbols, fmt.Sprintf("$%d", i))
	}
	return ion.NewSharedSymbolTable("YJ_symbols", 10, symbols)
}

func createProlog() []byte {
	buf := bytes.Buffer{}
	if err := ion.NewBinaryWriter(&buf, sharedSymbolTable).Finish(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func readDataFrom(r io.Reader, v any) error {
	if err := binary.Read(r, binary.LittleEndian, v); err != nil {
		return err
	}
	if val, ok := v.(interface{ validate() error }); ok {
		return val.validate()
	}
	return nil
}

func readData(data []byte, v any) (int, error) {
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, v); err != nil {
		return 0, err
	}
	if val, ok := v.(interface{ validate() error }); ok {
		return len(data) - r.Len(), val.validate()
	}
	return len(data) - r.Len(), nil
}

func decodeIon(prolog, data []byte, v any) error {
	if len(data) < len(ionBVM) {
		return fmt.Errorf("ion data too short: got %d bytes, need at least %d", len(data), len(ionBVM))
	}
	if err := ion.Unmarshal(append(prolog, data[len(ionBVM):]...), v, sharedSymbolTable); err != nil {
		return err
	}
	if val, ok := v.(interface{ validate() error }); ok {
		return val.validate()
	}
	return nil
}

func decodeSymbolTable(data []byte) (ion.SymbolTable, error) {
	r := ion.NewReaderCat(bytes.NewReader(data), ion.NewCatalog(sharedSymbolTable))
	r.Next() // we are not interested in the actual values and in most cases this will return false anyways
	if err := r.Err(); err != nil {
		return nil, err
	}
	return r.SymbolTable(), nil
}
