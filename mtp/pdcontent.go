//go:build mtp && windows

package mtp

import (
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceContent struct {
	ole.IUnknown
}

type IPortableDeviceContentVtbl struct {
	ole.IUnknownVtbl
	EnumObjects                         uintptr
	Properties                          uintptr
	Transfer                            uintptr
	CreateObjectWithPropertiesOnly      uintptr
	CreateObjectWithPropertiesAndData   uintptr
	Delete                              uintptr
	GetObjectIDsFromPersistentUniqueIDs uintptr
	Cancel                              uintptr
	Move                                uintptr
	Copy                                uintptr
}

func (v *IPortableDeviceContent) VTable() *IPortableDeviceContentVtbl {
	return (*IPortableDeviceContentVtbl)(unsafe.Pointer(v.RawVTable))
}

// WINSDK\um\PortableDeviceApi.idl
//---------------------------------------------------------
// This interface is used to work with content on a
// portable device.  From this interface you can enumerate,
// create and delete objects, as well as get interfaces to
// transfer content data and properties.
//---------------------------------------------------------
// interface IPortableDeviceContent : IUnknown
// {
//     HRESULT EnumObjects(
//         [in]         const DWORD                     dwFlags,
//         [in]         LPCWSTR                         pszParentObjectID,
//         [in, unique] IPortableDeviceValues*          pFilter,
//         [out]        IEnumPortableDeviceObjectIDs**  ppEnum);
//
//     HRESULT Properties(
//         [out]  IPortableDeviceProperties** ppProperties);
//
//     HRESULT Transfer(
//         [out]  IPortableDeviceResources** ppResources);
//
//     HRESULT CreateObjectWithPropertiesOnly(
//        [in]              IPortableDeviceValues* pValues,
//        [in, out, unique] LPWSTR*                ppszObjectID);
//
//     HRESULT CreateObjectWithPropertiesAndData(
//        [in]              IPortableDeviceValues* pValues,
//        [out]             IStream**              ppData,
//        [in, out, unique] DWORD*                 pdwOptimalWriteBufferSize,
//        [in, out, unique] LPWSTR*                ppszCookie);
//
//     HRESULT Delete(
//        [in] const DWORD                                         dwOptions,
//        [in] IPortableDevicePropVariantCollection*               pObjectIDs,
//        [in, out, unique] IPortableDevicePropVariantCollection** ppResults);
//
//     HRESULT GetObjectIDsFromPersistentUniqueIDs(
//         [in]    IPortableDevicePropVariantCollection*  pPersistentUniqueIDs,
//         [out]   IPortableDevicePropVariantCollection** ppObjectIDs);
//
//     HRESULT Cancel();
//
//     HRESULT Move(
//         [in] IPortableDevicePropVariantCollection*               pObjectIDs,
//         [in] LPCWSTR                                             pszDestinationFolderObjectID,
//         [in, out, unique] IPortableDevicePropVariantCollection** ppResults);
//
//     HRESULT Copy(
//         [in] IPortableDevicePropVariantCollection*               pObjectIDs,
//         [in] LPCWSTR                                             pszDestinationFolderObjectID,
//         [in, out, unique] IPortableDevicePropVariantCollection** ppResults);
// };
