package imgutils

import (
	"bytes"
	"encoding/binary"
)

// This is specific to go - when encoding jpeg standard encoder does not create JFIF APP0 segment and Kindle does not like it.

// JpegDPIType specifyes type of the DPI units
type JpegDPIType uint8

// DPI units type values
const (
	DpiNoUnits JpegDPIType = iota
	DpiPxPerInch
	DpiPxPerCm
)

var (
	marker = []byte{0xFF, 0xE0}                               // APP0 segment marker
	jfif   = []byte{0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x02} // jfif + version
)

// SetJpegDPI creates JFIF APP0 with provided DPI if segment is missing in image.
func SetJpegDPI(buf *bytes.Buffer, dpit JpegDPIType, xdensity, ydensity int16) (*bytes.Buffer, bool) {

	data := buf.Bytes()

	// Need at least SOI (2 bytes) + marker (2 bytes) to check
	if len(data) < 4 {
		return buf, false
	}

	// If JFIF APP0 segment is there - do not do anything
	if bytes.Equal(data[2:4], marker) {
		return buf, false
	}

	var newbuf = new(bytes.Buffer)

	newbuf.Write(data[:2])
	newbuf.Write(marker)
	binary.Write(newbuf, binary.BigEndian, uint16(0x10)) // length
	newbuf.Write(jfif)
	binary.Write(newbuf, binary.BigEndian, uint8(dpit))
	binary.Write(newbuf, binary.BigEndian, uint16(xdensity))
	binary.Write(newbuf, binary.BigEndian, uint16(ydensity))
	binary.Write(newbuf, binary.BigEndian, uint16(0)) // no thumbnail segment
	newbuf.Write(data[2:])

	return newbuf, true
}
