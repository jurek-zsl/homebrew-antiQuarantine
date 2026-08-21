package cli

import (
	"fmt"
	"os"

	"antiQuarantine/internal/history"
	"github.com/spf13/cobra"
)

var (
	flagRestoreLast bool
)

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restore [paths...]",
		Aliases: []string{"undo"},
		Short:   "Restore previously stripped com.apple.quarantine attributes from the undo vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagRestoreLast {
				rec, err := history.RestoreLast()
				if err != nil {
					return err
				}
				fmt.Printf("🔄 Restored quarantine attribute on: %s\n", rec.FilePath)
				fmt.Printf("   • Raw Value:  %s\n", rec.RawQuarantine)
				fmt.Printf("   • Stripped:   %s\n", rec.StrippedAt.Format("2006-01-02 15:04:05 UTC"))
				return nil
			}

			if len(args) == 0 {
				return cmd.Help()
			}

			for _, path := range args {
				rec, err := history.RestorePath(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error restoring %s: %v\n", path, err)
					continue
				}
				fmt.Printf("🔄 Restored quarantine attribute on: %s\n", rec.FilePath)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagRestoreLast, "last", "l", false, "Restore the most recently stripped file")

	return cmd
}
