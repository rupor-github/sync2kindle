//go:build usb && darwin

package usbms

// #cgo LDFLAGS: -framework CoreFoundation -framework DiskArbitration -framework IOKit
// #include <CoreFoundation/CoreFoundation.h>
// #include <DiskArbitration/DiskArbitration.h>
// #include <IOKit/IOKitLib.h>
// #include <stdio.h>
// #include <stdlib.h>
// #include <string.h>
// #include <sys/mount.h>
//
// typedef struct {
//     int vid;
//     int pid;
//     char *serial;
//     char *product;
//     char *manufacturer;
//     char *bsd_name;
//     char *mount;
// } s2k_darwin_usbms_device;
//
// typedef struct {
//     int done;
//     int status;
//     char *message;
//     CFRunLoopRef run_loop;
// } s2k_da_context;
//
// static void s2k_set_error(char **err, const char *msg) {
//     if (err != NULL && msg != NULL) {
//         *err = strdup(msg);
//     }
// }
//
// static char *s2k_copy_cfstring(CFTypeRef value) {
//     if (value == NULL || CFGetTypeID(value) != CFStringGetTypeID()) {
//         return NULL;
//     }
//     CFStringRef str = (CFStringRef)value;
//     CFIndex len = CFStringGetLength(str);
//     CFIndex max = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
//     char *buf = (char *)malloc((size_t)max);
//     if (buf == NULL) {
//         return NULL;
//     }
//     if (!CFStringGetCString(str, buf, max, kCFStringEncodingUTF8)) {
//         free(buf);
//         return NULL;
//     }
//     return buf;
// }
//
// static int s2k_copy_cfnumber_int(CFTypeRef value, int *out) {
//     if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
//         return 0;
//     }
//     return CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, out);
// }
//
// static CFTypeRef s2k_search_property(io_registry_entry_t entry, CFStringRef key) {
//     return IORegistryEntrySearchCFProperty(
//         entry,
//         kIOServicePlane,
//         key,
//         kCFAllocatorDefault,
//         kIORegistryIterateRecursively | kIORegistryIterateParents
//     );
// }
//
// static int s2k_search_int_property(io_registry_entry_t entry, CFStringRef key, int *out) {
//     CFTypeRef value = s2k_search_property(entry, key);
//     if (value == NULL) {
//         return 0;
//     }
//     int ok = s2k_copy_cfnumber_int(value, out);
//     CFRelease(value);
//     return ok;
// }
//
// static char *s2k_search_string_property(io_registry_entry_t entry, CFStringRef key) {
//     CFTypeRef value = s2k_search_property(entry, key);
//     if (value == NULL) {
//         return NULL;
//     }
//     char *result = s2k_copy_cfstring(value);
//     CFRelease(value);
//     return result;
// }
//
// static char *s2k_first_string_property(io_registry_entry_t entry, CFStringRef *keys, int count) {
//     for (int i = 0; i < count; i++) {
//         char *value = s2k_search_string_property(entry, keys[i]);
//         if (value != NULL && value[0] != '\0') {
//             return value;
//         }
//         free(value);
//     }
//     return NULL;
// }
//
// static void s2k_free_darwin_usbms_device(s2k_darwin_usbms_device *device) {
//     if (device == NULL) {
//         return;
//     }
//     free(device->serial);
//     free(device->product);
//     free(device->manufacturer);
//     free(device->bsd_name);
//     free(device->mount);
// }
//
// void s2k_free_darwin_usbms_devices(s2k_darwin_usbms_device *devices, int count) {
//     if (devices == NULL) {
//         return;
//     }
//     for (int i = 0; i < count; i++) {
//         s2k_free_darwin_usbms_device(&devices[i]);
//     }
//     free(devices);
// }
//
// int s2k_list_darwin_usbms_devices(s2k_darwin_usbms_device **out, int *count, char **err) {
//     *out = NULL;
//     *count = 0;
//
//     struct statfs *mounts = NULL;
//     int mount_count = getmntinfo(&mounts, MNT_NOWAIT);
//     if (mount_count < 0) {
//         s2k_set_error(err, "unable to enumerate mounted filesystems");
//         return -1;
//     }
//
//     DASessionRef session = DASessionCreate(kCFAllocatorDefault);
//     if (session == NULL) {
//         s2k_set_error(err, "unable to create Disk Arbitration session");
//         return -1;
//     }
//
//     s2k_darwin_usbms_device *devices = NULL;
//     int device_count = 0;
//
//     for (int i = 0; i < mount_count; i++) {
//         const char *from = mounts[i].f_mntfromname;
//         if (strncmp(from, "/dev/", 5) != 0) {
//             continue;
//         }
//
//         const char *bsd_name = from + 5;
//         DADiskRef disk = DADiskCreateFromBSDName(kCFAllocatorDefault, session, bsd_name);
//         if (disk == NULL) {
//             continue;
//         }
//
//         io_service_t media = DADiskCopyIOMedia(disk);
//         if (media == IO_OBJECT_NULL) {
//             CFRelease(disk);
//             continue;
//         }
//
//         int vid = 0;
//         int pid = 0;
//         int have_vid = s2k_search_int_property(media, CFSTR("idVendor"), &vid);
//         int have_pid = s2k_search_int_property(media, CFSTR("idProduct"), &pid);
//         if (!have_vid || !have_pid) {
//             IOObjectRelease(media);
//             CFRelease(disk);
//             continue;
//         }
//
//         CFStringRef serial_keys[] = {CFSTR("USB Serial Number"), CFSTR("Serial Number"), CFSTR("serial")};
//         CFStringRef product_keys[] = {CFSTR("USB Product Name"), CFSTR("Product Name"), CFSTR("product")};
//         CFStringRef manufacturer_keys[] = {CFSTR("USB Vendor Name"), CFSTR("USB Vendor String"), CFSTR("Manufacturer"), CFSTR("manufacturer")};
//
//         s2k_darwin_usbms_device *next = (s2k_darwin_usbms_device *)realloc(
//             devices,
//             sizeof(s2k_darwin_usbms_device) * (size_t)(device_count + 1)
//         );
//         if (next == NULL) {
//             IOObjectRelease(media);
//             CFRelease(disk);
//             s2k_free_darwin_usbms_devices(devices, device_count);
//             s2k_set_error(err, "unable to allocate USB device list");
//             return -1;
//         }
//         devices = next;
//
//         s2k_darwin_usbms_device *device = &devices[device_count];
//         memset(device, 0, sizeof(*device));
//         device->vid = vid;
//         device->pid = pid;
//         device->serial = s2k_first_string_property(media, serial_keys, 3);
//         device->product = s2k_first_string_property(media, product_keys, 3);
//         device->manufacturer = s2k_first_string_property(media, manufacturer_keys, 4);
//         device->bsd_name = strdup(bsd_name);
//         device->mount = strdup(mounts[i].f_mntonname);
//
//         if (device->bsd_name == NULL || device->mount == NULL) {
//             IOObjectRelease(media);
//             CFRelease(disk);
//             s2k_free_darwin_usbms_devices(devices, device_count + 1);
//             s2k_set_error(err, "unable to allocate USB device details");
//             return -1;
//         }
//
//         device_count++;
//
//         IOObjectRelease(media);
//         CFRelease(disk);
//     }
//
//     CFRelease(session);
//     *out = devices;
//     *count = device_count;
//     return 0;
// }
//
// static void s2k_da_callback(DADiskRef disk, DADissenterRef dissenter, void *ctx) {
//     s2k_da_context *context = (s2k_da_context *)ctx;
//     if (dissenter != NULL) {
//         context->status = DADissenterGetStatus(dissenter);
//         context->message = s2k_copy_cfstring(DADissenterGetStatusString(dissenter));
//     }
//     context->done = 1;
//     CFRunLoopStop(context->run_loop);
// }
//
// static int s2k_wait_for_da_operation(DASessionRef session, DADiskRef disk, int eject, char **err) {
//     s2k_da_context context;
//     memset(&context, 0, sizeof(context));
//     context.run_loop = CFRunLoopGetCurrent();
//
//     DASessionScheduleWithRunLoop(session, context.run_loop, kCFRunLoopDefaultMode);
//     if (eject) {
//         DADiskEject(disk, kDADiskEjectOptionDefault, s2k_da_callback, &context);
//     } else {
//         DADiskUnmount(disk, kDADiskUnmountOptionDefault, s2k_da_callback, &context);
//     }
//     while (!context.done) {
//         CFRunLoopRun();
//     }
//     DASessionUnscheduleFromRunLoop(session, context.run_loop, kCFRunLoopDefaultMode);
//
//     if (context.status != 0) {
//         if (context.message != NULL) {
//             s2k_set_error(err, context.message);
//             free(context.message);
//         } else {
//             char message[128];
//             snprintf(message, sizeof(message), "Disk Arbitration operation failed with status %d", context.status);
//             s2k_set_error(err, message);
//         }
//         return -1;
//     }
//     free(context.message);
//     return 0;
// }
//
// int s2k_eject_darwin_disk(const char *bsd_name, char **err) {
//     DASessionRef session = DASessionCreate(kCFAllocatorDefault);
//     if (session == NULL) {
//         s2k_set_error(err, "unable to create Disk Arbitration session");
//         return -1;
//     }
//
//     DADiskRef disk = DADiskCreateFromBSDName(kCFAllocatorDefault, session, bsd_name);
//     if (disk == NULL) {
//         CFRelease(session);
//         s2k_set_error(err, "unable to create disk reference");
//         return -1;
//     }
//     DADiskRef whole_disk = DADiskCopyWholeDisk(disk);
//
//     int result = s2k_wait_for_da_operation(session, disk, 0, err);
//     if (result == 0) {
//         result = s2k_wait_for_da_operation(session, whole_disk != NULL ? whole_disk : disk, 1, err);
//     }
//
//     if (whole_disk != NULL) {
//         CFRelease(whole_disk);
//     }
//     CFRelease(disk);
//     CFRelease(session);
//     return result;
// }
import "C"

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	humanize "github.com/dustin/go-humanize"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"s2k/common"
	"s2k/files"
)

