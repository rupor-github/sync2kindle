//go:build !windows

package objects

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const prefix = "o"

type ObjectID uint32

// fmt.Stringer
func (p ObjectID) String() string {
	return fmt.Sprintf(prefix+"%X", uint32(p))
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
		s := strings.TrimPrefix(s, prefix)

		// special case to be able to run Windows test datasets on Linux
		s = strings.TrimPrefix(s, "s")

		d, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return err
		}
		*p = ObjectID(d)
	}
	return nil
}
