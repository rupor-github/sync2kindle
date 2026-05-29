//go:build mtp && windows

package mtp

import (
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

func (v *IPortableDeviceValues) SetStringValue(key *PropertyKey, value string) error {
	val, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(v.VTable().SetStringValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(val)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

// same as SetStringValue but allows us to use ObjectID directly without conversion.
func (v *IPortableDeviceValues) SetUTF16Value(key *PropertyKey, value []uint16) error {
	hr, _, _ := syscall.SyscallN(v.VTable().SetStringValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&value[0])))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceValues) GetStringValue(key *PropertyKey) (string, error) {
	var val *uint16
	hr, _, _ := syscall.SyscallN(v.VTable().GetStringValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&val)))
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	result := windows.UTF16PtrToString(val)
	ole.CoTaskMemFree(uintptr(unsafe.Pointer(val)))
	return result, nil
}

func (v *IPortableDeviceValues) SetUnsignedIntegerValue(key *PropertyKey, value uint32) error {
	hr, _, _ := syscall.SyscallN(v.VTable().SetUnsignedIntegerValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(value))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceValues) GetUnsignedIntegerValue(key *PropertyKey) (val uint32, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().GetUnsignedIntegerValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&val)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDeviceValues) GetUnsignedLargeIntegerValue(key *PropertyKey) (val uint64, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().GetUnsignedLargeIntegerValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&val)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDeviceValues) SetUnsignedLargeIntegerValue(key *PropertyKey, val uint64) error {
	hr, _, _ := syscall.SyscallN(v.VTable().SetUnsignedLargeIntegerValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(val))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceValues) GetGuidValue(key *PropertyKey) (val ole.GUID, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().GetGuidValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&val)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDeviceValues) SetGuidValue(key *PropertyKey, val *ole.GUID) error {
	hr, _, _ := syscall.SyscallN(v.VTable().SetGuidValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(val)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceValues) GetBoolValue(key *PropertyKey) (bool, error) {
	var val uint32 // BOOL is 4 bytes
	hr, _, _ := syscall.SyscallN(v.VTable().GetBoolValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&val)))
	if hr != 0 {
		return false, ole.NewError(hr)
	}
	return val != 0, nil
}

func (v *IPortableDeviceValues) GetValue(key *PropertyKey) (*PROPVARIANT, error) {
	pv := PROPVARIANT{}
	hr, _, _ := syscall.SyscallN(v.VTable().GetValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&pv)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return &pv, nil
}

func (v *IPortableDeviceValues) SetValue(key *PropertyKey, val *PROPVARIANT) error {
	hr, _, _ := syscall.SyscallN(v.VTable().SetValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(val)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *IPortableDeviceValues) GetBufferValue(key *PropertyKey) ([]byte, error) {
	var (
		valPtr *byte
		count  uint32
	)
	hr, _, _ := syscall.SyscallN(v.VTable().GetBufferValue, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&valPtr)), uintptr(unsafe.Pointer(&count)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	buf := make([]byte, count)
	copy(buf, unsafe.Slice(valPtr, count))
	ole.CoTaskMemFree(uintptr(unsafe.Pointer(valPtr)))
	return buf, nil
}
