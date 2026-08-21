package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"antiQuarantine/internal/watcher"
	"github.com/spf13/cobra"
)

var (
	flagWatchAutoStrip bool
	flagWatchNotify    bool
	flagWatchExts      []string
)

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "watch [directories...]",
		Aliases: []string{"monitor", "daemon"},
		Short:   "Monitor directories (e.g. ~/Downloads) and detect/strip quarantine attributes on new files",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs := args
			if len(dirs) == 0 {
				home, err := os.UserHomeDir()
				if err == nil {
					downloads := filepath.Join(home, "Downloads")
					if _, err := os.Stat(downloads); err == nil {
						dirs = []string{downloads}
					}
				}
			}

			if len(dirs) == 0 {
				return fmt.Errorf("no directory specified to watch")
			}

			w, err := watcher.New(watcher.Config{
				Directories: dirs,
				AutoStrip:   flagWatchAutoStrip,
				Notify:      flagWatchNotify,
				Extensions:  flagWatchExts,
				Quiet:       flagQuiet,
			})
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\nStopping watcher...")
				cancel()
			}()

			return w.Start(ctx)
		},
	}

	cmd.Flags().BoolVarP(&flagWatchAutoStrip, "auto-strip", "a", false, "Automatically strip quarantine attributes from incoming files")
	cmd.Flags().BoolVarP(&flagWatchNotify, "notify", "N", true, "Send macOS desktop notification when quarantine is detected or stripped")
	cmd.Flags().StringSliceVarP(&flagWatchExts, "ext", "e", nil, "Filter extensions (e.g. dmg,app,zip,pkg)")

	return cmd
}
