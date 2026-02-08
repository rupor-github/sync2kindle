package usbms

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"s2k/common"
	"s2k/files"
)

type Device struct {
	*files.Device
	id    *common.PnPDeviceID
	log   *zap.Logger
	mount string
	eject bool
}

// Connect to the supported device.
func Connect(paths, serial string, eject bool, log *zap.Logger) (*Device, error) {

	id, mount, err := pickDevice(serial, log)
	if err != nil {
		return nil, err
	}

	d := &Device{log: log.Named(driverName), id: id, mount: mount, eject: eject}
	d.Device, err = files.Connect(paths, filepath.ToSlash(mount), nil, d.log)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// driver interface

func (d *Device) Disconnect() {
	if d != nil && d.eject {
		if err := unix.Unmount(d.mount, unix.MNT_DETACH); err != nil {
			d.log.Error("Unable to unmount device", zap.String("mount", d.mount), zap.Error(err))
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

type deviceDetails struct {
	Path, Volume, Mount string
	Capacity            int64 // for compatibility always reported in 512 bytes blocks
}

func pickDevice(serial string, log *zap.Logger) (*common.PnPDeviceID, string, error) {
	var (
		usbIDs *common.PnPDeviceID
		mount  string
	)
	if err := filepath.Walk("/sys/devices", func(usbPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if strings.HasSuffix(usbPath, "idVendor") {
			devPath := filepath.Dir(usbPath)
			var (
				vid, pid, bcd int64
				sn            string
			)
			for p, f := range map[string]func(string) error{
				filepath.Join(devPath, "idVendor"):  common.FromSysfsNumber(&vid, 16),
				filepath.Join(devPath, "idProduct"): common.FromSysfsNumber(&pid, 16),
				filepath.Join(devPath, "bcdDevice"): common.FromSysfsNumber(&bcd, 16), // version as major/minor (binary coded decimal from usb descriptor)
				filepath.Join(devPath, "serial"):    common.FromSysfsString(&sn),
			} {
				if err := unix.Access(p, unix.R_OK); err != nil {
					return nil
				}
				if err := f(p); err != nil {
					return err
				}
			}
			devIDs := common.NewPnPDeviceID(int(vid), int(pid), int(bcd), sn)

			supported := common.IsKindleDevice(common.ProtocolUSB, devIDs.VendorID(), devIDs.ProductID())
			log.Debug("Driver Info",
				zap.Stringer("PnP ID", devIDs),
				zap.Bool("supported", supported),
			)

			if !supported {
				return nil
			}

			if len(serial) > 0 {
				if !strings.EqualFold(serial, devIDs.Serial()) {
					return nil
				}
				// we are targeting a specific device
			} else {
				if !usbIDs.Empty() {
					return nil
				}
				// pick the first supported device
			}
			usbIDs = devIDs

			var name, mfr string
			if err := common.FromSysfsString(&name)(filepath.Join(devPath, "product")); err != nil {
				return err
			}
			if err := common.FromSysfsString(&mfr)(filepath.Join(devPath, "manufacturer")); err != nil {
				return err
			}

			details, err := getVolumeDetails(devPath)
			if err != nil {
				return fmt.Errorf("unable to get volume details for '%s': %w", devPath, err)
			}
			mount = details.Mount

			var stat unix.Statfs_t
			if err := unix.Statfs(details.Mount, &stat); err != nil {
				return fmt.Errorf("unable to get file system stats for '%s': %w", details.Mount, err)
			}

			log.Debug("Device Info",
				zap.String("Name", name),
				zap.String("Manufacturer", mfr),
				zap.Stringer("Device ID", usbIDs),
				zap.Any("Details", details),
				zap.String("Available bytes", humanize.Comma(details.Capacity*512)),
				zap.String("Free bytes", humanize.Comma(int64(stat.Bavail)*stat.Bsize)),
				zap.Bool("supported", supported),
			)
		}
		return nil
	}); err != nil {
		return nil, "", err
	}

	if usbIDs.Empty() || len(mount) == 0 {
		return nil, "", common.ErrNoDevice
	}
	return usbIDs, mount, nil
}

func getVolumeDetails(root string) (*deviceDetails, error) {
	var dd deviceDetails
	if err := filepath.Walk(root, func(usbPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.Contains(usbPath, "/block/") {
			return nil
		}
		parts := strings.Split(usbPath, "/")
		if i := slices.Index(parts, "block"); i == len(parts)-2 {

			// So far all Kindles have a single accessible partition.
			part := parts[i+1] + "1"
			dd.Path = filepath.Join(usbPath, part)

			if err := common.FromSysfsNumber(&dd.Capacity, 10)(filepath.Join(dd.Path, "size")); err != nil {
				return err
			}
			dd.Volume = filepath.Join("/dev", part)

			var err error
			dd.Mount, err = getVolumePath(dd.Volume)
			if err != nil {
				return err
			}
			return filepath.SkipAll
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return &dd, nil
}

func getVolumePath(volume string) (string, error) {
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("unable to read mounts: %w", err)
	}
	defer mounts.Close()

	sc := bufio.NewScanner(mounts)
	for sc.Scan() {
		flds := strings.Fields(sc.Text())
		if len(flds) >= 2 && flds[0] == volume {
			return flds[1], nil
		}
	}
	return "", fmt.Errorf("unable to find mount path for volume '%s'", volume)
}
