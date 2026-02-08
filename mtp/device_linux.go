package mtp

// #cgo pkg-config: libmtp
// #include <libmtp.h>
// #include <stdlib.h>
// #include <stdio.h>
// #include <unistd.h>
// #include <fcntl.h>
//
// // redirect stderr to nul
// int null_stderr() {
//     int backup, replace;
//     fflush(stderr);
//     backup = dup(STDERR_FILENO);
//     replace = open("/dev/null", O_WRONLY);
//     dup2(replace, STDERR_FILENO);
//     close(replace);
//     return backup;
// }
//
// // restore stderr
// void restore_stderr(int backup) {
//     fflush(stderr);
//     dup2(backup, STDERR_FILENO);
//     close(backup);
// }
//
import "C"

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"s2k/common"
	"s2k/objects"
)

type Device struct {
	log   *zap.Logger
	id    *common.PnPDeviceID
	dev   *C.LIBMTP_mtpdevice_t
	roots []string
}

// Connect to the supported device.
func Connect(paths, serial string, verbose bool, log *zap.Logger) (*Device, error) {
	C.LIBMTP_Init()

	if !verbose {
		// NOTE: when verbose specified we may look into setting up different debugging levels later if needed
		C.LIBMTP_Set_Debug(C.LIBMTP_DEBUG_NONE)
	}

	id, dev, err := pickDevice(serial, verbose, log)
	if err != nil {
		return nil, err
	}

	// So far all Kindles have a single storage
	info := propSet{
		"Storage Description": C.GoString(dev.storage.StorageDescription),
		"Storage Type":        WPDStorageType(dev.storage.StorageType).String(),
		"File System Type":    WPDStorageFileSystemType(dev.storage.FilesystemType).String(),
		"Storage Access":      WPDStorageAccessCapability(dev.storage.AccessCapability).String(),
		"Storage Free Space":  WPDStorageBytes(dev.storage.FreeSpaceInBytes).String(),
		"Storage Capacity":    WPDStorageBytes(dev.storage.MaxCapacity).String(),
	}
	log.Debug("Device Storage", zap.Any("Properties", info))

	d := &Device{
		log: log.Named(driverName),
		id:  id,
		dev: dev,
	}
	if WPDStorageAccessCapability(dev.storage.AccessCapability) != WPD_STORAGE_ACCESS_CAPABILITY_READWRITE {
		return nil, common.ErrNoAccess
	}
	// prepare filters for the device - has to be after we know selected storage root
	ps := filepath.SplitList(paths)
	for _, p := range ps {
		if !slices.Contains(d.roots, p) {
			d.roots = append(d.roots, p)
		}
	}
	d.log.Debug("Device paths of interest", zap.Any("Roots", d.roots))

	return d, nil
}

// driver interface

func (d *Device) Disconnect() {
	if d == nil {
		return
	}
	if d.dev != nil {
		C.LIBMTP_Release_Device(d.dev)
	}
}

func (d *Device) Name() string {
	return driverName
}

func (d *Device) UniqueID() string {
	return d.id.Serial()
}

