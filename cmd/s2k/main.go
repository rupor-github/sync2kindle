package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	cli "github.com/urfave/cli/v3"
	"go.uber.org/zap"

	"s2k/config"
	"s2k/history"
	"s2k/misc"
	"s2k/state"
	"s2k/sync"
)

func beforeAppRun(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	var err error

	if cmd.NArg() == 0 {
		return ctx, nil
	}
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)

	configFile := cmd.String("config")
	if env.Cfg, err = config.LoadConfiguration(configFile); err != nil {
		return ctx, fmt.Errorf("unable to prepare configuration: %w", err)
	}
	if cmd.Bool("debug") {
		if env.Rpt, err = env.Cfg.Reporting.Prepare(); err != nil {
			return ctx, fmt.Errorf("unable to prepare debug reporter: %w", err)
		}
		// save complete processed configuration if external configuration was provided
		if len(configFile) > 0 {
			// we do not want any of your secrets!
			if data, err := config.Dump(env.Cfg); err == nil {
				env.Rpt.StoreData(fmt.Sprintf("config/%s", filepath.Base(configFile)), data)
			}
		}
	}
	if env.Log, err = env.Cfg.Logging.Prepare(env.Rpt); err != nil {
		return ctx, fmt.Errorf("unable to prepare logs: %w", err)
	}
	env.RestoreStdLog = zap.RedirectStdLog(env.Log)

	env.Log.Debug("Program started", zap.Strings("args", os.Args), zap.String("ver", misc.GetVersion()+" ("+runtime.Version()+") : "+misc.GetGitHash()))
	if env.Rpt != nil {
		env.Log.Info("Creating debug report", zap.String("location", env.Rpt.Name()))
	}
	return ctx, nil
}

func afterAppRun(ctx context.Context, cmd *cli.Command) error {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	if env.Log != nil {
		env.Log.Debug("Program ended", zap.Duration("elapsed", time.Since(env.Start)), zap.Strings("parsed args", cmd.Args().Slice()))
	}
	return nil
}

func beforeCmdRun(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	env := ctx.Value(state.EnvValue).(*state.LocalEnv)

	configFile := cmd.String("config")
	if len(configFile) == 0 && env.Log != nil {
		env.Log.Info("Using defaults (no configuration file)")
	}
	return ctx, nil
}

