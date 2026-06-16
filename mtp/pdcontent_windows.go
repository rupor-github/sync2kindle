//go:build mtp && windows

package mtp

import (
	"fmt"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"

	"s2k/common"
	"s2k/objects"
)

func (v *IPortableDeviceContent) Properties() (ipdp *IPortableDeviceProperties, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().Properties, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&ipdp)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDeviceContent) EnumObjects(flags uint32, parent objects.ObjectID, filter *IPortableDeviceValues) (ipe *IEnumPortableDeviceObjectIDs, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().EnumObjects, uintptr(unsafe.Pointer(v)),
		uintptr(flags), uintptr(unsafe.Pointer(&parent[0])), uintptr(unsafe.Pointer(filter)),
		uintptr(unsafe.Pointer(&ipe)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDeviceContent) CreateObjectWithPropertiesOnly(pValues *IPortableDeviceValues) (objects.ObjectID, error) {
	var ptr *uint16
	hr, _, _ := syscall.SyscallN(v.VTable().CreateObjectWithPropertiesOnly, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(pValues)), uintptr(unsafe.Pointer(&ptr)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	id := common.UTF16PtrToUTF16(ptr)
	ole.CoTaskMemFree(uintptr(unsafe.Pointer(ptr)))
	return id, nil
}

func (v *IPortableDeviceContent) CreateObjectWithPropertiesAndData(pValues *IPortableDeviceValues) (*IPortableDeviceDataStream, int, error) {
	var (
		bufsize uint32
		unk     *ole.IUnknown
		stream  *IPortableDeviceDataStream
	)
	hr, _, _ := syscall.SyscallN(v.VTable().CreateObjectWithPropertiesAndData, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(pValues)), uintptr(unsafe.Pointer(&unk)), uintptr(unsafe.Pointer(&bufsize)), 0)
	if hr != 0 {
		return nil, 0, ole.NewError(hr)
	}
	defer unk.Release()

	if err := unk.PutQueryInterface(ole.NewGUID("88e04db3-1012-4d64-9996-f703a950d3f4"), &stream); err != nil {
		return nil, 0, fmt.Errorf("failed to get IPortableDeviceDataStream: %w", err)
	}
	return stream, int(bufsize), nil
}

type DeleteObjectOptions uint32

const (
	DeviceDeleteNoRecursion   DeleteObjectOptions = 0
	DeviceDeleteWithRecursion DeleteObjectOptions = 1
)

func (v *IPortableDeviceContent) Delete(options DeleteObjectOptions, objectIDs *IPortableDevicePropVariantCollection) error {
	hr, _, _ := syscall.SyscallN(v.VTable().Delete, uintptr(unsafe.Pointer(v)),
		uintptr(uint32(options)), uintptr(unsafe.Pointer(objectIDs)), 0)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}
