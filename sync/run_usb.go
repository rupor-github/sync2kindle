//go:build usb

package sync

import (
	"context"
	"path/filepath"
	"strings"

	cli "github.com/urfave/cli/v3"

	"s2k/common"
	"s2k/state"
	"s2k/usbms"
)

func RunUSB(ctx context.Context, cmd *cli.Command) error {
	return run(ctx, cmd, common.ProtocolUSB, connectUSB)
}

func connectUSB(cmd *cli.Command, env *state.LocalEnv) (driver, error) {
	return usbms.Connect(
		strings.Join([]string{env.Cfg.TargetPath, common.ThumbnailFolder}, string(filepath.ListSeparator)),
		env.Cfg.DeviceSerial, cmd.Bool("unmount") && !cmd.Bool("dry-run"), env.Log.Named("sync"))
}
