package cli

import (
	"encoding/json"
	"fmt"

	"antiQuarantine/internal/bundle"
	"github.com/spf13/cobra"
)

var (
	flagAppCodesign bool
	flagAppSpctl    bool
)

func newFixAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fix-app <path/to/Application.app>",
		Aliases: []string{"fix", "sanitize-app"},
		Short:   "Deep sanitize a macOS .app bundle (recursively strip quarantine & ad-hoc codesign)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appPath := args[0]
			rep, err := bundle.FixBundle(appPath, bundle.Options{
				AdHocCodesign: flagAppCodesign,
				CheckSpctl:    flagAppSpctl,
				DryRun:        flagDryRun,
				Verbose:       !flagQuiet,
			})
			if err != nil {
				return err
			}

			if flagJSON {
				bytes, _ := json.MarshalIndent(rep, "", "  ")
				fmt.Println(string(bytes))
				return nil
			}

			fmt.Println("📦 Application Bundle Sanitizer Report")
			fmt.Printf("   • Target:       %s\n", rep.BundlePath)
			fmt.Printf("   • Stripped:     %d file(s)\n", rep.StrippedCount)
			if flagAppCodesign {
				fmt.Printf("   • Ad-Hoc Sign:  %v\n", rep.Codesigned)
				if rep.CodesignOutput != "" {
					fmt.Printf("     Output: %s\n", rep.CodesignOutput)
				}
			}
			if flagAppSpctl && rep.SpctlAssessment != "" {
				fmt.Printf("   • Gatekeeper:   %s\n", rep.SpctlAssessment)
			}
			if rep.Success {
				fmt.Println("✨ Bundle successfully sanitized and ready to launch!")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagAppCodesign, "codesign", "s", true, "Perform ad-hoc codesigning (codesign --force --deep --sign -)")
	cmd.Flags().BoolVar(&flagAppSpctl, "spctl", true, "Check Gatekeeper assessment status (spctl --assess)")

	return cmd
}
