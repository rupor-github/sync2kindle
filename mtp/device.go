package mtp

import (
	humanize "github.com/dustin/go-humanize"
	ole "github.com/go-ole/go-ole"

	"s2k/objects"
)

// should be usable in the zap log.Named()
const driverName = "mtp-device"

// most of this originated on Windows, but some MTP related types are cross-platform.

type propSet map[string]string

type PropertyKey struct {
	fmtid ole.GUID
	pid   uint32
}

var (
	WPD_DEVICE_OBJECT_ID objects.ObjectID

	// client information properties
	WPD_CLIENT_INFORMATION_PROPERTIES_V1 = ole.GUID{Data1: 0x204D9F0C, Data2: 0x2292, Data3: 0x4080, Data4: [8]byte{0x9F, 0x42, 0x40, 0x66, 0x4E, 0x70, 0xF8, 0x59}}

	WPD_CLIENT_NAME                        = &PropertyKey{fmtid: WPD_CLIENT_INFORMATION_PROPERTIES_V1, pid: 2}
	WPD_CLIENT_MAJOR_VERSION               = &PropertyKey{fmtid: WPD_CLIENT_INFORMATION_PROPERTIES_V1, pid: 3}
	WPD_CLIENT_MINOR_VERSION               = &PropertyKey{fmtid: WPD_CLIENT_INFORMATION_PROPERTIES_V1, pid: 4}
	WPD_CLIENT_REVISION                    = &PropertyKey{fmtid: WPD_CLIENT_INFORMATION_PROPERTIES_V1, pid: 5}
	WPD_CLIENT_SECURITY_QUALITY_OF_SERVICE = &PropertyKey{fmtid: WPD_CLIENT_INFORMATION_PROPERTIES_V1, pid: 8}

	// device properties (v1)
	WPD_DEVICE_PROPERTIES_V1 = ole.GUID{Data1: 0x26D4979A, Data2: 0xE643, Data3: 0x4626, Data4: [8]byte{0x9E, 0x2B, 0x73, 0x6D, 0xC0, 0xC9, 0x2F, 0xDC}}

	WPD_DEVICE_FIRMWARE_VERSION = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 3}
	WPD_DEVICE_PROTOCOL         = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 6}
	WPD_DEVICE_MANUFACTURER     = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 7}
	WPD_DEVICE_MODEL            = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 8}
	WPD_DEVICE_SERIAL_NUMBER    = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 9}
	WPD_DEVICE_FRIENDLY_NAME    = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 12}
	WPD_DEVICE_TYPE             = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V1, pid: 15}

	// device properties (v2)
	WPD_DEVICE_PROPERTIES_V2 = ole.GUID{Data1: 0x463DD662, Data2: 0x7FC4, Data3: 0x4291, Data4: [8]byte{0x91, 0x1C, 0x7F, 0x4C, 0x9C, 0xCA, 0x97, 0x99}}

	WPD_DEVICE_FUNCTIONAL_UNIQUE_ID = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V2, pid: 2}
	WPD_DEVICE_MODEL_UNIQUE_ID      = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V2, pid: 3}
	WPD_DEVICE_TRANSPORT            = &PropertyKey{fmtid: WPD_DEVICE_PROPERTIES_V2, pid: 4}

	// functional object properties (v1)
	WPD_FUNCTIONAL_OBJECT_PROPERTIES_V1 = ole.GUID{Data1: 0x8F052D93, Data2: 0xABCA, Data3: 0x4FC5, Data4: [8]byte{0xA5, 0xAC, 0xB0, 0x1D, 0xF4, 0xDB, 0xE5, 0x98}}

	WPD_FUNCTIONAL_OBJECT_CATEGORY = &PropertyKey{fmtid: WPD_FUNCTIONAL_OBJECT_PROPERTIES_V1, pid: 2}

	// object properties (v1)
	WPD_OBJECT_PROPERTIES_V1 = ole.GUID{Data1: 0xEF6B490D, Data2: 0x5CD8, Data3: 0x437A, Data4: [8]byte{0xAF, 0xFC, 0xDA, 0x8B, 0x60, 0xEE, 0x4A, 0x3C}}

	// Common object properties
	WPD_OBJECT_CONTENT_TYPE = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 7}
	// Legacy WPD Properties
	WPD_OBJECT_PARENT_ID            = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 3}
	WPD_OBJECT_NAME                 = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 4}
	WPD_OBJECT_PERSISTENT_UNIQUE_ID = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 5}
	WPD_OBJECT_FORMAT               = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 6}
	WPD_OBJECT_ISHIDDEN             = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 9}
	WPD_OBJECT_ISSYSTEM             = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 10}
	WPD_OBJECT_SIZE                 = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 11}
	WPD_OBJECT_ORIGINAL_FILE_NAME   = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 12}
	WPD_OBJECT_DATE_CREATED         = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 18}
	WPD_OBJECT_DATE_MODIFIED        = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 19}
	WPD_OBJECT_CAN_DELETE           = &PropertyKey{fmtid: WPD_OBJECT_PROPERTIES_V1, pid: 26}

	// Legacy WPD Formats
	WPD_OBJECT_FORMAT_UNSPECIFIED = ole.GUID{Data1: 0x30000000, Data2: 0xAE6C, Data3: 0x4804, Data4: [8]byte{0x98, 0xBA, 0xC5, 0x7B, 0x46, 0x96, 0x5F, 0xE7}}

	// WPD content types (only the ones I observed, full WPD list is much longer

	// Indicates this object represents a functional object, not content data on the device.
	WPD_CONTENT_TYPE_FUNCTIONAL_OBJECT = ole.GUID{Data1: 0x99ED0160, Data2: 0x17FF, Data3: 0x4C44, Data4: [8]byte{0x9D, 0x98, 0x1D, 0x7A, 0x6F, 0x94, 0x19, 0x21}}
	// Indicates this object is a folder.
	WPD_CONTENT_TYPE_FOLDER = ole.GUID{Data1: 0x27E2E392, Data2: 0xA111, Data3: 0x48E0, Data4: [8]byte{0xAB, 0x0C, 0xE1, 0x77, 0x05, 0xA0, 0x5F, 0x85}}
	// Indicates this object represents image data (e.g. a JPEG file)
	WPD_CONTENT_TYPE_IMAGE = ole.GUID{Data1: 0xef2107d5, Data2: 0xa52a, Data3: 0x4243, Data4: [8]byte{0xa2, 0x6b, 0x62, 0xd4, 0x17, 0x6d, 0x76, 0x03}}
	// Indicates this object represents document data (e.g. a MS WORD file, TEXT file, etc.)
	WPD_CONTENT_TYPE_DOCUMENT = ole.GUID{Data1: 0x680ADF52, Data2: 0x950A, Data3: 0x4041, Data4: [8]byte{0x9B, 0x41, 0x65, 0xE3, 0x93, 0x64, 0x81, 0x55}}
	// Indicates this object represents a file that does not fall into any of the other predefined WPD types for files.
	WPD_CONTENT_TYPE_GENERIC_FILE = ole.GUID{Data1: 0x0085E0A6, Data2: 0x8D34, Data3: 0x45D7, Data4: [8]byte{0xBC, 0x5C, 0x44, 0x7E, 0x59, 0xC7, 0x3D, 0x48}}
	// Indicates this object doesn't fall into the predefined WPD content types
	WPD_CONTENT_TYPE_UNSPECIFIED = ole.GUID{Data1: 0x28D8D31E, Data2: 0x249C, Data3: 0x454E, Data4: [8]byte{0xAA, 0xBC, 0x34, 0x88, 0x31, 0x68, 0xE6, 0x34}}

	// functional categories
	WPD_FUNCTIONAL_CATEGORY_STORAGE = ole.GUID{Data1: 0x23F05BBC, Data2: 0x15DE, Data3: 0x4C2A, Data4: [8]byte{0xA5, 0x5B, 0xA9, 0xAF, 0x5C, 0xE4, 0x12, 0xEF}}

	// storage object properties (v1)
	WPD_STORAGE_OBJECT_PROPERTIES_V1 = ole.GUID{Data1: 0x01A3057A, Data2: 0x74D6, Data3: 0x4E80, Data4: [8]byte{0xBE, 0xA7, 0xDC, 0x4C, 0x21, 0x2C, 0xE5, 0x0A}}

	WPD_STORAGE_TYPE                  = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 2}
	WPD_STORAGE_FILE_SYSTEM_TYPE      = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 3}
	WPD_STORAGE_CAPACITY              = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 4}
	WPD_STORAGE_FREE_SPACE_IN_BYTES   = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 5}
	WPD_STORAGE_FREE_SPACE_IN_OBJECTS = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 6}
	WPD_STORAGE_DESCRIPTION           = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 7}
	WPD_STORAGE_SERIAL_NUMBER         = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 8}
	WPD_STORAGE_MAX_OBJECT_SIZE       = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 9}
	WPD_STORAGE_CAPACITY_IN_OBJECTS   = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 10}
	WPD_STORAGE_ACCESS_CAPABILITY     = &PropertyKey{fmtid: WPD_STORAGE_OBJECT_PROPERTIES_V1, pid: 11}
)

