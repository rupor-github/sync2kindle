//go:build mtp && windows

package mtp

import (
	"fmt"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDevice struct {
	ole.IUnknown
}

type IPortableDeviceVtbl struct {
	ole.IUnknownVtbl
	Open           uintptr
	SendCommand    uintptr
	Content        uintptr
	Capabilities   uintptr
	Cancel         uintptr
	Close          uintptr
	Advise         uintptr
	Unadvise       uintptr
	GetPnPDeviceID uintptr
}

func (v *IPortableDevice) VTable() *IPortableDeviceVtbl {
	return (*IPortableDeviceVtbl)(unsafe.Pointer(v.RawVTable))
}

func CreatePortableDevice() (*IPortableDevice, error) {

	// Looks like Go runtime requires legacy marshaling - FTM sporadically crashes in Copy action.
	// FTM: iu, err := ole.CreateInstance(ole.NewGUID("f7c0039a-4762-488a-b4b3-760ef9a1ba9b"),

	iu, err := ole.CreateInstance(ole.NewGUID("728a21c5-3d9e-48d7-9810-864848f0f404"),
		ole.NewGUID("625e2df8-6392-4cf0-9ad1-3cfa5f17775c"))
	if err != nil {
		return nil, fmt.Errorf("unable to create IPortableDevice: %w", err)
	}
	return (*IPortableDevice)(unsafe.Pointer(iu)), nil
}

// WINSDK\um\PortableDeviceApi.idl
//---------------------------------------------------------
// This interface forms the basis of communication from
// applications to Windows Portable Devices devices.
// Since this offers fairly low-level access, higher
// level functionality is provided via API objects which
// build on this class.
//---------------------------------------------------------
// interface IPortableDevice : IUnknown
// {
//     HRESULT Open(
//         [in]    LPCWSTR                 pszPnPDeviceID,
//         [in]    IPortableDeviceValues*  pClientInfo);
//
//     HRESULT SendCommand(
//         [in]   const DWORD              dwFlags,
//         [in]   IPortableDeviceValues*   pParameters,
//         [out]  IPortableDeviceValues**  ppResults);
//
//     HRESULT Content(
//        [out]  IPortableDeviceContent** ppContent);
//
//     HRESULT Capabilities(
//        [out]  IPortableDeviceCapabilities** ppCapabilities);
//
//     HRESULT Cancel();
//
//     HRESULT Close();
//
//     HRESULT Advise(
//         [in]   const DWORD                      dwFlags,
//         [in]   IPortableDeviceEventCallback*    pCallback,
//         [in, unique]   IPortableDeviceValues*   pParameters,
//         [out]  LPWSTR*                          ppszCookie);
//
//     HRESULT Unadvise(
//         [in]   LPCWSTR pszCookie);
//
//     HRESULT GetPnPDeviceID(
//         [out]  LPWSTR*  ppszPnPDeviceID);
// };
