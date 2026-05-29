//go:build usb

package main

import (
	"fmt"

	cli "github.com/urfave/cli/v3"

	"s2k/sync"
)

func appendUSBCommand(commands []*cli.Command) []*cli.Command {
	return append(commands, &cli.Command{
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
	})
}