type WPDStorageBytes uint64

func (n WPDStorageBytes) String() string {
	return humanize.Bytes(uint64(n)) + " - " + humanize.Comma(int64(n)) + " bytes"
}

type WPDStorageObjects int64

func (n WPDStorageObjects) String() string {
	return humanize.Comma(int64(n))
}

type WPDStorageAccessCapability uint32

const (
	WPD_STORAGE_ACCESS_CAPABILITY_READWRITE WPDStorageAccessCapability = iota
	WPD_STORAGE_ACCESS_CAPABILITY_READ_ONLY_WITHOUT_OBJECT_DELETION
	WPD_STORAGE_ACCESS_CAPABILITY_READ_ONLY_WITH_OBJECT_DELETION
)

func (c WPDStorageAccessCapability) String() string {
	switch c {
	case WPD_STORAGE_ACCESS_CAPABILITY_READWRITE:
		return "Read/Write"
	case WPD_STORAGE_ACCESS_CAPABILITY_READ_ONLY_WITHOUT_OBJECT_DELETION:
		return "Read-Only without Object Deletion"
	case WPD_STORAGE_ACCESS_CAPABILITY_READ_ONLY_WITH_OBJECT_DELETION:
		return "Read-Only with Object Deletion"
	}
	return "Unknown"
}

