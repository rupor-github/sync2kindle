//go:build mtp && windows

package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

func (v *IPortableDeviceCapabilities) GetFunctionalCategories() (ipdp *IPortableDevicePropVariantCollection, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().GetFunctionalCategories, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&ipdp)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}
