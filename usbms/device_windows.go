//go:build usb && windows

package usbms

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"s2k/common"
	"s2k/files"
)

type Device struct {
	*files.Device
	log     *zap.Logger
	id      common.PnPDeviceID
	devinst windows.DEVINST
	eject   bool
}

// Connect to the supported device.
func Connect(paths, serial string, eject bool, log *zap.Logger) (*Device, error) {

	var mount string
	id, mount, devinst, err := pickDevice(serial, log)
	if err != nil {
		return nil, err
	}

	d := &Device{log: log.Named(driverName), id: id, devinst: devinst, eject: eject}
	d.Device, err = files.Connect(paths, filepath.ToSlash(mount), nil, d.log)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// driver interface

func (d *Device) Disconnect() {
	if d != nil && d.eject {
		vetoName := make([]uint16, windows.MAX_PATH)
		vetoType := vetoType(0)

		if rc, _, _ := cmRequestDeviceEject.Call(
			uintptr(d.devinst), uintptr(unsafe.Pointer(&vetoType)),
			uintptr(unsafe.Pointer(&vetoName[0])), uintptr(len(vetoName)), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			if windows.CONFIGRET(rc) == windows.CR_REMOVE_VETOED {
				d.log.Warn("Eject failed", zap.Stringer("reason", vetoType))
				info := windows.UTF16ToString(vetoName)
				if len(info) > 0 {
					d.log.Debug("Eject failed", zap.String("Additional info", info))
				}
				return
			}
			d.log.Error("Eject failed", zap.Error(windows.CONFIGRET(rc)))
		}
	}
}

func (d *Device) Name() string {
	return driverName
}

func (d *Device) UniqueID() string {
	return d.id.Serial()
}

// implementation

type vetoType uint32

const (
	PNP_VetoTypeUnknown vetoType = iota
	PNP_VetoLegacyDevice
	PNP_VetoPendingClose
	PNP_VetoWindowsApp
	PNP_VetoWindowsService
	PNP_VetoOutstandingOpen
	PNP_VetoDevice
	PNP_VetoDriver
	PNP_VetoIllegalDeviceRequest
	PNP_VetoInsufficientPower
	PNP_VetoNonDisableable
	PNP_VetoLegacyDriver
	PNP_VetoInsufficientRights
	PNP_VetoAlreadyRemoved
)

func (v vetoType) String() string {
	switch v {
	case PNP_VetoLegacyDevice:
		return "The device does not support the specified PnP operation"
	case PNP_VetoPendingClose:
		return "The specified operation cannot be completed because of a pending close operation"
	case PNP_VetoWindowsApp:
		return "A Microsoft Win32 application vetoed the specified operation"
	case PNP_VetoWindowsService:
		return "A Win32 service vetoed the specified operation"
	case PNP_VetoOutstandingOpen:
		return "The requested operation was rejected because of outstanding open handles"
	case PNP_VetoDevice:
		return "The device supports the specified operation, but the device rejected the operation"
	case PNP_VetoDriver:
		return "The driver supports the specified operation, but the driver rejected the operation"
	case PNP_VetoIllegalDeviceRequest:
		return "The device does not support the specified operation"
	case PNP_VetoInsufficientPower:
		return "There is insufficient power to perform the requested operation"
	case PNP_VetoNonDisableable:
		return "The device cannot be disabled"
	case PNP_VetoLegacyDriver:
		return "The driver does not support the specified PnP operation"
	case PNP_VetoInsufficientRights:
		return "The caller has insufficient privileges to complete the operation"
	case PNP_VetoAlreadyRemoved:
		return "The device has already been removed"
	case PNP_VetoTypeUnknown:
		fallthrough
	default:
		return "The specified operation was rejected for an unknown reason"
	}
}

var (
	modsetupapi = windows.NewLazySystemDLL("setupapi.dll")

	setupDiGetClassDevs             = modsetupapi.NewProc("SetupDiGetClassDevsW")
	setupDiEnumDeviceInterfaces     = modsetupapi.NewProc("SetupDiEnumDeviceInterfaces")
	setupDiGetDeviceInterfaceDetail = modsetupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	setupDiDestroyDeviceInfoList    = modsetupapi.NewProc("SetupDiDestroyDeviceInfoList")

	modecmgr = windows.NewLazySystemDLL("cfgmgr32.dll")

	cmGetChild           = modecmgr.NewProc("CM_Get_Child")
	cmGetDeviceIDSize    = modecmgr.NewProc("CM_Get_Device_ID_Size")
	cmGetDeviceID        = modecmgr.NewProc("CM_Get_Device_IDW")
	cmRequestDeviceEject = modecmgr.NewProc("CM_Request_Device_EjectW")
)

type (
	devInfoData struct {
		size      uint32
		ClassGUID windows.GUID
		DevInst   windows.DEVINST
		_         uintptr
	}

	deviceInterfaceData struct {
		cbSize             uint32
		InterfaceClassGuid windows.GUID
		Flags              uint32
		_                  uintptr
	}

	deviceInterfaceDetailData struct {
		cbSize     uint32
		DevicePath [1]uint16 // ANYSIZE_ARRAY
	}
)

func pickDevice(serial string, log *zap.Logger) (common.PnPDeviceID, string, windows.DEVINST, error) {

	guid := ole.NewGUID("a5dcbf10-6530-11d2-901f-00c04fb951ed") // GUID_DEVINTERFACE_USB_DEVICE

	hDevInfo, _, err := setupDiGetClassDevs.Call(uintptr(unsafe.Pointer(guid)), uintptr(0), uintptr(0),
		uintptr(windows.DIGCF_PRESENT|windows.DIGCF_DEVICEINTERFACE))
	if windows.Handle(hDevInfo) == windows.InvalidHandle {
		return nil, "", windows.DEVINST(0), fmt.Errorf("unable to enumerate disk devices: %w", err)
	}
	defer setupDiDestroyDeviceInfoList.Call(hDevInfo)

	var (
		result     common.PnPDeviceID
		devinst    windows.DEVINST
		volumePath string
		dwSize     uint32
	)
	for dwIndex := 0; ; dwIndex++ {
		spdid := &deviceInterfaceData{}
		spdid.cbSize = uint32(unsafe.Sizeof(*spdid))

		res, _, err := setupDiEnumDeviceInterfaces.Call(hDevInfo,
			uintptr(0), uintptr(unsafe.Pointer(guid)), uintptr(dwIndex), uintptr(unsafe.Pointer(spdid)))
		if res == 0 {
			var errno syscall.Errno
			if errors.As(err, &errno) && errno == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to enumerate disk device interfaces: %w", err)
		}

		setupDiGetDeviceInterfaceDetail.Call(hDevInfo, uintptr(unsafe.Pointer(spdid)), uintptr(0), uintptr(0), uintptr(unsafe.Pointer(&dwSize)), uintptr(0))
		if dwSize <= 0 {
			continue
		}

		buf := make([]byte, dwSize)
		spdidd := (*deviceInterfaceDetailData)(unsafe.Pointer(&buf[0]))
		spdidd.cbSize = uint32(unsafe.Sizeof(spdidd))

		spdd := &devInfoData{}
		spdd.size = uint32(unsafe.Sizeof(*spdd))

		res, _, err = setupDiGetDeviceInterfaceDetail.Call(hDevInfo,
			uintptr(unsafe.Pointer(spdid)), uintptr(unsafe.Pointer(spdidd)), uintptr(dwSize), uintptr(unsafe.Pointer(&dwSize)), uintptr(unsafe.Pointer(spdd)))
		if res == 0 {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get disk device interface details: %w", err)
		}

		id := common.PnPDeviceID(common.UTF16PtrToUTF16(&spdidd.DevicePath[0]))
		vid, pid := id.VendorID(), id.ProductID()
		supported := common.IsKindleDevice(common.ProtocolUSB, vid, pid)
		log.Debug("Driver Info",
			zap.Stringer("PnP ID", id),
			zap.Bool("supported", supported),
		)

		if !supported {
			continue
		}

		if len(serial) > 0 {
			if !strings.EqualFold(serial, id.Serial()) {
				continue
			}
			// we are targeting a specific device
		} else {
			if len(result) != 0 {
				continue
			}
			// Pick first supported device
		}
		result = id
		devinst = spdd.DevInst

		var disk windows.DEVINST
		if rc, _, _ := cmGetChild.Call(uintptr(unsafe.Pointer(&disk)), uintptr(spdd.DevInst), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to find kindle disk device: %w", windows.CONFIGRET(rc))
		}
		idSize := uint32(0)
		if rc, _, _ := cmGetDeviceIDSize.Call(uintptr(unsafe.Pointer(&idSize)), uintptr(disk), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get kindle disk device ID size: %w", windows.CONFIGRET(rc))
		}
		diskID := make([]uint16, idSize+1)
		if rc, _, _ := cmGetDeviceID.Call(uintptr(disk), uintptr(unsafe.Pointer(&diskID[0])), uintptr(len(diskID)), 0); windows.CONFIGRET(rc) != windows.CR_SUCCESS {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get kindle disk device ID: %w", windows.CONFIGRET(rc))
		}

		volume, err := findCorrespondingVolume(diskID)
		if err != nil {
			return nil, "", windows.DEVINST(0), err
		}

		if volumePath, err = getVolumePath(volume); err != nil {
			return nil, "", windows.DEVINST(0), fmt.Errorf("unable to get volume root path for volume '%s': %w", windows.UTF16ToString(volume), err)
		}

		log.Debug("Device Info",
			zap.Stringer("Device ID", result),
			zap.String("VendorID", fmt.Sprintf("0x%04X", vid)),
			zap.String("ProductID", fmt.Sprintf("0x%04X", pid)),
			zap.String("Serial", id.Serial()),
			zap.String("Disk ID", windows.UTF16ToString(diskID)),
			zap.String("Volume", windows.UTF16ToString(volume)),
			zap.String("Mount path", volumePath),
		)
	}
	if len(result) == 0 || len(volumePath) == 0 {
		return nil, "", windows.DEVINST(0), common.ErrNoDevice
	}
	return result, volumePath, devinst, nil
}

func findCorrespondingVolume(parentID []uint16) ([]uint16, error) {

	guid := ole.NewGUID("53f5630d-b6bf-11d0-94f2-00a0c91efb8b") // GUID_DEVINTERFACE_VOLUME

	hDevInfo, _, err := setupDiGetClassDevs.Call(uintptr(unsafe.Pointer(guid)), uintptr(0), uintptr(0),
		uintptr(windows.DIGCF_PRESENT|windows.DIGCF_DEVICEINTERFACE))
	if windows.Handle(hDevInfo) == windows.InvalidHandle {
		return nil, fmt.Errorf("unable to enumerate volumes: %w", err)
	}
	defer setupDiDestroyDeviceInfoList.Call(hDevInfo)

	var dwSize uint32
	for dwIndex := 0; ; dwIndex++ {
		spdid := &deviceInterfaceData{}
		spdid.cbSize = uint32(unsafe.Sizeof(*spdid))

		res, _, err := setupDiEnumDeviceInterfaces.Call(hDevInfo,
			uintptr(0), uintptr(unsafe.Pointer(guid)), uintptr(dwIndex), uintptr(unsafe.Pointer(spdid)))
		if res == 0 {
			var errno syscall.Errno
			if errors.As(err, &errno) && errno == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			return nil, fmt.Errorf("unable to enumerate volume device interfaces: %w", err)
		}

		setupDiGetDeviceInterfaceDetail.Call(hDevInfo, uintptr(unsafe.Pointer(spdid)), uintptr(0), uintptr(0), uintptr(unsafe.Pointer(&dwSize)), uintptr(0))
		if dwSize <= 0 {
			continue
		}

		buf := make([]byte, dwSize)
		spdidd := (*deviceInterfaceDetailData)(unsafe.Pointer(&buf[0]))
		spdidd.cbSize = uint32(unsafe.Sizeof(spdidd))

		spdd := &devInfoData{}
		spdd.size = uint32(unsafe.Sizeof(*spdd))

		res, _, err = setupDiGetDeviceInterfaceDetail.Call(hDevInfo,
			uintptr(unsafe.Pointer(spdid)), uintptr(unsafe.Pointer(spdidd)), uintptr(dwSize), uintptr(unsafe.Pointer(&dwSize)), uintptr(unsafe.Pointer(spdd)))
		if res == 0 {
			return nil, fmt.Errorf("unable to get volume device interface details: %w", err)
		}

		mountPoint := common.UTF16PtrToUTF16(&spdidd.DevicePath[0])
		if strings.Contains(windows.UTF16ToString(mountPoint), strings.ReplaceAll(strings.ToLower(windows.UTF16ToString(parentID)), `\`, `#`)) {
			mountPoint = append(mountPoint[:len(mountPoint)-1], uint16('\\'), 0)
			volumeName := make([]uint16, windows.MAX_PATH+1)
			if err := windows.GetVolumeNameForVolumeMountPoint(&mountPoint[0], &volumeName[0], uint32(len(volumeName))); err != nil {
				return nil, fmt.Errorf("unable to get volume name for mount point '%s', %w", windows.UTF16ToString(mountPoint), err)
			}
			return common.UTF16PtrToUTF16(&volumeName[0]), nil
		}
	}
	return nil, common.ErrNoDevice
}

func getVolumePath(volume []uint16) (string, error) {
	var err error

	size := uint32(windows.MAX_PATH + 1)
	names := make([]uint16, size)

	for {
		if err = windows.GetVolumePathNamesForVolumeName(&volume[0], &names[0], size, &size); err == nil {
			break
		}
		if err != syscall.ERROR_MORE_DATA {
			return "", fmt.Errorf("unable to get volume path names for volume '%s': %w", windows.UTF16ToString(volume), err)
		}
		names = make([]uint16, size)
	}

	var paths = []string{}
	start := 0
	for end := 0; end < int(size); end++ {
		if names[end] == 0 {
			if start < end {
				paths = append(paths, windows.UTF16ToString(names[start:end]))
			}
			start = end + 1
		}
	}
	if len(paths) == 0 || len(paths) > 1 {
		return "", fmt.Errorf("ambiguous volume path names found for volume '%s' : '%+v'", windows.UTF16ToString(volume), paths)
	}
	return paths[0], nil
}