type Device struct {
	*files.Device
	id      *common.PnPDeviceID
	log     *zap.Logger
	bsdName string
	mount   string
	eject   bool
}

type deviceDetails struct {
	BSDName, Mount        string
	Product, Manufacturer string
	Capacity              int64
}

func Connect(paths, serial string, eject bool, log *zap.Logger) (*Device, error) {
	id, details, err := pickDevice(serial, log)
	if err != nil {
		return nil, err
	}

	d := &Device{log: log.Named(driverName), id: id, bsdName: details.BSDName, mount: details.Mount, eject: eject}
	d.Device, err = files.Connect(paths, filepath.ToSlash(details.Mount), nil, d.log)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Device) Disconnect() {
	if d == nil || !d.eject {
		return
	}
	if err := ejectDisk(d.bsdName); err != nil {
		d.log.Error("Eject failed", zap.String("disk", d.bsdName), zap.String("mount", d.mount), zap.Error(err))
	}
}

func (d *Device) Name() string {
	return driverName
}

func (d *Device) UniqueID() string {
	return d.id.Serial()
}

func pickDevice(serial string, log *zap.Logger) (*common.PnPDeviceID, *deviceDetails, error) {
	devices, err := listDarwinUSBMSDevices()
	if err != nil {
		return nil, nil, err
	}

	var (
		usbIDs  *common.PnPDeviceID
		details *deviceDetails
	)
	for _, device := range devices {
		devIDs := common.NewPnPDeviceID(device.vid, device.pid, 0, device.serial)
		supported := common.IsKindleDevice(common.ProtocolUSB, devIDs.VendorID(), devIDs.ProductID())
		log.Debug("Driver Info",
			zap.Stringer("PnP ID", devIDs),
			zap.Bool("supported", supported),
		)

		if !supported {
			continue
		}

		if len(serial) > 0 {
			if !strings.EqualFold(serial, devIDs.Serial()) {
				continue
			}
		} else if !usbIDs.Empty() {
			continue
		}

		nextDetails, err := getVolumeDetails(device)
		if err != nil {
			return nil, nil, err
		}

		usbIDs = devIDs
		details = nextDetails

		log.Debug("Device Info",
			zap.String("Name", details.Product),
			zap.String("Manufacturer", details.Manufacturer),
			zap.Stringer("Device ID", usbIDs),
			zap.Any("Details", details),
			zap.String("Available bytes", humanize.Comma(details.Capacity)),
			zap.Bool("supported", supported),
		)
	}

	if usbIDs.Empty() || details == nil || len(details.Mount) == 0 {
		return nil, nil, common.ErrNoDevice
	}
	return usbIDs, details, nil
}