func (d *Device) MkDir(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("MkDir is called with nil object")
	}
	parent := obj.OIS.Find(path.Dir(obj.FullPath))
	if parent == nil {
		return fmt.Errorf("parent object not found for '%s'", obj.FullPath)
	}
	obj.OidParent = parent.Oid

	defer func(start time.Time) {
		d.log.Debug("Executed action MkDir", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	name := C.CString(obj.Name)
	defer C.free(unsafe.Pointer(name))

	id := C.LIBMTP_Create_Folder(d.dev, name, C.uint32_t(obj.OidParent), d.dev.storage.id)
	if id == 0 {
		return fmt.Errorf("failed to create folder '%s': %w", obj.Name, d.getErrors())
	}
	obj.Oid = objects.ObjectID(id)
	return nil
}

func (d *Device) Remove(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Remove is called with nil object")
	}

	defer func(start time.Time) {
		d.log.Debug("Executed action Remove", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	if res := C.LIBMTP_Delete_Object(d.dev, C.uint32_t(obj.Oid)); res != 0 {
		return fmt.Errorf("failed to delete object '%s': %w", obj.Oid, d.getErrors())
	}
	return nil
}

func (d *Device) Copy(obj *objects.ObjectInfo) (err error) {
	if obj == nil {
		panic("Copy is called with nil object")
	}
	parent := obj.OIS.Find(path.Dir(obj.FullPath))
	if parent == nil {
		return fmt.Errorf("parent object not found for '%s'", obj.FullPath)
	}
	obj.OidParent = parent.Oid

	defer func(start time.Time) {
		d.log.Debug("Executed action Copy", zap.String("actor", d.Name()), zap.Any("object", obj), zap.Duration("elapsed", time.Since(start)), zap.Error(err))
	}(time.Now())

	target := C.LIBMTP_new_file_t()
	defer C.LIBMTP_destroy_file_t(target)

	target.filesize = C.uint64_t(obj.ObjSize)
	target.filename = C.CString(obj.Name)
	target.filetype = C.LIBMTP_FILETYPE_UNKNOWN
	target.parent_id = C.uint32_t(obj.OidParent)
	target.storage_id = d.dev.storage.id
	target.modificationdate = C.time_t(time.Now().Unix())

	from := C.CString(obj.ObjectName)
	defer C.free(unsafe.Pointer(from))

	if res := C.LIBMTP_Send_File_From_File(d.dev, from, target, nil, nil); res != 0 {
		return fmt.Errorf("failed to copy file '%s' (%d) to '%s': %w", obj.ObjectName, obj.ObjSize, obj.FullPath, d.getErrors())
	}
	obj.Oid = objects.ObjectID(target.item_id)
	return nil
}

func (d *Device) GetObjectInfos() (objects.ObjectInfoSet, error) {
	infos := d.enumerateObjects(WPD_DEVICE_OBJECT_ID, "", make([]*objects.ObjectInfo, 0))
	if len(infos) == 0 {
		if err := d.getErrors(); err != nil {
			return nil, err
		}
		return nil, common.ErrNoObjects
	}

	// index the results by full path, loosing target directories
	oset := objects.New()
	for _, info := range infos {
		oset[info.FullPath] = info
	}
	return oset, nil
}

// implementation

func (d *Device) getErrors() (err error) {
	if d == nil || d.dev == nil {
		return
	}
	for stack := C.LIBMTP_Get_Errorstack(d.dev); stack != nil; stack = stack.next {
		if err != nil {
			err = fmt.Errorf("%s(%d): %w", C.GoString(stack.error_text), stack.errornumber, err)
		} else {
			err = fmt.Errorf("%s (%d)", C.GoString(stack.error_text), stack.errornumber)
		}
	}
	C.LIBMTP_Clear_Errorstack(d.dev)
	return
}

func (d *Device) enumerateObjects(parent objects.ObjectID, root string, infos []*objects.ObjectInfo) []*objects.ObjectInfo {
	objs := C.LIBMTP_Get_Files_And_Folders(d.dev, d.dev.storage.id, C.uint32_t(parent))
	if objs == nil {
		return infos
	}

	var prev *C.LIBMTP_file_t
	for obj := objs; obj != nil; obj = obj.next {
		info := getObjectInfo(obj)
		info.FullPath = filepath.Join(root, info.Name)

		cont := false
		for _, r := range d.roots {
			if strings.HasPrefix(info.FullPath, r) || strings.HasPrefix(r, info.FullPath) {
				cont = true
				break
			}
		}

		// to save time we only drill down and keep objects under paths of interest
		if !cont {
			continue
		}

		if exists := slices.ContainsFunc(infos, func(oi *objects.ObjectInfo) bool {
			return oi.Oid == info.Oid
		}); exists {
			d.log.Warn("Object already in map, ignoring", zap.String("root", root), zap.Stringer("obj", info.Oid))
		} else {
			infos = append(infos, info)
		}
		if info.Dir {
			// recurse into the directory
			infos = d.enumerateObjects(info.Oid, info.FullPath, infos)
		}

		if prev != nil {
			C.LIBMTP_destroy_file_t(prev)
		}
		prev = obj
	}
	if prev != nil {
		C.LIBMTP_destroy_file_t(prev)
	}
	return infos
}

func getObjectInfo(obj *C.LIBMTP_file_t) *objects.ObjectInfo {
	return &objects.ObjectInfo{
		Name:       C.GoString(obj.filename),
		Modified:   time.Unix(int64(obj.modificationdate), 0),
		File:       obj.filetype != C.LIBMTP_FILETYPE_FOLDER,
		Dir:        obj.filetype == C.LIBMTP_FILETYPE_FOLDER,
		Oid:        objects.ObjectID(obj.item_id),
		OidParent:  objects.ObjectID(obj.parent_id),
		ObjSize:    int64(obj.filesize), // Do not think overflow is ever an issue here given device memory size
		ObjectName: C.GoString(obj.filename),
	}
}

func pickDevice(serial string, verbose bool, log *zap.Logger) (usbIDs *common.PnPDeviceID, dev *C.LIBMTP_mtpdevice_t, err error) {
	defer func() {
		if err != nil && dev != nil {
			C.LIBMTP_Release_Device(dev)
		}
	}()

	var (
		rawdevs       *C.LIBMTP_raw_device_t
		numdevs, save C.int
	)

	if !verbose {
		// up to version of 1.1.22 libmtp did not know about the new MTP Kindles and on many
		// distros its version way older than that. It will complain in terminal and there
		//is no proper way to shut it because of library design, so...
		save = C.null_stderr()
	}
	errno := C.LIBMTP_Detect_Raw_Devices(&rawdevs, &numdevs)
	defer C.free(unsafe.Pointer(rawdevs))
	if !verbose {
		defer C.restore_stderr(save)
	}

	switch errno {
	case C.LIBMTP_ERROR_NO_DEVICE_ATTACHED:
		return nil, nil, common.ErrNoDevice
	case C.LIBMTP_ERROR_CONNECTING:
		return nil, nil, errors.New("libmtp connection error")
	case C.LIBMTP_ERROR_MEMORY_ALLOCATION:
		return nil, nil, unix.ENOMEM
	case C.LIBMTP_ERROR_NONE:
	default:
		return nil, nil, errors.New("libmtp failed to detect raw MTP devices")
	}

	rdevs := unsafe.Slice(rawdevs, int(numdevs))
	for i := 0; i < int(numdevs); i++ {
		candidate := C.LIBMTP_Open_Raw_Device_Uncached(&rdevs[i])
		if candidate != nil {

			vid, pid, bus, devnum := int(rdevs[i].device_entry.vendor_id),
				int(rdevs[i].device_entry.product_id),
				int(rdevs[i].bus_location),
				int(rdevs[i].devnum)

			var sn string
			sn, err = getSerialNumber(vid, pid, bus, devnum)
			if err != nil {
				C.LIBMTP_Release_Device(candidate)
				return nil, nil, fmt.Errorf("libmtp failed to get serial number for device '%04X:%04X:%02X:%02X': %w",
					vid, pid, bus, devnum, err)
			}

			devIDs := common.NewPnPDeviceID(vid, pid, bus&0xFF<<8+devnum&0xff, sn)

			supported := common.IsKindleDevice(common.ProtocolMTP, vid, pid)
			log.Debug("Driver Info",
				zap.Stringer("PnP ID", devIDs),
				zap.Bool("supported", supported),
			)

			if !supported {
				C.LIBMTP_Release_Device(candidate)
				continue
			}

			if len(serial) > 0 {
				if !strings.EqualFold(serial, devIDs.Serial()) {
					C.LIBMTP_Release_Device(candidate)
					continue
				}
				// we are targeting a specific device
			} else {
				if !usbIDs.Empty() {
					C.LIBMTP_Release_Device(candidate)
					continue
				}
				// pick the first supported device
			}

			// Release previously selected device if any
			if dev != nil {
				C.LIBMTP_Release_Device(dev)
			}
			dev = candidate
			usbIDs = devIDs

			name := C.LIBMTP_Get_Friendlyname(dev)
			defer C.free(unsafe.Pointer(name))

			descr := C.LIBMTP_Get_Modelname(dev)
			defer C.free(unsafe.Pointer(descr))

			mfr := C.LIBMTP_Get_Manufacturername(dev)
			defer C.free(unsafe.Pointer(mfr))

			log.Debug("Device Info",
				zap.Stringer("Device ID", devIDs),
				zap.String("Name", C.GoString(name)),
				zap.String("Description", C.GoString(descr)),
				zap.String("Manufacturer", C.GoString(mfr)),
				zap.String("VendorID", fmt.Sprintf("0x%04X", vid)),
				zap.String("ProductID", fmt.Sprintf("0x%04X", pid)),
				zap.String("Serial", sn),
				zap.Bool("supported", supported),
			)
		}
	}
	if usbIDs.Empty() {
		return nil, nil, common.ErrNoDevice
	}
	return
}

func getSerialNumber(srcvid, srcpid, srcbus, srcdev int) (string, error) {
	root := "/sys/bus/usb/devices"

	sysfs, err := os.Open(root)
	if err != nil {
		return "", err
	}
	defer sysfs.Close()

	entries, err := sysfs.ReadDir(0)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		// should only be symlinks
		if fi, err := entry.Info(); err != nil {
			return "", err
		} else if fi.Mode().IsRegular() {
			continue
		}
		realPath, err := filepath.EvalSymlinks(filepath.Join(root, entry.Name()))
		if err != nil {
			return "", err
		}
		if serial, err := findSNForConnectedDevice(realPath, srcvid, srcpid, srcbus, srcdev); err != nil {
			return "", err
		} else if len(serial) != 0 {
			// we got it
			return serial, nil
		}
	}
	return "", nil
}

func findSNForConnectedDevice(dir string, srcvid, srcpid, srcbus, srcdev int) (string, error) {
	var result string

	if err := filepath.Walk(dir, func(usbPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if strings.HasSuffix(usbPath, "idVendor") {
			devPath := filepath.Dir(usbPath)
			var (
				vid, pid, busnum, devnum int64
				serial                   string
			)
			for p, f := range map[string]func(string) error{
				filepath.Join(devPath, "idVendor"):  common.FromSysfsNumber(&vid, 16),
				filepath.Join(devPath, "idProduct"): common.FromSysfsNumber(&pid, 16),
				filepath.Join(devPath, "busnum"):    common.FromSysfsNumber(&busnum, 16),
				filepath.Join(devPath, "devnum"):    common.FromSysfsNumber(&devnum, 16),
				filepath.Join(devPath, "serial"):    common.FromSysfsString(&serial),
			} {
				if err := unix.Access(p, unix.R_OK); err != nil {
					return nil
				}
				if err := f(p); err != nil {
					return err
				}
			}
			if srcvid == int(vid) && srcpid == int(pid) &&
				srcbus == int(busnum) && srcdev == int(devnum) {
				// found
				result = serial
				return filepath.SkipAll
			}
		}
		return nil
	}); err != nil {
		return "", err
	}
	return result, nil
}

func init() {
	// initialize WPD_DEVICE_OBJECT_ID for global usage
	WPD_DEVICE_OBJECT_ID = C.LIBMTP_FILES_AND_FOLDERS_ROOT
}
