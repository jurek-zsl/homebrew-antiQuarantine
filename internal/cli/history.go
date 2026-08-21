package cli

import (
	"encoding/json"
	"fmt"

	"antiQuarantine/internal/history"
	"github.com/spf13/cobra"
)

var (
	flagHistoryLimit int
	flagHistoryClear bool
)

func newHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "history",
		Aliases: []string{"log", "vault"},
		Short:   "View or manage stripped quarantine history in the local undo vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagHistoryClear {
				if err := history.ClearHistory(); err != nil {
					return err
				}
				fmt.Println("Cleared all quarantine undo history.")
				return nil
			}

			records, err := history.ListRecent(flagHistoryLimit)
			if err != nil {
				return err
			}

			if len(records) == 0 {
				fmt.Println("History vault is empty.")
				return nil
			}

			if flagJSON {
				bytes, _ := json.MarshalIndent(records, "", "  ")
				fmt.Println(string(bytes))
				return nil
			}

			fmt.Println("📜 Quarantine History Vault (Recent entries):")
			for _, r := range records {
				status := "Stripped"
				if r.RestoredAt != nil {
					status = fmt.Sprintf("Restored at %s", r.RestoredAt.Format("2006-01-02 15:04:05"))
				}
				fmt.Printf(" [#%d] %s (%s)\n", r.ID, r.FilePath, status)
				fmt.Printf("       • Date:  %s\n", r.StrippedAt.Format("2006-01-02 15:04:05 UTC"))
				fmt.Printf("       • Value: %s\n", r.RawQuarantine)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&flagHistoryLimit, "limit", "l", 20, "Number of recent history records to display")
	cmd.Flags().BoolVar(&flagHistoryClear, "clear", false, "Clear all history records from the vault")

	return cmd
}
