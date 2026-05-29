//go:build !usb

package main

import cli "github.com/urfave/cli/v3"

func appendUSBCommand(commands []*cli.Command) []*cli.Command {
	return commands
}
