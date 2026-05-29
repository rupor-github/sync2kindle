//go:build mtp && windows

package mtp

import (
	"fmt"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceValues struct {
	ole.IUnknown
}

type IPortableDeviceValuesVtbl struct {
	ole.IUnknownVtbl
	GetCount                                     uintptr
	GetAt                                        uintptr
	SetValue                                     uintptr
	GetValue                                     uintptr
	SetStringValue                               uintptr
	GetStringValue                               uintptr
	SetUnsignedIntegerValue                      uintptr
	GetUnsignedIntegerValue                      uintptr
	SetSignedIntegerValue                        uintptr
	GetSignedIntegerValue                        uintptr
	SetUnsignedLargeIntegerValue                 uintptr
	GetUnsignedLargeIntegerValue                 uintptr
	SetSignedLargeIntegerValue                   uintptr
	GetSignedLargeIntegerValue                   uintptr
	SetFloatValue                                uintptr
	GetFloatValue                                uintptr
	SetErrorValue                                uintptr
	GetErrorValue                                uintptr
	SetKeyValue                                  uintptr
	GetKeyValue                                  uintptr
	SetBoolValue                                 uintptr
	GetBoolValue                                 uintptr
	SetIUnknownValue                             uintptr
	GetIUnknownValue                             uintptr
	SetGuidValue                                 uintptr
	GetGuidValue                                 uintptr
	SetBufferValue                               uintptr
	GetBufferValue                               uintptr
	SetIPortableDeviceValuesValue                uintptr
	GetIPortableDeviceValuesValue                uintptr
	SetIPortableDevicePropVariantCollectionValue uintptr
	GetIPortableDevicePropVariantCollectionValue uintptr
	SetIPortableDeviceKeyCollectionValue         uintptr
	GetIPortableDeviceKeyCollectionValue         uintptr
	SetIPortableDeviceValuesCollectionValue      uintptr
	GetIPortableDeviceValuesCollectionValue      uintptr
	RemoveValue                                  uintptr
	CopyValuesFromPropertyStore                  uintptr
	CopyValuesToPropertyStore                    uintptr
	Clear                                        uintptr
}

func (v *IPortableDeviceValues) VTable() *IPortableDeviceValuesVtbl {
	return (*IPortableDeviceValuesVtbl)(unsafe.Pointer(v.RawVTable))
}

func CreatePortableDeviceValues() (*IPortableDeviceValues, error) {
	iu, err := ole.CreateInstance(ole.NewGUID("0c15d503-d017-47ce-9016-7b3f978721cc"),
		ole.NewGUID("6848f6f2-3155-4f86-b6f5-263eeeab3143"))
	if err != nil {
		return nil, fmt.Errorf("unable to create IPortableDeviceValues: %w", err)
	}
	return (*IPortableDeviceValues)(unsafe.Pointer(iu)), nil
}

// WINSDK\um\portabledevicetypes.idl
///////////////////////////////////////////////////////////
// IPortableDeviceValues
//---------------------------------------------------------
// A collection of property/value pairs.  Used to get/set
// properties.
//---------------------------------------------------------
// interface IPortableDeviceValues : IUnknown
// {
//     HRESULT GetCount(
//         [in]    DWORD*      pcelt);
//
//     HRESULT GetAt(
//         [in]                const DWORD     index,
//         [in, out, unique]   PROPERTYKEY*    pKey,
//         [in, out, unique]   PROPVARIANT*    pValue);
//
//     HRESULT SetValue(
//         [in]    REFPROPERTYKEY      key,
//         [in]    const PROPVARIANT*  pValue);
//
//     HRESULT GetValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   PROPVARIANT*   pValue);
//
//     HRESULT SetStringValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    LPCWSTR        Value);
//
//     HRESULT GetStringValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   LPWSTR*        pValue);
//
//     HRESULT SetUnsignedIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    const ULONG    Value);
//
//     HRESULT GetUnsignedIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   ULONG*         pValue);
//
//     HRESULT SetSignedIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    const LONG     Value);
//
//     HRESULT GetSignedIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   LONG*          pValue);
//
//     HRESULT SetUnsignedLargeIntegerValue(
//         [in]    REFPROPERTYKEY  key,
//         [in]    const ULONGLONG Value);
//
//     HRESULT GetUnsignedLargeIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   ULONGLONG*     pValue);
//
//     HRESULT SetSignedLargeIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    const LONGLONG Value);
//
//     HRESULT GetSignedLargeIntegerValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   LONGLONG*      pValue);
//
//     HRESULT SetFloatValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    const FLOAT    Value);
//
//     HRESULT GetFloatValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   FLOAT*         pValue);
//
//     HRESULT SetErrorValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    const HRESULT  Value);
//
//     HRESULT GetErrorValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   HRESULT*       pValue);
//
//     HRESULT SetKeyValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    REFPROPERTYKEY Value);
//
//     HRESULT GetKeyValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   PROPERTYKEY*   pValue);
//
//     HRESULT SetBoolValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    const BOOL     Value);
//
//     HRESULT GetBoolValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   BOOL*          pValue);
//
//     HRESULT SetIUnknownValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    IUnknown*      pValue);
//
//     HRESULT GetIUnknownValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   IUnknown**     ppValue);
//
//     HRESULT SetGuidValue(
//         [in]    REFPROPERTYKEY key,
//         [in]    REFGUID        Value);
//
//     HRESULT GetGuidValue(
//         [in]    REFPROPERTYKEY key,
//         [out]   GUID*          pValue);
//
//     HRESULT SetBufferValue(
//         [in]    REFPROPERTYKEY key,
//         [in, size_is(cbValue)]
//                 BYTE*          pValue,
//         [in]    DWORD          cbValue);
//
//     HRESULT GetBufferValue(
//         [in]    REFPROPERTYKEY key,
//         [out, size_is(, *pcbValue)]
//                 BYTE**         ppValue,
//         [out]   DWORD*         pcbValue);
//
//     HRESULT SetIPortableDeviceValuesValue(
//         [in]    REFPROPERTYKEY         key,
//         [in]    IPortableDeviceValues* pValue);
//
//     HRESULT GetIPortableDeviceValuesValue(
//         [in]    REFPROPERTYKEY            key,
//         [out]   IPortableDeviceValues**   ppValue);
//
//     HRESULT SetIPortableDevicePropVariantCollectionValue(
//     [in]    REFPROPERTYKEY                        key,
//     [in]    IPortableDevicePropVariantCollection* pValue);
//
//     HRESULT GetIPortableDevicePropVariantCollectionValue(
//         [in]    REFPROPERTYKEY                          key,
//         [out]   IPortableDevicePropVariantCollection**  ppValue);
//
//     HRESULT SetIPortableDeviceKeyCollectionValue(
//         [in]    REFPROPERTYKEY                key,
//         [in]    IPortableDeviceKeyCollection* pValue);
//
//     HRESULT GetIPortableDeviceKeyCollectionValue(
//         [in]    REFPROPERTYKEY                  key,
//         [out]   IPortableDeviceKeyCollection**  ppValue);
//
//     HRESULT SetIPortableDeviceValuesCollectionValue(
//         [in]    REFPROPERTYKEY                   key,
//         [in]    IPortableDeviceValuesCollection* pValue);
//
//     HRESULT GetIPortableDeviceValuesCollectionValue(
//         [in]    REFPROPERTYKEY                    key,
//         [out]   IPortableDeviceValuesCollection** ppValue);
//
//     HRESULT RemoveValue(
//         [in] REFPROPERTYKEY key);
//
//     HRESULT CopyValuesFromPropertyStore(
//         [in] IPropertyStore* pStore);
//
//     HRESULT CopyValuesToPropertyStore(
//         [in] IPropertyStore* pStore);
//
//     HRESULT Clear();
// };
