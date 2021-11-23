package info

import (
	"context"

	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/print"
	"github.com/airplanedev/lib/pkg/build/logger"
	"github.com/spf13/cobra"
)

func New(c *cli.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get information about the currently logged in user / team",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), c)
		},
	}
	return cmd
}

func run(ctx context.Context, c *cli.Config) error {
	var client = c.Client

	res, err := client.AuthInfo(ctx)
	if err != nil {
		return err
	}

	print.Print(res, func() {
		var userStr string
		if res.User == nil {
			userStr = logger.Gray("<no user>")
		} else {
			userStr = res.User.Email
		}
		logger.Log("  Signed in as %s", logger.Blue(userStr))
		logger.Log("  Using team %s (ID: %s)", logger.Blue(res.Team.Name), res.Team.ID)
	})

	return nil
}
