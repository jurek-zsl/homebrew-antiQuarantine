package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the aq version and build information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aq version %s (%s) built at %s on %s/%s\n", Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
		},
	}
	return cmd
}
