package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"

	"s2k/common"
	"s2k/objects"
)

func (v *IEnumPortableDeviceObjectIDs) Next(number uint32) ([]objects.ObjectID, error) {
	pObjIDs, cObjIDs := make([]*uint16, number), int32(0)
	hr, _, _ := syscall.SyscallN(v.VTable().Next, uintptr(unsafe.Pointer(v)),
		uintptr(number), uintptr(unsafe.Pointer(&pObjIDs[0])), uintptr(unsafe.Pointer(&cObjIDs)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	// copy results into Go managed memory and free up windows one
	oids := make([]objects.ObjectID, cObjIDs)
	for i := 0; i < int(cObjIDs); i++ {
		oids[i] = common.UTF16PtrToUTF16(pObjIDs[i])
		ole.CoTaskMemFree(uintptr(unsafe.Pointer(pObjIDs[i])))
		pObjIDs[i] = nil
	}
	return oids, nil
}
