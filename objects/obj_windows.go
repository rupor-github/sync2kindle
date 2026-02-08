package objects

import (
	"encoding/json"
	"unicode/utf16"

	"s2k/common"
)

// NOTE: we keep zero terminator in the slice to avoid additional UTF16 to UTF16Ptr conversion
// NOTE: in stringer and marshaler we are relying on ObjectID to be well behaved Unicode sequence.
type ObjectID []uint16

// fmt.Stringer
func (p ObjectID) String() string {
	return common.UTF16ToString(p)
}

// For convinience marshal ObjectID as a string, rather than []uint16
func (p *ObjectID) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// For convinience unmarshal ObjectID from a string, rather than []uint16
func (p *ObjectID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if len(s) > 0 {
		*p = utf16.Encode([]rune(s))
		// make sure it's zero-terminated
		*p = append(*p, 0)
	}
	return nil
}