type WPDStorageFileSystemType uint32

const (
	WPD_FST_UNDEFINED WPDStorageFileSystemType = iota
	WPD_FST_GENERICFLAT
	WPD_FST_GENERICHIERARCHICAL
	WPD_FST_DCF
)

func (t WPDStorageFileSystemType) String() string {
	switch t {
	case WPD_FST_UNDEFINED:
		return "Undefined"
	case WPD_FST_GENERICFLAT:
		return "Generic Flat"
	case WPD_FST_GENERICHIERARCHICAL:
		return "Generic Hierarchical"
	case WPD_FST_DCF:
		return "DCF"
	}
	return "Unknown"
}

type WPDStorageType uint32

const (
	WPD_STORAGE_TYPE_UNDEFINED WPDStorageType = iota
	WPD_STORAGE_TYPE_FIXED_ROM
	WPD_STORAGE_TYPE_REMOVABLE_ROM
	WPD_STORAGE_TYPE_FIXED_RAM
	WPD_STORAGE_TYPE_REMOVABLE_RAM
)

func (t WPDStorageType) String() string {
	switch t {
	case WPD_STORAGE_TYPE_UNDEFINED:
		return "Undefined"
	case WPD_STORAGE_TYPE_FIXED_ROM:
		return "Fixed ROM"
	case WPD_STORAGE_TYPE_REMOVABLE_ROM:
		return "Removable ROM"
	case WPD_STORAGE_TYPE_FIXED_RAM:
		return "Fixed RAM"
	case WPD_STORAGE_TYPE_REMOVABLE_RAM:
		return "Removable RAM"
	}
	return "Unknown"
}

type WPDDeviceTypes uint32

const (
	WPD_DEVICE_TYPE_GENERIC WPDDeviceTypes = iota
	WPD_DEVICE_TYPE_CAMERA
	WPD_DEVICE_TYPE_MEDIA_PLAYER
	WPD_DEVICE_TYPE_PHONE
	WPD_DEVICE_TYPE_VIDEO
	WPD_DEVICE_TYPE_PERSONAL_INFORMATION_MANAGER
	WPD_DEVICE_TYPE_AUDIO_RECORDER
)

func (t WPDDeviceTypes) String() string {
	switch t {
	case WPD_DEVICE_TYPE_GENERIC:
		return "Generic"
	case WPD_DEVICE_TYPE_CAMERA:
		return "Camera"
	case WPD_DEVICE_TYPE_MEDIA_PLAYER:
		return "Media Player"
	case WPD_DEVICE_TYPE_PHONE:
		return "Phone"
	case WPD_DEVICE_TYPE_VIDEO:
		return "Video"
	case WPD_DEVICE_TYPE_PERSONAL_INFORMATION_MANAGER:
		return "Personal Information Manager"
	case WPD_DEVICE_TYPE_AUDIO_RECORDER:
		return "Audio Recorder"
	}
	return "Unknown"
}

type WPDDeviceTransports uint32

const (
	WPD_DEVICE_TRANSPORT_UNSPECIFIED WPDDeviceTransports = iota
	WPD_DEVICE_TRANSPORT_USB
	WPD_DEVICE_TRANSPORT_IP
	WPD_DEVICE_TRANSPORT_BLUETOOTH
)

func (t WPDDeviceTransports) String() string {
	switch t {
	case WPD_DEVICE_TRANSPORT_UNSPECIFIED:
		return "Unspecified"
	case WPD_DEVICE_TRANSPORT_USB:
		return "USB"
	case WPD_DEVICE_TRANSPORT_IP:
		return "IP"
	case WPD_DEVICE_TRANSPORT_BLUETOOTH:
		return "Bluetooth"
	}
	return "Unknown"
}
