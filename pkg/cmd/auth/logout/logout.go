package logout

import (
	"context"

	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/conf"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func New(c *cli.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of Airplane",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cmd, c)
		},
	}
	return cmd
}

func run(ctx context.Context, cmd *cobra.Command, c *cli.Config) error {
	cfg, err := conf.ReadDefault()
	if !errors.Is(err, conf.ErrMissing) {
		cfg.Token = ""

		if err := conf.WriteDefault(cfg); err != nil {
			return err
		}
	}

	cmd.Printf("Logged out.\n")

	return nil
}