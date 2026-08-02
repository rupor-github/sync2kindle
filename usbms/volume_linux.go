//go:build linux

package usbms

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

func findVolumePathInMountInfo(mountInfo io.Reader, deviceID string) (string, error) {
	sc := bufio.NewScanner(mountInfo)
	for sc.Scan() {
		flds := strings.Fields(sc.Text())
		separator := slices.Index(flds, "-")
		if len(flds) >= 5 && separator > 0 && separator+3 <= len(flds) && flds[2] == deviceID {
			return unescapeLinuxMountPath(flds[4]), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("unable to scan mountinfo: %w", err)
	}
	return "", fmt.Errorf("unable to find mount path for device '%s'", deviceID)
}

func findVolumePathInMounts(mounts io.Reader, volume string) (string, error) {
	sc := bufio.NewScanner(mounts)
	for sc.Scan() {
		flds := strings.Fields(sc.Text())
		if len(flds) >= 2 && flds[0] == volume {
			return unescapeLinuxMountPath(flds[1]), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("unable to scan mounts: %w", err)
	}
	return "", fmt.Errorf("unable to find mount path for volume '%s'", volume)
}

func unescapeLinuxMountPath(path string) string {
	var result strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] != '\\' || i+3 >= len(path) {
			result.WriteByte(path[i])
			continue
		}
		value, err := strconv.ParseUint(path[i+1:i+4], 8, 8)
		if err != nil {
			result.WriteByte(path[i])
			continue
		}
		result.WriteByte(byte(value))
		i += 3
	}
	return result.String()
}
