//go:build mtp && windows

package mtp

import (
	"syscall"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

var (
	modpropsys = windows.NewLazySystemDLL("propsys.dll")
	modole32   = windows.NewLazySystemDLL("ole32.dll")

	procInitVariantFromFiletime = modpropsys.NewProc("InitVariantFromFileTime")
	procCoTaskMemAlloc          = modole32.NewProc("CoTaskMemAlloc")
	procPropVariantClear        = modole32.NewProc("PropVariantClear")
)

// From go/src/internal/goarch.go:
// PtrSize is the size of a pointer in bytes - unsafe.Sizeof(uintptr(0)) but as an ideal constant.
// It is also the size of the machine's native word size (that is, 4 on 32-bit systems, 8 on 64-bit).
const ptrSize = 4 << (^uintptr(0) >> 63)

// NOTE, it looks to me that PROPVARIANT has exactly the same size and structure as VARIANT, only
// with more supported formats and fields are interpreted differently, so theoretically we could
// extend ole.Variant machinery here, but just to be on a safe side let's do what Window does and
// redifine it completely. It should work for all Windows platforms - both arm and amd, 32 and 64 bits,
// but needs validation as go-ole handles architectures differently
type PROPVARIANT struct {
	Vt         ole.VT
	WReserved1 uint16
	WReserved2 uint16
	WReserved3 uint16
	storage1   [ptrSize * 2]byte
}

func (pv *PROPVARIANT) Clear() error {
	hr, _, _ := procPropVariantClear.Call(uintptr(unsafe.Pointer(pv)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (pv *PROPVARIANT) Puuid() *ole.GUID {
	return *(**ole.GUID)(unsafe.Pointer(&pv.storage1[0]))
}

func (pv *PROPVARIANT) Time() time.Time {
	t, err := ole.GetVariantDate(*(*uint64)(unsafe.Pointer(&pv.storage1[0])))
	if err != nil {
		return time.Time{}
	}
	return t
}

func (v *IPortableDevicePropVariantCollection) GetCount() (c uint32, err error) {
	hr, _, _ := syscall.SyscallN(v.VTable().GetCount, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&c)))
	if hr != 0 {
		err = ole.NewError(hr)
	}
	return
}

func (v *IPortableDevicePropVariantCollection) GetAt(i uint32) (*PROPVARIANT, error) {
	pv := PROPVARIANT{}
	hr, _, _ := syscall.SyscallN(v.VTable().GetAt, uintptr(unsafe.Pointer(v)),
		uintptr(i), uintptr(unsafe.Pointer(&pv)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return &pv, nil
}

func (v *IPortableDevicePropVariantCollection) Add(pv *PROPVARIANT) error {
	hr, _, _ := syscall.SyscallN(v.VTable().Add, uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(pv)))
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

// Allows us to use ObjectID directly without conversion. It is always zero terminated.
func NewPropVariantFromUTF16(data []uint16) (*PROPVARIANT, error) {
	symSize := unsafe.Sizeof(data[0])
	pv := &PROPVARIANT{Vt: ole.VT_LPWSTR}
	ptr, _, _ := procCoTaskMemAlloc.Call(uintptr(len(data)) * symSize)
	if ptr == 0 {
		return nil, windows.ERROR_OUTOFMEMORY
	}
	*(*uintptr)(unsafe.Pointer(&pv.storage1[0])) = ptr
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(data)), data)
	return pv, nil
}

func NewPropVariantFromTime(t time.Time) (*PROPVARIANT, error) {
	pv := &PROPVARIANT{}
	ft := syscall.NsecToFiletime(t.UnixNano())
	hr, _, _ := procInitVariantFromFiletime.Call(uintptr(unsafe.Pointer(&ft)), uintptr(unsafe.Pointer(pv)))
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return pv, nil
}
