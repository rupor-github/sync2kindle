//go:build mtp && windows

package mtp

import (
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

type IPortableDeviceCapabilities struct {
	ole.IUnknown
}

type IPortableDeviceCapabilitiesVtbl struct {
	ole.IUnknownVtbl
	GetSupportedCommands         uintptr
	GetCommandOptions            uintptr
	GetFunctionalCategories      uintptr
	GetFunctionalObjects         uintptr
	GetSupportedContentTypes     uintptr
	GetSupportedFormats          uintptr
	GetSupportedFormatProperties uintptr
	GetFixedPropertyAttributes   uintptr
	Cancel                       uintptr
	GetSupportedEvents           uintptr
	GetEventOptions              uintptr
}

func (v *IPortableDeviceCapabilities) VTable() *IPortableDeviceCapabilitiesVtbl {
	return (*IPortableDeviceCapabilitiesVtbl)(unsafe.Pointer(v.RawVTable))
}

// WINSDK\um\PortableDeviceApi.idl
//---------------------------------------------------------
// Clients use this interface to discover the capabilities
// of the device.
//---------------------------------------------------------
// interface IPortableDeviceCapabilities : IUnknown
// {
//     HRESULT GetSupportedCommands(
//         [out] IPortableDeviceKeyCollection** ppCommands);
//
//     HRESULT GetCommandOptions(
//         [in]  REFPROPERTYKEY           Command,
//         [out] IPortableDeviceValues**  ppOptions);
//
//     HRESULT GetFunctionalCategories(
//         [out] IPortableDevicePropVariantCollection** ppCategories);
//
//     HRESULT GetFunctionalObjects(
//         [in]  REFGUID                                Category,
//         [out] IPortableDevicePropVariantCollection** ppObjectIDs);
//
//     HRESULT GetSupportedContentTypes(
//         [in]  REFGUID                                Category,
//         [out] IPortableDevicePropVariantCollection** ppContentTypes);
//
//     HRESULT GetSupportedFormats(
//         [in]  REFGUID                                ContentType,
//         [out] IPortableDevicePropVariantCollection** ppFormats);
//
//     HRESULT GetSupportedFormatProperties(
//         [in]  REFGUID                        Format,
//         [out] IPortableDeviceKeyCollection** ppKeys);
//
//     HRESULT GetFixedPropertyAttributes(
//         [in]  REFGUID                 Format,
//         [in]  REFPROPERTYKEY          Key,
//         [out] IPortableDeviceValues** ppAttributes);
//
//     HRESULT Cancel();
//
//     HRESULT GetSupportedEvents(
//         [out] IPortableDevicePropVariantCollection** ppEvents);
//
//     HRESULT GetEventOptions(
//         [in]  REFGUID                  Event,
//         [out] IPortableDeviceValues**  ppOptions);
// };
