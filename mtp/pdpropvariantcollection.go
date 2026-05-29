//go:build mtp && windows

package mtp

import (
	"fmt"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDevicePropVariantCollection struct {
	ole.IUnknown
}

type IPortableDevicePropVariantCollectionVtbl struct {
	ole.IUnknownVtbl
	GetCount   uintptr
	GetAt      uintptr
	Add        uintptr
	GetType    uintptr
	ChangeType uintptr
	Clear      uintptr
	RemoveAt   uintptr
}

func (v *IPortableDevicePropVariantCollection) VTable() *IPortableDevicePropVariantCollectionVtbl {
	return (*IPortableDevicePropVariantCollectionVtbl)(unsafe.Pointer(v.RawVTable))
}

func CreatePortableDevicePropVariantCollection() (*IPortableDevicePropVariantCollection, error) {
	iu, err := ole.CreateInstance(ole.NewGUID("08a99e2f-6d6d-4b80-af5a-baf2bcbe4cb9"),
		ole.NewGUID("89b2e422-4f1b-4316-bcef-a44afea83eb3"))
	if err != nil {
		return nil, fmt.Errorf("unable to create IPortableDevicePropVariantCollection: %w", err)
	}
	return (*IPortableDevicePropVariantCollection)(unsafe.Pointer(iu)), nil
}

// WINSDK\um\PortableDeviceTypes.idl
///////////////////////////////////////////////////////////
// IPortableDevicePropVariantCollection
//---------------------------------------------------------
// interface IPortableDevicePropVariantCollection : IUnknown
// {
//    HRESULT GetCount(
//       [in] DWORD* pcElems);
//
//    HRESULT GetAt(
//       [in] const DWORD  dwIndex,
//       [in] PROPVARIANT* pValue);
//
//    HRESULT Add(
//       [in] const PROPVARIANT* pValue);
//
//    HRESULT GetType(
//       [out] VARTYPE* pvt);
//
//    HRESULT ChangeType(
//       [in] const VARTYPE vt);
//
//    HRESULT Clear();
//
//    HRESULT RemoveAt(
//       [in] const DWORD dwIndex);
// };
