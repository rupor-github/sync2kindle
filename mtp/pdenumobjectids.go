//go:build mtp && windows

package mtp

import (
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IEnumPortableDeviceObjectIDs struct {
	ole.IUnknown
}

type IEnumPortableDeviceObjectIDsVtbl struct {
	ole.IUnknownVtbl
	Next   uintptr
	Skip   uintptr
	Reset  uintptr
	Clone  uintptr
	Cancel uintptr
}

func (v *IEnumPortableDeviceObjectIDs) VTable() *IEnumPortableDeviceObjectIDsVtbl {
	return (*IEnumPortableDeviceObjectIDsVtbl)(unsafe.Pointer(v.RawVTable))
}

// WINSDK\um\PortableDeviceApi.idl
//---------------------------------------------------------
// This interface is used to enumerate Objects on a Portable
// device.
//---------------------------------------------------------
// interface IEnumPortableDeviceObjectIDs : IUnknown
// {
//     HRESULT Next(
//         [in]              ULONG   cObjects,
//         [out, size_is(cObjects), length_is(*pcFetched)] LPWSTR* pObjIDs,
//         [in, out, unique] ULONG*  pcFetched);
//
//     HRESULT Skip(
//         [in]  ULONG   cObjects);
//
//     HRESULT Reset();
//
//     HRESULT Clone(
//         [out] IEnumPortableDeviceObjectIDs **ppEnum);
//
//     HRESULT Cancel();
// };
