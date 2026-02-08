package mobi

import (
	"encoding/binary"
	"fmt"
)

const (
	// important  pdb header offsets
	uniqueIDSseed      = 68
	numberOfPdbRecords = 76

	bookLength      = 4
	bookRecordCount = 8
	firstPdbRecord  = 78

	// important rec0 offsets
	lengthOfBook      = 4
	cryptoType        = 12
	mobiHeaderBase    = 16
	mobiHeaderLength  = 20
	mobiType          = 24
	mobiVersion       = 36
	firstNonText      = 80
	titleOffset       = 84
	firstRescRecord   = 108
	firstContentIndex = 192
	lastContentIndex  = 194
	kf8FdstIndex      = 192
	fcisIndex         = 200
	flisIndex         = 208
	srcsIndex         = 224
	srcsCount         = 228
	primaryIndex      = 244
	datpIndex         = 256
	huffOffset        = 112
	huffTableOffset   = 120

	// exth records of interest
	exthASIN          = 113
	exthStartReading  = 116
	exthKF8Offset     = 121
	exthCoverOffset   = 201
	exthThumbOffset   = 202
	exthThumbnailURI  = 129
	exthCDEType       = 501
	exthCDEContentKey = 504
)

func getInt16(data []byte, ofs int) (int, error) {
	if ofs < 0 || ofs+2 > len(data) {
		return 0, fmt.Errorf("getInt16: offset %d out of bounds (len %d)", ofs, len(data))
	}
	return int(binary.BigEndian.Uint16(data[ofs:])), nil
}

func getInt32(data []byte, ofs int) (int, error) {
	if ofs < 0 || ofs+4 > len(data) {
		return 0, fmt.Errorf("getInt32: offset %d out of bounds (len %d)", ofs, len(data))
	}
	return int(binary.BigEndian.Uint32(data[ofs:])), nil
}

func getSectionAddr(data []byte, secno int) (int, int, error) {

	nsec, err := getInt16(data, numberOfPdbRecords)
	if err != nil {
		return 0, 0, fmt.Errorf("getSectionAddr: %w", err)
	}
	if secno < 0 || secno >= nsec {
		return 0, 0, fmt.Errorf("secno %d is out of range [0, %d)", secno, nsec)
	}

	var start, end int
	start, err = getInt32(data, firstPdbRecord+secno*8)
	if err != nil {
		return 0, 0, fmt.Errorf("getSectionAddr start: %w", err)
	}
	if secno == nsec-1 {
		end = len(data)
	} else {
		end, err = getInt32(data, firstPdbRecord+(secno+1)*8)
		if err != nil {
			return 0, 0, fmt.Errorf("getSectionAddr end: %w", err)
		}
	}
	if start < 0 || end < 0 || start > len(data) || end > len(data) || start > end {
		return 0, 0, fmt.Errorf("getSectionAddr: invalid section bounds [%d, %d) for data length %d", start, end, len(data))
	}
	return start, end, nil
}

func getExthParams(rec0 []byte) (int, int, int, error) {
	headerLen, err := getInt32(rec0, mobiHeaderLength)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("getExthParams headerLen: %w", err)
	}
	ebase := mobiHeaderBase + headerLen
	numItems, err := getInt32(rec0, ebase+4)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("getExthParams numItems: %w", err)
	}
	numRecords, err := getInt32(rec0, ebase+8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("getExthParams numRecords: %w", err)
	}
	return ebase, numItems, numRecords, nil
}

func readExth(rec0 []byte, recnum int) ([][]byte, error) {

	var values [][]byte

	ebase, _, enum, err := getExthParams(rec0)
	if err != nil {
		return nil, err
	}
	ebase += 12

	for enum > 0 {
		exthID, err := getInt32(rec0, ebase)
		if err != nil {
			return nil, fmt.Errorf("readExth exthID: %w", err)
		}
		exthLen, err := getInt32(rec0, ebase+4)
		if err != nil {
			return nil, fmt.Errorf("readExth exthLen: %w", err)
		}
		if exthLen < 8 {
			return nil, fmt.Errorf("readExth: invalid exth record length %d", exthLen)
		}
		if ebase+exthLen > len(rec0) {
			return nil, fmt.Errorf("readExth: exth record extends beyond data (offset %d, len %d, data len %d)", ebase, exthLen, len(rec0))
		}
		if exthID == recnum {
			// We might have multiple exths, so build a list.
			values = append(values, rec0[ebase+8:ebase+exthLen])
		}
		enum--
		ebase += exthLen
	}
	return values, nil
}

func readSection(data []byte, secno int) ([]byte, error) {
	start, end, err := getSectionAddr(data, secno)
	if err != nil {
		return nil, err
	}
	return data[start:end], nil
}
