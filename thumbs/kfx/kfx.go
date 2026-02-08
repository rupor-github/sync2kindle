package kfx

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"unsafe"

	"github.com/amazon-ion/ion-go/ion"
)

const (
	symBookMetadata     = 490
	symExternalResource = 164
	symRawMedia         = 417
)

type containerHeader struct {
	Signature  [4]byte
	Version    uint16
	Size       uint32
	InfoOffset uint32
	InfoSize   uint32
}

func (c *containerHeader) validate() error {
	const (
		maxContVersion  = 2
		minContainerLen = 18 // Minimum CONT header length
	)

	if !bytes.Equal(c.Signature[:], []byte("CONT")) {
		return fmt.Errorf("wrong signature for KFX container: % X", c.Signature[:])
	}
	if c.Version > maxContVersion {
		return fmt.Errorf("unsupported KFX container version: %d", c.Version)
	}
	if uintptr(c.Size) < minContainerLen {
		return fmt.Errorf("invalid KFX container header size: %d", c.Size)
	}
	return nil
}

type containerInfo struct {
	ContainerId         string `ion:"$409"`
	CompressionType     int    `ion:"$410"`
	DRMScheme           int    `ion:"$411"`
	ChunkSize           int    `ion:"$412"`
	IndexTabOffset      int    `ion:"$413"`
	IndexTabLength      int    `ion:"$414"`
	DocSymOffset        int    `ion:"$415"`
	DocSymLength        int    `ion:"$416"`
	FCapabilitiesOffset int    `ion:"$594"`
	FCapabilitiesLength int    `ion:"$595"`
}

func (c *containerInfo) validate() error {
	const (
		defaultCompressionType = 0
		defaultDRMScheme       = 0
	)

	if c.CompressionType != defaultCompressionType {
		return fmt.Errorf("unsupported KFX container compression type: %d", c.CompressionType)
	}
	if c.DRMScheme != defaultDRMScheme {
		return fmt.Errorf("unsupported KFX container DRM: %d", c.DRMScheme)
	}
	return nil
}

type indexTableEntry struct {
	NumID, NumType uint32
	Offset, Size   uint64
}

func (e *indexTableEntry) readFrom(r io.Reader, start uint32, limit int, st ion.SymbolTable) error {
	*e = indexTableEntry{}

	if err := readDataFrom(r, e); err != nil {
		return err
	}
	if uint64(start)+e.Offset > uint64(limit) {
		return fmt.Errorf("entity is out of bounds: %d + %d > %d", uint64(start)+e.Offset, e.Size, limit)
	}
	if _, ok := st.FindByID(uint64(e.NumID)); !ok {
		return fmt.Errorf("entity ID not found in the symbol table: %d", e.NumID)
	}
	if _, ok := st.FindByID(uint64(e.NumType)); !ok {
		return fmt.Errorf("entity type not found in the symbol table: %d", e.NumType)
	}
	return nil
}

type entityHeader struct {
	Signature [4]byte
	Version   uint16
	Size      uint32
}

func (e *entityHeader) validate() error {
	const (
		maxEntityVersion = 1
	)

	if !bytes.Equal(e.Signature[:], []byte("ENTY")) {
		return fmt.Errorf("wrong signature for KFX entity: % X", e.Signature[:])
	}
	if e.Version > maxEntityVersion {
		return fmt.Errorf("unsupported KFX entity version: %d", e.Version)
	}
	if uintptr(e.Size) < unsafe.Sizeof(*e) {
		return fmt.Errorf("invalid KFX entity header size: %d", e.Size)
	}
	return nil
}

type entityInfo struct {
	CompressionType int `ion:"$410"`
	DRMScheme       int `ion:"$411"`
}

func (e *entityInfo) validate() error {
	const (
		defaultCompressionType = 0
		defaultDRMScheme       = 0
	)

	if e.CompressionType != defaultCompressionType {
		return fmt.Errorf("unsupported KFX entity compression type: %d", e.CompressionType)
	}
	if e.DRMScheme != defaultDRMScheme {
		return fmt.Errorf("unsupported KFX entity DRM: %d", e.DRMScheme)
	}
	return nil
}

type (
	entity struct {
		id, idType uint32
		data       []byte
	}
	entitySet struct {
		st        ion.SymbolTable
		fragments map[uint32][]*entity
	}
)

func newEntitySet(st ion.SymbolTable) *entitySet {
	return &entitySet{
		st:        st,
		fragments: make(map[uint32][]*entity),
	}
}

func (s *entitySet) addEntity(id, idType uint32, data []byte) {
	s.fragments[idType] = append(s.fragments[idType],
		&entity{
			id:     id,
			idType: idType,
			data:   data,
		})
}

func (s *entitySet) getAllOfType(idType uint32) []*entity {
	entities, exists := s.fragments[idType]
	if !exists {
		return nil
	}
	return entities
}

func (s *entitySet) getByName(name string, idType uint32) (*entity, error) {
	entities, exists := s.fragments[idType]
	if !exists {
		return nil, fmt.Errorf("no entities of type '%d' found", idType)
	}
	id, ok := s.st.FindByName(name)
	if !ok {
		return nil, fmt.Errorf("symbol '%s' not found in the symbol table", name)
	}
	id -= ion.V1SystemSymbolTable.MaxID()

	var index int
	if index = slices.IndexFunc(entities, func(e *entity) bool {
		return uint64(e.id) == id
	}); index == -1 {
		return nil, fmt.Errorf("entity fragment '%s' of type '%d' not found", name, idType)
	}
	return entities[index], nil
}

type (
	property struct {
		Key   string `ion:"$492"`
		Value any    `ion:"$307"`
	}
	properties struct {
		Category string     `ion:"$495"`
		Metadata []property `ion:"$258"`
	}
	bookMetadata struct {
		CategorizedMetadata []properties `ion:"$491"`
	}
	coverResource struct {
		Location string `ion:"$165"`
		// ResourceName any    `ion:"$175"` // ion.SymbolToken
		// Format       any    `ion:"$161"` // ion.SymbolToken
		// Width        int    `ion:"$422"`
		// Height       int    `ion:"$423"`
		// Mime         string `ion:"$162"`
	}
)
