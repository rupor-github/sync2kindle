//go:build mtp

package sync

import (
	"context"
	"path/filepath"
	"strings"

	cli "github.com/urfave/cli/v3"

	"s2k/common"
	"s2k/mtp"
	"s2k/state"
)

func RunMTP(ctx context.Context, cmd *cli.Command) error {
	return run(ctx, cmd, common.ProtocolMTP, connectMTP)
}

func connectMTP(cmd *cli.Command, env *state.LocalEnv) (driver, error) {
	return mtp.Connect(
		strings.Join([]string{env.Cfg.TargetPath, common.ThumbnailFolder}, string(filepath.ListSeparator)),
		env.Cfg.DeviceSerial, cmd.Bool("debug"), env.Log.Named("sync"))
}
