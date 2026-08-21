package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"antiQuarantine/internal/quarantine"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [paths...]",
		Short: "Inspect detailed Gatekeeper quarantine metadata and provenance URLs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var results []*quarantine.FileInfo
			for _, target := range args {
				info, err := quarantine.InspectFile(target)
				if err != nil && os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "File not found: %s\n", target)
					continue
				}
				results = append(results, info)
			}

			if flagJSON {
				bytes, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(bytes))
				return nil
			}

			for _, info := range results {
				printInspectCard(info)
			}
			return nil
		},
	}
	return cmd
}

func printInspectCard(info *quarantine.FileInfo) {
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("📂 Target: %s\n", info.Path)
	if !info.HasQuarantine {
		fmt.Println("🛡️  Status: Clean (No com.apple.quarantine attribute)")
		return
	}

	fmt.Println("🚨 Status: Quarantined")
	if info.Metadata != nil {
		m := info.Metadata
		fmt.Printf("   • Raw Value:   %s\n", m.Raw)
		fmt.Printf("   • Agent:       %s\n", m.Agent)
		fmt.Printf("   • Timestamp:   %s (%s)\n", m.Timestamp.Format("2006-01-02 15:04:05 UTC"), m.Timestamp.Local().Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("   • Flags:       0x%s\n", m.FlagsHex)
		for _, lbl := range m.FlagLabels {
			fmt.Printf("     - %s\n", lbl)
		}
		if m.EventUUID != "" {
			fmt.Printf("   • Event UUID:  %s\n", m.EventUUID)
		}
	}

	if info.Provenance != nil {
		p := info.Provenance
		fmt.Println("🌐 Provenance (LaunchServices Quarantine Events DB):")
		if p.AgentBundleID != "" {
			fmt.Printf("   • Bundle ID:   %s\n", p.AgentBundleID)
		}
		if p.DataURL != "" {
			fmt.Printf("   • Origin URL:  %s\n", p.DataURL)
		}
		if p.OriginURL != "" {
			fmt.Printf("   • Referrer:    %s\n", p.OriginURL)
		}
		if p.SenderName != "" || p.SenderAddress != "" {
			fmt.Printf("   • Sender:      %s <%s>\n", p.SenderName, p.SenderAddress)
		}
	}
	fmt.Println("────────────────────────────────────────────────────────────")
}
