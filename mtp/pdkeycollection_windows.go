//go:build mtp && windows

package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

func (v *IPortableDeviceKeyCollection) Add(key *PropertyKey) error {
	hr, _, _ := syscall.SyscallN(v.VTable().Add, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}
