package cli

import (
	"os"
	"path/filepath"

	"antiQuarantine/internal/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tui [directory]",
		Aliases: []string{"ui", "browse"},
		Short:   "Launch the interactive Terminal UI inspector to browse and strip quarantine flags",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			} else {
				// Default to ~/Downloads if it exists and current dir is empty or root
				home, err := os.UserHomeDir()
				if err == nil {
					downloads := filepath.Join(home, "Downloads")
					if _, err := os.Stat(downloads); err == nil && dir == "." {
						// use current working dir by default
						dir = "."
					}
				}
			}
			return tui.Run(dir)
		},
	}
	return cmd
}
