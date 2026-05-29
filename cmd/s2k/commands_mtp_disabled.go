//go:build !mtp

package main

import cli "github.com/urfave/cli/v3"

func appendMTPCommand(commands []*cli.Command) []*cli.Command {
	return commands
}
