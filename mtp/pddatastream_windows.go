package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"

	"s2k/common"
	"s2k/objects"
)

// implements io.Writer interface
func (v *IPortableDeviceDataStream) Write(p []byte) (int, error) {
	var pcbWritten uint32
	hr, _, _ := syscall.SyscallN(v.VTable().Write, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&p[0])), uintptr(uint32(len(p))), uintptr(unsafe.Pointer(&pcbWritten)))
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	if pcbWritten != uint32(len(p)) {
		return 0, ole.NewError(ole.E_FAIL)
	}
	return int(pcbWritten), nil
}

type STGCCommitFlags uint32

const (
	STGCDefault                            STGCCommitFlags = 0
	STGCOverWrite                          STGCCommitFlags = 1
	STGCOnlyIfCurrent                      STGCCommitFlags = 2
	STGCDangerouslyCommitMerelyToDiskCache STGCCommitFlags = 4
	STGCConsolidate                        STGCCommitFlags = 8
)

func (v *IPortableDeviceDataStream) Commit(flags STGCCommitFlags) error {
	hr, _, _ := syscall.SyscallN(v.VTable().Commit, uintptr(unsafe.Pointer(v)), uintptr(uint32(flags)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceDataStream) Revert() error {
	hr, _, _ := syscall.SyscallN(v.VTable().Revert, uintptr(unsafe.Pointer(v)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceDataStream) GetObjectID() (objects.ObjectID, error) {
	var ptr *uint16
	hr, _, _ := syscall.SyscallN(v.VTable().GetObjectID, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&ptr)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	id := common.UTF16PtrToUTF16(ptr)
	ole.CoTaskMemFree(uintptr(unsafe.Pointer(ptr)))
	return id, nil
}
