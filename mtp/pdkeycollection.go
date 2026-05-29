//go:build mtp && windows

package mtp

import (
	"fmt"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceKeyCollection struct {
	ole.IUnknown
}

type IPortableDeviceKeyCollectionVtbl struct {
	ole.IUnknownVtbl
	GetCount uintptr
	GetAt    uintptr
	Add      uintptr
	Clear    uintptr
	RemoveAt uintptr
}

func (v *IPortableDeviceKeyCollection) VTable() *IPortableDeviceKeyCollectionVtbl {
	return (*IPortableDeviceKeyCollectionVtbl)(unsafe.Pointer(v.RawVTable))
}

func CreatePortableDeviceKeyCollection() (*IPortableDeviceKeyCollection, error) {
	iu, err := ole.CreateInstance(ole.NewGUID("de2d022d-2480-43be-97f0-d1fa2cf98f4f"),
		ole.NewGUID("dada2357-e0ad-492e-98db-dd61c53ba353"))
	if err != nil {
		return nil, fmt.Errorf("unable to create IPortableDeviceKeyCollection: %w", err)
	}
	return (*IPortableDeviceKeyCollection)(unsafe.Pointer(iu)), nil
}

// WINSDK\um\PortableDeviceTypes.idl
///////////////////////////////////////////////////////////
// IPortableDeviceKeyCollection
//---------------------------------------------------------
// interface IPortableDeviceKeyCollection : IUnknown
// {
//    HRESULT GetCount(
//       [in] DWORD* pcElems);
//
//    HRESULT GetAt(
//       [in] const DWORD dwIndex,
//       [in] PROPERTYKEY* pKey);
//
//    HRESULT Add(
//       [in] REFPROPERTYKEY Key);
//
//    HRESULT Clear();
//
//    HRESULT RemoveAt(
//       [in] const DWORD dwIndex);
// };
