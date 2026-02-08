//go:build !windows

package common

import (
	"fmt"
)

type PnPDeviceID struct {
	vid, pid, bcd int
	serial        string
}

func NewPnPDeviceID(vid, pid, bcd int, serial string) *PnPDeviceID {
	return &PnPDeviceID{vid: vid, pid: pid, bcd: bcd, serial: serial}
}

func (id *PnPDeviceID) Empty() bool {
	return id == nil || (id.vid == -1 && id.pid == -1)
}

func (id *PnPDeviceID) VendorID() int {
	return id.vid
}

func (id PnPDeviceID) ProductID() int {
	return id.pid
}

func (id PnPDeviceID) Serial() string {
	return id.serial
}

func (id PnPDeviceID) String() string {
	return fmt.Sprintf("USB&VID_%04X&PID_%04X#%s#%02d.%d%d", id.vid, id.pid, id.serial, id.bcd>>8, id.bcd&0xF0, id.bcd&0x0F)
}
