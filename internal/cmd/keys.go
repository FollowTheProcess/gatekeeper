package cmd

import (
	"context"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/gatekeeper/internal/gatekeeper"
)

func buildKeysCommand() (*cli.Command, error) {
	var options gatekeeper.KeysOptions

	return cli.New(
		"keys",
		cli.Short("Generate a private/public ed255519 key pair"),
		cli.Flag(&options.Debug, "debug", 'd', "Enable debug logging"),
		cli.Flag(&options.Name, "name", 'n', "Name for the key pair, defaults to a random name"),
		cli.Arg(&options.Path, "path", "Path in which to save the keys", cli.ArgDefault(".")),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			gk := gatekeeper.New(options.Debug, cmd.Stdin(), cmd.Stdout(), cmd.Stderr())

			return gk.Keys(ctx, options)
		}),
	)
}
