package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"antiQuarantine/internal/history"
	"antiQuarantine/internal/quarantine"
	"antiQuarantine/internal/walker"
	"github.com/spf13/cobra"
)

var (
	Version = "2.0.0"
	Commit  = "none"
	Date    = "unknown"

	flagRecursive   bool
	flagRemove      bool
	flagDryRun      bool
	flagJSON        bool
	flagQuiet       bool
	flagFolder      string
	flagCat2gether  bool
	flagNoCrossDev  bool
	flagFollowLinks bool
)

// Cat2gether Easter egg banner
func printCat2getherAd() {
	fmt.Println()
	fmt.Println("cat2gether — best dating app for geeks & nerds!")
	fmt.Println("Find your purr-fect match!")
	fmt.Println()
	fmt.Println("   |\\---/|")
	fmt.Println("   | ,_, |")
	fmt.Println("    \\_`_/-..----.")
	fmt.Println(" ___/ `   ' ,\"\"+ \\  ")
	fmt.Println("(__...'   __\\    |`.___.';")
	fmt.Println("  (_,...'(_,.`__)/'.....+")
	fmt.Println()
	fmt.Println("Find out more: https://cat2gether.com")
	fmt.Println()
}

// NewRootCmd initializes the primary command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "aq [flags] [paths...]",
		Short: "antiQuarantine (aq) — High-performance macOS Gatekeeper quarantine management",
		Long: `antiQuarantine (aq) is a high-performance macOS systems utility engineered to 
inspect, strip, restore, and monitor the 'com.apple.quarantine' extended attribute (xattr).

Direct Darwin syscalls bypass shell execution overhead for lightning-fast batch processing.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagCat2gether {
				printCat2getherAd()
				return nil
			}

			// Handle legacy -f / --folder flag
			var targets []string
			if flagFolder != "" {
				targets = append(targets, flagFolder)
				flagRecursive = true
			}
			targets = append(targets, args...)

			if len(targets) == 0 {
				return cmd.Help()
			}

			// Default behavior: if --remove or -r is set, strip quarantine; otherwise check/inspect
			if flagRemove {
				return runStrip(targets)
			}
			return runCheck(targets)
		},
	}

	rootCmd.SetVersionTemplate("aq {{.Version}}\n")

	rootCmd.Flags().BoolVarP(&flagRemove, "remove", "r", false, "Remove com.apple.quarantine extended attribute")
	rootCmd.Flags().StringVarP(&flagFolder, "folder", "f", "", "Legacy compatibility flag for target directory")
	rootCmd.Flags().BoolVarP(&flagCat2gether, "cat2gether", "c", false, "Display Cat2gether easter egg banner")

	rootCmd.PersistentFlags().BoolVarP(&flagRecursive, "recursive", "R", false, "Recursively process directory contents")
	rootCmd.PersistentFlags().BoolVarP(&flagDryRun, "dry-run", "n", false, "Simulate actions without modifying extended attributes")
	rootCmd.PersistentFlags().BoolVarP(&flagJSON, "json", "j", false, "Output results in structured JSON format")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&flagNoCrossDev, "one-file-system", true, "Do not cross filesystem mount boundaries")
	rootCmd.PersistentFlags().BoolVar(&flagFollowLinks, "follow-symlinks", false, "Follow symbolic links during recursive traversal")

	// Register subcommands
	rootCmd.AddCommand(newCheckCmd())
	rootCmd.AddCommand(newStripCmd())
	rootCmd.AddCommand(newInspectCmd())
	rootCmd.AddCommand(newFixAppCmd())
	rootCmd.AddCommand(newRestoreCmd())
	rootCmd.AddCommand(newWatchCmd())
	rootCmd.AddCommand(newTUICmd())
	rootCmd.AddCommand(newHistoryCmd())
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

// Execute runs the root command, intercepting combined legacy flags (-rf, -fr, -c2g, -v)
func Execute() {
	args := preprocessArgs(os.Args[1:])
	rootCmd := NewRootCmd()
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var knownSubcommands = map[string]bool{
	"check": true, "strip": true, "remove": true, "rm": true, "clean": true, "unquarantine": true,
	"inspect": true, "fix-app": true, "fix": true, "sanitize-app": true, "restore": true, "undo": true,
	"watch": true, "monitor": true, "daemon": true, "tui": true, "ui": true, "browse": true,
	"history": true, "log": true, "vault": true, "version": true, "completion": true, "help": true,
}

// preprocessArgs preserves full backwards compatibility for legacy flags and routes to subcommands
func preprocessArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	for _, a := range args {
		if a == "-c2g" || a == "--cat2gether" {
			return []string{"--cat2gether"}
		}
	}

	// Check if already begins with a known subcommand
	firstArg := args[0]
	if knownSubcommands[firstArg] {
		return normalizeFlags(args)
	}

	// If help or version flag passed alone
	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		return []string{"--version"}
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		return []string{"--help"}
	}

	// Check if any non-flag token matches a known subcommand
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && knownSubcommands[a] {
			return normalizeFlags(args)
		}
	}

	// Legacy syntax detected: determine whether action is strip or check
	isRemove := false
	isRecursive := false
	var remaining []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-r" || a == "--remove" {
			isRemove = true
			continue
		}
		if a == "-rf" || a == "-fr" {
			isRemove = true
			isRecursive = true
			continue
		}
		if a == "-f" || a == "--folder" {
			isRecursive = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				remaining = append(remaining, args[i+1])
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "--folder=") {
			isRecursive = true
			remaining = append(remaining, strings.TrimPrefix(a, "--folder="))
			continue
		}
		if strings.HasPrefix(a, "-f=") {
			isRecursive = true
			remaining = append(remaining, strings.TrimPrefix(a, "-f="))
			continue
		}
		if a == "-R" || a == "--recursive" {
			isRecursive = true
			continue
		}
		remaining = append(remaining, a)
	}

	var result []string
	if isRemove {
		result = append(result, "strip")
	} else {
		result = append(result, "check")
	}

	if isRecursive {
		result = append(result, "-R")
	}

	result = append(result, remaining...)
	return normalizeFlags(result)
}

func normalizeFlags(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-v" {
			out = append(out, "--version")
			continue
		}
		if a == "-rf" || a == "-fr" {
			out = append(out, "-R")
			continue
		}
		out = append(out, a)
	}
	return out
}

func runCheck(targets []string) error {
	// If checking in non-recursive mode
	if !flagRecursive {
		var results []*quarantine.FileInfo
		for _, target := range targets {
			info, err := quarantine.InspectFile(target)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "File not found: %s\n", target)
					continue
				}
				return err
			}
			results = append(results, info)

			if !flagJSON && !flagQuiet {
				if info.HasQuarantine {
					fmt.Println(target, "|| HAS com.apple.quarantine")
				} else {
					fmt.Println(target, "|| does NOT have com.apple.quarantine")
				}
			}
		}

		if flagJSON {
			bytes, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(bytes))
		}
		return nil
	}

	// Recursive check
	res, err := walker.Walk(targets, walker.Options{
		Recursive:      flagRecursive,
		FollowSymlinks: flagFollowLinks,
		CrossDevice:    !flagNoCrossDev,
		DryRun:         false,
		Strip:          false,
		OnQuarantined: func(path string, info *quarantine.FileInfo) {
			if !flagJSON && !flagQuiet {
				fmt.Println(path)
			}
		},
	})
	if err != nil {
		return err
	}

	if flagJSON {
		bytes, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(bytes))
	} else if !flagQuiet && flagRecursive {
		fmt.Fprintf(os.Stderr, "\nFound %d quarantined file(s) across %d scanned.\n", res.TotalQuarantined, res.TotalScanned)
	}

	return nil
}

func runStrip(targets []string) error {
	// If stripping in non-recursive mode
	if !flagRecursive {
		for _, target := range targets {
			raw, _ := quarantine.GetRawQuarantine(target)

			if flagDryRun {
				if !flagQuiet {
					fmt.Printf("[Dry-Run] Would remove quarantine from: %s\n", target)
				}
				continue
			}

			// Save snapshot for restore
			if raw != "" {
				_ = history.RecordStrip(target, raw)
			}

			err := quarantine.RemoveQuarantine(target)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "File not found: %s\n", target)
					continue
				}
				return err
			}

			if !flagQuiet {
				fmt.Println("Removed quarantine from:", target)
			}
		}
		return nil
	}

	// Recursive strip
	res, err := walker.Walk(targets, walker.Options{
		Recursive:      flagRecursive,
		FollowSymlinks: flagFollowLinks,
		CrossDevice:    !flagNoCrossDev,
		DryRun:         flagDryRun,
		Strip:          true,
		OnProcessed: func(path string, action string) {
			if !flagQuiet && !flagJSON {
				if flagDryRun {
					fmt.Printf("[Dry-Run] Would remove quarantine from: %s\n", path)
				} else {
					fmt.Println("Removed quarantine from:", path)
				}
			}
		},
		OnQuarantined: func(path string, info *quarantine.FileInfo) {
			if !flagDryRun && info != nil && info.Metadata != nil {
				_ = history.RecordStrip(path, info.Metadata.Raw)
			}
		},
	})
	if err != nil {
		return err
	}

	if flagJSON {
		bytes, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(bytes))
	} else if !flagQuiet && flagRecursive {
		fmt.Fprintf(os.Stderr, "\nRemoved quarantine from %d file(s) (Scanned: %d).\n", res.TotalProcessed, res.TotalScanned)
	}

	return nil
}
