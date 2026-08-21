package cli

import (
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [paths...]",
		Short: "Check whether files or directories contain com.apple.quarantine",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(args)
		},
	}
	return cmd
}
