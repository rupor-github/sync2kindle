package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"

	"s2k/common"
)

func (v *IPortableDevice) Open(id common.PnPDeviceID, ci *IPortableDeviceValues) error {
	hr, _, _ := syscall.SyscallN(v.VTable().Open, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), uintptr(unsafe.Pointer(ci)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDevice) Content() (ipdc *IPortableDeviceContent, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().Content, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&ipdc)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDevice) Capabilities() (ipdc *IPortableDeviceCapabilities, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().Capabilities, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&ipdc)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}
