// Package cmd implements the gatekeeper cli.
package cmd

import (
	"go.followtheprocess.codes/cli"
)

//nolint:gochecknoglobals // These have to be global for ldflags to set them.
var (
	version string
	commit  string
	date    string
)

// Build constructs and returns the entire gatekeeper cli.
func Build() (*cli.Command, error) {
	var debug bool

	return cli.New(
		"gatekeeper",
		cli.Short("A custom CI -> AWS auth mechanism powering my personal software catalog 🗄️"),
		cli.Version(version),
		cli.Commit(commit),
		cli.BuildDate(date),
		cli.Flag(&debug, "debug", 'd', "Enable debug logging"),
		cli.SubCommands(buildKeysCommand, buildAuthCommand),
	)
}
