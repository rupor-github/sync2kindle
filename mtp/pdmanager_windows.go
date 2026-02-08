package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"

	"s2k/common"
)

func (v *IPortableDeviceManager) VTable() *IPortableDeviceManagerVtbl {
	return (*IPortableDeviceManagerVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IPortableDeviceManager) GetDevices() ([]common.PnPDeviceID, error) {
	// get number of attached devices
	var cPnPDeviceIDs uint32
	hr, _, _ := syscall.SyscallN(v.VTable().GetDevices, uintptr(unsafe.Pointer(v)),
		0, uintptr(unsafe.Pointer(&cPnPDeviceIDs)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	if cPnPDeviceIDs == 0 {
		return nil, nil
	}

	// get device ids for all attached devices
	pPnPDeviceIDs := make([]*uint16, cPnPDeviceIDs)
	hr, _, _ = syscall.SyscallN(v.VTable().GetDevices, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&pPnPDeviceIDs[0])), uintptr(unsafe.Pointer(&cPnPDeviceIDs)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}

	// copy results into Go managed memory and free up windows one
	dids := make([]common.PnPDeviceID, cPnPDeviceIDs)
	for i := 0; i < int(cPnPDeviceIDs); i++ {
		dids[i] = common.UTF16PtrToUTF16(pPnPDeviceIDs[i])
		ole.CoTaskMemFree(uintptr(unsafe.Pointer(pPnPDeviceIDs[i])))
		pPnPDeviceIDs[i] = nil
	}
	return dids, nil
}

func (v *IPortableDeviceManager) RefreshDeviceList() error {
	hr, _, _ := syscall.SyscallN(v.VTable().RefreshDeviceList, uintptr(unsafe.Pointer(v)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceManager) GetDeviceFriendlyName(id common.PnPDeviceID) (string, error) {
	// get length
	var cFriendlyName uint32
	hr, _, _ := syscall.SyscallN(v.VTable().GetDeviceFriendlyName, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), 0, uintptr(unsafe.Pointer(&cFriendlyName)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	if cFriendlyName == 0 {
		return "", nil
	}
	// get value
	pFriendlyName := make([]uint16, cFriendlyName)
	hr, _, _ = syscall.SyscallN(v.VTable().GetDeviceFriendlyName, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), uintptr(unsafe.Pointer(&pFriendlyName[0])), uintptr(unsafe.Pointer(&cFriendlyName)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	return syscall.UTF16ToString(pFriendlyName), nil
}

func (v *IPortableDeviceManager) GetDeviceDescription(id common.PnPDeviceID) (string, error) {
	// get length
	var cDeviceDescription uint32
	hr, _, _ := syscall.SyscallN(v.VTable().GetDeviceDescription, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), 0, uintptr(unsafe.Pointer(&cDeviceDescription)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	if cDeviceDescription == 0 {
		return "", nil
	}
	// get value
	pDeviceDescription := make([]uint16, cDeviceDescription)
	hr, _, _ = syscall.SyscallN(v.VTable().GetDeviceDescription, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), uintptr(unsafe.Pointer(&pDeviceDescription[0])), uintptr(unsafe.Pointer(&cDeviceDescription)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	return syscall.UTF16ToString(pDeviceDescription), nil
}

func (v *IPortableDeviceManager) GetDeviceManufacturer(id common.PnPDeviceID) (string, error) {
	// get length
	var cDeviceManufacturer uint32
	hr, _, _ := syscall.SyscallN(v.VTable().GetDeviceManufacturer, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), 0, uintptr(unsafe.Pointer(&cDeviceManufacturer)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	if cDeviceManufacturer == 0 {
		return "", nil
	}
	// get value
	pDeviceManufacturer := make([]uint16, cDeviceManufacturer)
	hr, _, _ = syscall.SyscallN(v.VTable().GetDeviceManufacturer, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&id[0])), uintptr(unsafe.Pointer(&pDeviceManufacturer[0])), uintptr(unsafe.Pointer(&cDeviceManufacturer)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	return syscall.UTF16ToString(pDeviceManufacturer), nil
}