func main() {

	env := state.NewLocalEnv()
	ctx := context.WithValue(context.Background(), state.EnvValue, env)

	app := &cli.Command{
		Name:            "s2k",
		Usage:           "synchronizing local books with supported kindle device over MTP protocol, USBMS mount or using e-mail",
		Version:         misc.GetVersion() + " (" + runtime.Version() + ") : " + misc.GetGitHash(),
		HideHelpCommand: true,
		Before:          beforeAppRun,
		After:           afterAppRun,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, DefaultText: "", Usage: "load configuration from `FILE` (YAML)"},
			&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Usage: "changes program behavior to help troubleshooting"},
		},
		Commands: []*cli.Command{
			{
				Name:   "mtp",
				Usage:  "Synchronizes books between local source and target device over MTP protocol",
				Before: beforeCmdRun,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "ignore-device-removals", Aliases: []string{"i"}, Usage: "do not respect books removals on the device"},
					&cli.BoolFlag{Name: "dry-run", Usage: "do not perform any actual changes"},
				},
				Action: sync.RunMTP,
				CustomHelpTemplate: fmt.Sprintf(`%s
Using MTP protocol syncronizes books between 'source' local directory and 'target' path on the device.
Both could be specified in configuration file, otherwise 'source' is current working directory and 'target' is "documents/mybooks".
Kindle device is expected to be connected at the time of operation.

When 'ignore-device-removals' flag is set, books removed from the device are not removed from the local source.
`, cli.CommandHelpTemplate),
			},
			{
				Name:   "usb",
				Usage:  "Synchronizes books between local source and target device using USBMS mount",
				Before: beforeCmdRun,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "ignore-device-removals", Aliases: []string{"i"}, Usage: "do not respect books removals on the device"},
					&cli.BoolFlag{Name: "dry-run", Usage: "do not perform any actual changes"},
					&cli.BoolFlag{Name: "unmount", Aliases: []string{"u"}, Usage: "Attempts to prepare device for safe disconnect"},
				},
				Action: sync.RunUSB,
				CustomHelpTemplate: fmt.Sprintf(`%s
Using device storage mounted over USB syncronizes books between 'source' local directory and 'target' path on the device.
Both could be specified in configuration file, otherwise 'source' is current working directory and 'target' is "documents/mybooks".
Kindle device is expected to be mounted at the time of operation.

When 'ignore-device-removals' flag is set, books removed from the device are not removed from the local source.

With 'unmount' flag set, attempt is made to safely unmount storage after sync operation. Has no effect with 'dry-run'.
Results of this flag are very OS dependent, for example on Windows it may fail if not all buffers have been yet written
to storage and will fail if something still have device opened, on Linux it requires admin priviliges and will only
unmount filesystem after mount seases to be busy, etc. Since this is command line tool this flag mostly makes sense
on Windows, where standard way of unmounting USB media from the command line has been missing for years. On Linux
you could simply use 'eject' or 'udisksctl' commands.
`, cli.CommandHelpTemplate),
			},
			{
				Name:   "mail",
				Usage:  "Synchronizes books between local source and target device using kindle e-mail",
				Before: beforeCmdRun,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dry-run", Usage: "do not perform any actual changes"},
				},
				Action: sync.RunMail,
				CustomHelpTemplate: fmt.Sprintf(`%s
Using Amazon e-mail delivery syncronizes books between 'source' local directory and 'target' device.
Both could be specified in configuration file, otherwise 'source' is current working directory and 'target' has no default.
In this case have no way of accessing device content, so all decisions are based on local files and history.

Proper configuration is expected for succesful operation, including working smtp server auth and authorized e-mail address
(amazon account settings).
`, cli.CommandHelpTemplate),
			},
			{
				Name:   "history",
				Usage:  "Lists details for local history files",
				Before: beforeCmdRun,
				Action: history.RunList,
				CustomHelpTemplate: fmt.Sprintf(`%s
Lists local history databases specifying details for each of them.
`, cli.CommandHelpTemplate),
			},
			{
				Name:   "dumpconfig",
				Usage:  "Dumps either default or active configuration (YAML)",
				Before: beforeCmdRun,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dry-run", Usage: "output active configuration to be used in actual operations, including values from --config file"},
				},
				Action:    outputConfiguration,
				ArgsUsage: "DESTINATION",
				CustomHelpTemplate: fmt.Sprintf(`%s
DESTINATION:
    file name to write configuration to, if absent - STDOUT

Produces file with default configuration values.
To see actual "active" configuration use dry-run mode.
`, cli.CommandHelpTemplate),
			},
		},
	}

	err := app.Run(ctx, os.Args)
	if err != nil {
		if env.Log != nil {
			env.Log.Error("Command ended with error", zap.Error(err))
		} else {
			// if we do not have logger yet, we can only print to stderr
			fmt.Fprintf(os.Stderr, "Command ended with error: %v\n", err)
		}
	}
	if env.Log != nil {
		_ = env.Log.Sync()
		env.RestoreStdLog()
		env.Log = nil
	}
	if env.Rpt != nil {
		if err := env.Rpt.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating debug report: %v\n", err)
		}
	}
	// cleanup temporary directory with thumbnails if any
	if env.Cfg != nil && len(env.Cfg.Thumbnails.Dir) > 0 {
		os.RemoveAll(env.Cfg.Thumbnails.Dir)
	}
	// cleanup temporary directory with mails if any
	if env.Cfg != nil && len(env.Cfg.Smtp.Dir) > 0 {
		os.RemoveAll(env.Cfg.Smtp.Dir)
	}
	// remove empty panic log if any
	if env.Cfg != nil && len(env.Cfg.Logging.FileLogger.Destination) > 0 {
		debug.SetCrashOutput(nil, debug.CrashOptions{})
		fname := filepath.Join(filepath.Dir(env.Cfg.Logging.FileLogger.Destination), "sync2kindle-panic.log")
		if fi, err := os.Stat(fname); err == nil && fi.Size() == 0 {
			os.Remove(fname)
		}
	}
	if err != nil {
		os.Exit(1)
	}
}

func outputConfiguration(ctx context.Context, cmd *cli.Command) error {

	env := ctx.Value(state.EnvValue).(*state.LocalEnv)
	if cmd.Args().Len() > 1 {
		env.Log.Warn("Malformed command line, too many destinations", zap.Strings("ignoring", cmd.Args().Slice()[1:]))
	}

	fname := cmd.Args().Get(0)

	var (
		err   error
		data  []byte
		state string
	)

	out := os.Stdout
	if len(fname) > 0 {
		out, err = os.Create(fname)
		if err != nil {
			return fmt.Errorf("unable to create destination file '%s': %w", fname, err)
		}
		defer out.Close()

	}

	if cmd.Bool("dry-run") {
		state = "active"
		data, err = config.Dump(env.Cfg)
	} else {
		state = "default"
		data, err = config.Prepare()
	}
	if err != nil {
		return fmt.Errorf("unable to get configuration: %w", err)
	}

	env.Log.Info("Outputing configuration", zap.String("state", state), zap.String("file", fname))

	_, err = out.Write(data)
	if err != nil {
		return fmt.Errorf("unable to write configuration: %w", err)
	}
	return nil
}
