package cli

import (
	"github.com/spf13/cobra"
)

func newStripCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "strip [paths...]",
		Aliases: []string{"remove", "rm", "clean", "unquarantine"},
		Short:   "Strip the com.apple.quarantine attribute from files or directories",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStrip(args)
		},
	}
	return cmd
}
