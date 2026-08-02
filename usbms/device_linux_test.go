//go:build linux

package usbms

import (
	"strings"
	"testing"
)

func TestFindVolumePathInMountInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mountInfo string
		deviceID  string
		want      string
	}{
		{
			name:      "matches device id",
			mountInfo: "36 25 8:33 / /run/media/user/Kindle rw,nosuid,nodev,relatime - vfat /dev/disk/by-label/Kindle rw\n",
			deviceID:  "8:33",
			want:      "/run/media/user/Kindle",
		},
		{
			name:      "unescapes mount path",
			mountInfo: "36 25 8:33 / /run/media/user/My\\040Kindle rw,nosuid,nodev,relatime - vfat /dev/sdc1 rw\n",
			deviceID:  "8:33",
			want:      "/run/media/user/My Kindle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := findVolumePathInMountInfo(strings.NewReader(tt.mountInfo), tt.deviceID)
			if err != nil {
				t.Fatalf("findVolumePathInMountInfo() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("findVolumePathInMountInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindVolumePathInMounts(t *testing.T) {
	t.Parallel()

	got, err := findVolumePathInMounts(strings.NewReader("/dev/sdc1 /run/media/user/My\\040Kindle vfat rw 0 0\n"), "/dev/sdc1")
	if err != nil {
		t.Fatalf("findVolumePathInMounts() error = %v", err)
	}
	if got != "/run/media/user/My Kindle" {
		t.Fatalf("findVolumePathInMounts() = %q, want %q", got, "/run/media/user/My Kindle")
	}
}
