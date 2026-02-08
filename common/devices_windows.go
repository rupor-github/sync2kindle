package common

import (
	"regexp"
	"strconv"
	"strings"
)

// NOTE: we keep zero terminator in the slice to avoid additional UTF16 to UTF16Ptr conversion
type PnPDeviceID []uint16

var (
	reVendorID  = regexp.MustCompile(`(?i)USB[#&]VID_([0-9A-F]+)&PID_[0-9A-F]+`)
	reProductID = regexp.MustCompile(`(?i)USB[#&]VID_[0-9A-F]+&PID_([0-9A-F]+)`)
	reSerial    = regexp.MustCompile(`(?i)USB[#&]VID_[0-9A-F]+&PID_[0-9A-F]+#(.+)#.+`)
)

// VendorID returns Vendor ID or zero if it cannot parse PnP device ID string.
// NOTE: this is not what Microsoft recommends doing, descriptor is supposed to be opaque,
// but it is working on all Windows versions so far and alternative is bulky at best.
// https://learn.microsoft.com/en-us/windows-hardware/drivers/install/standard-usb-identifiers
func (p PnPDeviceID) VendorID() int {
	matches := reVendorID.FindStringSubmatch(p.String())
	if len(matches) != 2 {
		return 0
	}
	var (
		id  int64
		err error
	)
	if id, err = strconv.ParseInt(matches[1], 16, 32); err != nil {
		return 0
	}
	return int(id)
}

// ProductID returns Product IDs or zero if it cannot parse PnP device ID string.
// NOTE: this is not what Microsoft recommends doing, descriptor is supposed to be opaque,
// but it is working on all Windows versions so far and alternative is bulky at best.
// https://learn.microsoft.com/en-us/windows-hardware/drivers/install/standard-usb-identifiers
func (p PnPDeviceID) ProductID() int {
	matches := reProductID.FindStringSubmatch(p.String())
	if len(matches) != 2 {
		return 0
	}
	var (
		id  int64
		err error
	)
	if id, err = strconv.ParseInt(matches[1], 16, 32); err != nil {
		return 0
	}
	return int(id)
}

// Serial returns SN for device or empty string if it cannot parse PnP device ID string.
// NOTE: this is not what Microsoft recommends doing, descriptor is supposed to be opaque,
// but it is working on all Windows versions so far and alternative is bulky at best.
func (p PnPDeviceID) Serial() string {
	matches := reSerial.FindStringSubmatch(p.String())
	if len(matches) != 2 {
		return ""
	}
	return strings.ToUpper(matches[1])
}

func (p PnPDeviceID) String() string {
	return UTF16ToString(p)
}
