//go:build mtp && darwin

package mtp

// #cgo LDFLAGS: -framework CoreFoundation -framework IOKit
// #include <CoreFoundation/CoreFoundation.h>
// #include <IOKit/IOKitLib.h>
// #include <stdlib.h>
// #include <string.h>
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
// static CFTypeRef s2k_copy_property(io_registry_entry_t entry, CFStringRef key) {
//     return IORegistryEntryCreateCFProperty(entry, key, kCFAllocatorDefault, 0);
// }
//
// static int s2k_copy_int_property(io_registry_entry_t entry, CFStringRef key, int *out) {
//     CFTypeRef value = s2k_copy_property(entry, key);
//     if (value == NULL) {
//         return 0;
//     }
//     int ok = s2k_copy_cfnumber_int(value, out);
//     CFRelease(value);
//     return ok;
// }
//
// static char *s2k_copy_string_property(io_registry_entry_t entry, CFStringRef key) {
//     CFTypeRef value = s2k_copy_property(entry, key);
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
//         char *value = s2k_copy_string_property(entry, keys[i]);
//         if (value != NULL && value[0] != '\0') {
//             return value;
//         }
//         free(value);
//     }
//     return NULL;
// }
//
// static int s2k_darwin_mtp_serial(int srcvid, int srcpid, int srcdev, char **serial, char **err) {
//     *serial = NULL;
//
//     CFMutableDictionaryRef matching = IOServiceMatching("IOUSBHostDevice");
//     if (matching == NULL) {
//         s2k_set_error(err, "unable to create IOKit USB matching dictionary");
//         return -1;
//     }
//
//     io_iterator_t iterator = IO_OBJECT_NULL;
//     kern_return_t kr = IOServiceGetMatchingServices(kIOMainPortDefault, matching, &iterator);
//     if (kr != KERN_SUCCESS) {
//         s2k_set_error(err, "unable to enumerate IOKit USB devices");
//         return -1;
//     }
//
//     int matches = 0;
//     io_service_t device;
//     while ((device = IOIteratorNext(iterator)) != IO_OBJECT_NULL) {
//         int vid = 0;
//         int pid = 0;
//         int address = 0;
//         int have_vid = s2k_copy_int_property(device, CFSTR("idVendor"), &vid);
//         int have_pid = s2k_copy_int_property(device, CFSTR("idProduct"), &pid);
//         if (!have_vid || !have_pid || vid != srcvid || pid != srcpid) {
//             IOObjectRelease(device);
//             continue;
//         }
//         if (srcdev != 0 && s2k_copy_int_property(device, CFSTR("USB Address"), &address) && address != srcdev) {
//             IOObjectRelease(device);
//             continue;
//         }
//
//         CFStringRef serial_keys[] = {CFSTR("USB Serial Number"), CFSTR("kUSBSerialNumberString"), CFSTR("serial")};
//         char *candidate = s2k_first_string_property(device, serial_keys, 3);
//         IOObjectRelease(device);
//         if (candidate == NULL) {
//             continue;
//         }
//
//         matches++;
//         if (matches == 1) {
//             *serial = candidate;
//         } else {
//             free(candidate);
//         }
//     }
//
//     IOObjectRelease(iterator);
//     if (matches > 1) {
//         free(*serial);
//         *serial = NULL;
//     }
//     return 0;
// }
import "C"

import (
	"fmt"
	"unsafe"
)

func getPlatformSerialNumber(vid, pid, bus, dev int) (string, error) {
	var (
		serial *C.char
		cErr   *C.char
	)
	_ = bus
	if rc := C.s2k_darwin_mtp_serial(C.int(vid), C.int(pid), C.int(dev), &serial, &cErr); rc != 0 {
		defer C.free(unsafe.Pointer(cErr))
		return "", fmt.Errorf("IOKit serial lookup failed: %s", C.GoString(cErr))
	}
	defer C.free(unsafe.Pointer(serial))
	return C.GoString(serial), nil
}
