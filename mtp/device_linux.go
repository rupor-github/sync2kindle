//go:build mtp && linux

package mtp

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"s2k/common"
)

func getPlatformSerialNumber(srcvid, srcpid, srcbus, srcdev int) (string, error) {
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
