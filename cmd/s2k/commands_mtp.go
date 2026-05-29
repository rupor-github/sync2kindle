//go:build mtp

package main

import (
	"fmt"

	cli "github.com/urfave/cli/v3"

	"s2k/sync"
)

func appendMTPCommand(commands []*cli.Command) []*cli.Command {
	return append(commands, &cli.Command{
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
	})
}
