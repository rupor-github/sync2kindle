package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"

	"s2k/objects"
)

func (v *IPortableDeviceProperties) GetValues(oid objects.ObjectID, keys *IPortableDeviceKeyCollection) (ipdv *IPortableDeviceValues, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().GetValues, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&oid[0])), uintptr(unsafe.Pointer(keys)), uintptr(unsafe.Pointer(&ipdv)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}
