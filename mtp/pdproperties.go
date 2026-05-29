//go:build mtp && windows

package mtp

import (
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceProperties struct {
	ole.IUnknown
}

type IPortableDevicePropertiesVtbl struct {
	ole.IUnknownVtbl
	GetSupportedProperties uintptr
	GetPropertyAttributes  uintptr
	GetValues              uintptr
	SetValues              uintptr
	Delete                 uintptr
	Cancel                 uintptr
}

func (v *IPortableDeviceProperties) VTable() *IPortableDevicePropertiesVtbl {
	return (*IPortableDevicePropertiesVtbl)(unsafe.Pointer(v.RawVTable))
}

// uuid(7f6d695c-03df-4439-a809-59266beee3a6),
//---------------------------------------------------------
// Clients use this interface to work with properties.
// Supports property enumeration, attributes, reading
// and writing.
//---------------------------------------------------------
// interface IPortableDeviceProperties : IUnknown
// {
//     HRESULT GetSupportedProperties(
//         [in]  LPCWSTR                        pszObjectID,
//         [out] IPortableDeviceKeyCollection** ppKeys);
//
//     HRESULT GetPropertyAttributes(
//         [in]  LPCWSTR                  pszObjectID,
//         [in]  REFPROPERTYKEY           Key,
//         [out] IPortableDeviceValues**  ppAttributes);
//
//     HRESULT GetValues(
//         [in]         LPCWSTR                       pszObjectID,
//         [in, unique] IPortableDeviceKeyCollection* pKeys,
//         [out]        IPortableDeviceValues**       ppValues);
//
//     HRESULT SetValues(
//         [in]  LPCWSTR                   pszObjectID,
//         [in]  IPortableDeviceValues*    pValues,
//         [out] IPortableDeviceValues**   ppResults);
//
//     HRESULT Delete(
//         [in]    LPCWSTR                       pszObjectID,
//         [in]    IPortableDeviceKeyCollection* pKeys);
//
//     HRESULT Cancel();
// };
