package root

import "github.com/spf13/cobra"

// New returns a new root cobra command.
func New() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "airplane <command>",
		Short: "Airplane CLI",
	}

	return cmd
}