type darwinUSBMSDevice struct {
	vid, pid              int
	serial, product       string
	manufacturer, bsdName string
	mount                 string
}

func listDarwinUSBMSDevices() ([]darwinUSBMSDevice, error) {
	var (
		cDevices *C.s2k_darwin_usbms_device
		cCount   C.int
		cErr     *C.char
	)
	if rc := C.s2k_list_darwin_usbms_devices(&cDevices, &cCount, &cErr); rc != 0 {
		defer C.free(unsafe.Pointer(cErr))
		return nil, fmt.Errorf("unable to enumerate USB mass storage devices: %s", C.GoString(cErr))
	}
	defer C.s2k_free_darwin_usbms_devices(cDevices, cCount)
	if cCount == 0 {
		return nil, nil
	}

	devices := make([]darwinUSBMSDevice, 0, int(cCount))
	for _, cDevice := range unsafe.Slice(cDevices, int(cCount)) {
		devices = append(devices, darwinUSBMSDevice{
			vid:          int(cDevice.vid),
			pid:          int(cDevice.pid),
			serial:       C.GoString(cDevice.serial),
			product:      C.GoString(cDevice.product),
			manufacturer: C.GoString(cDevice.manufacturer),
			bsdName:      C.GoString(cDevice.bsd_name),
			mount:        C.GoString(cDevice.mount),
		})
	}
	return devices, nil
}

func getVolumeDetails(device darwinUSBMSDevice) (*deviceDetails, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(device.mount, &stat); err != nil {
		return nil, fmt.Errorf("unable to get file system stats for '%s': %w", device.mount, err)
	}

	return &deviceDetails{
		BSDName:      device.bsdName,
		Mount:        device.mount,
		Product:      device.product,
		Manufacturer: device.manufacturer,
		Capacity:     int64(stat.Blocks) * int64(stat.Bsize),
	}, nil
}

func ejectDisk(bsdName string) error {
	cBSDName := C.CString(bsdName)
	defer C.free(unsafe.Pointer(cBSDName))

	var cErr *C.char
	if rc := C.s2k_eject_darwin_disk(cBSDName, &cErr); rc != 0 {
		defer C.free(unsafe.Pointer(cErr))
		return fmt.Errorf("%s", C.GoString(cErr))
	}
	return nil
}
