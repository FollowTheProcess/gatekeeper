package cmd

import (
	"context"
	"time"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/gatekeeper/internal/gatekeeper"
)

const defaultDuration = 5 * time.Minute

func buildAuthCommand() (*cli.Command, error) {
	var options gatekeeper.AuthOptions

	return cli.New(
		"auth",
		cli.Short("Authenticate a release session"),
		cli.Flag(&options.Debug, "debug", 'd', "Enable debug logging"),
		cli.Flag(
			&options.Duration,
			"duration",
			'D',
			"Requested duration of the session",
			cli.FlagDefault(defaultDuration),
		),
		cli.Flag(&options.URL, "url", 'u', "URL for the auth backend"),
		cli.Arg(&options.Project, "project", "Name of project to release"),
		cli.Arg(&options.Version, "version", "Version to release"),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			gk := gatekeeper.New(options.Debug, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())

			return gk.Auth(ctx, options)
		}),
	)
}
