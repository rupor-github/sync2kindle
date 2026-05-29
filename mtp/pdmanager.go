//go:build mtp && windows

package mtp

import (
	"fmt"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceManager struct {
	ole.IUnknown
}

type IPortableDeviceManagerVtbl struct {
	ole.IUnknownVtbl
	GetDevices            uintptr
	RefreshDeviceList     uintptr
	GetDeviceFriendlyName uintptr
	GetDeviceDescription  uintptr
	GetDeviceManufacturer uintptr
	GetDeviceProperty     uintptr
	GetPrivateDevices     uintptr
}

func CreatePortableDeviceManager() (*IPortableDeviceManager, error) {
	iu, err := ole.CreateInstance(ole.NewGUID("0af10cec-2ecd-4b92-9581-34f6ae0637f3"),
		ole.NewGUID("a1567595-4c2f-4574-a6fa-ecef917b9a40"))
	if err != nil {
		return nil, fmt.Errorf("unable to create IPortableDeviceManager: %w", err)
	}
	return (*IPortableDeviceManager)(unsafe.Pointer(iu)), nil
}

// WINSDK\um\PortableDeviceApi.idl
// ---------------------------------------------------------
// This interface is used to enumerate available portable
// devices.
// ---------------------------------------------------------
// interface IPortableDeviceManager : IUnknown
//
//	{
//	    HRESULT GetDevices(
//	        [in, out, unique]   LPWSTR* pPnPDeviceIDs,
//	        [in, out]           DWORD*  pcPnPDeviceIDs);
//
//	    HRESULT RefreshDeviceList();
//
//	    HRESULT GetDeviceFriendlyName(
//	        [in]                LPCWSTR pszPnPDeviceID,
//	        [in, out, unique]   WCHAR*  pDeviceFriendlyName,
//	        [in, out]           DWORD*  pcchDeviceFriendlyName);
//
//	    HRESULT GetDeviceDescription(
//	        [in]                LPCWSTR pszPnPDeviceID,
//	        [in, out, unique]   WCHAR*  pDeviceDescription,
//	        [in, out]           DWORD*  pcchDeviceDescription);
//
//	    HRESULT GetDeviceManufacturer(
//	        [in]                LPCWSTR pszPnPDeviceID,
//	        [in, out, unique]   WCHAR*  pDeviceManufacturer,
//	        [in, out]           DWORD*  pcchDeviceManufacturer);
//
//	    HRESULT GetDeviceProperty(
//	        [in]                LPCWSTR pszPnPDeviceID,
//	        [in]                LPCWSTR pszDevicePropertyName,
//	        [in, out, unique]   BYTE*   pData,
//	        [in, out, unique]   DWORD*  pcbData,
//	        [in, out, unique]   DWORD*  pdwType);
//
//	    HRESULT GetPrivateDevices(
//	        [in, out, unique]   LPWSTR* pPnPDeviceIDs,
//	        [in, out]           DWORD*  pcPnPDeviceIDs);
//	};
