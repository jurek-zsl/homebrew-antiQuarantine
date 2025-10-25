package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const version = "1.0.0"

func main() {
	// Manual arg parse to support long flags and combined -rf
	removeFlag, folderArg, versionFlag, helpFlag, positional, parseErr := parseArgs(os.Args[1:])
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "Error parsing args:", parseErr)
		printUsage()
		os.Exit(1)
	}

	if versionFlag {
		fmt.Println("aq", version)
		os.Exit(0)
	}

	if helpFlag {
		printUsage()
		os.Exit(0)
	}

	if folderArg != "" {
		// Folder mode
		if err := ensureExists(folderArg); err != nil {
			fmt.Fprintln(os.Stderr, "Directory not found:", folderArg)
			os.Exit(2)
		}
		if removeFlag {
			if err := removeQuarantineRecursive(folderArg); err != nil {
				fmt.Fprintln(os.Stderr, "Errors occurred:", err)
				os.Exit(1)
			}
		} else {
			if err := listQuarantinedInFolder(folderArg); err != nil {
				fmt.Fprintln(os.Stderr, "Error listing folder:", err)
				os.Exit(1)
			}
		}
		os.Exit(0)
	}

	// File mode
	if len(positional) == 0 {
		printUsage()
		os.Exit(1)
	}

	target := positional[0]
	if err := ensureExists(target); err != nil {
		fmt.Fprintln(os.Stderr, "File not found:", target)
		os.Exit(2)
	}

	if removeFlag {
		if err := removeQuarantine(target); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to remove quarantine:", err)
			os.Exit(1)
		}
		fmt.Println("Removed quarantine from:", target)
		os.Exit(0)
	}

	has, err := hasQuarantine(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error checking xattr:", err)
		os.Exit(1)
	}
	if has {
		fmt.Println(target, "|| HAS com.apple.quarantine")
		os.Exit(0)
	}
	fmt.Println(target, "|| does NOT have com.apple.quarantine")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "aq - antiQuarantine %s\n\n", version)
	fmt.Fprintf(os.Stderr, "We recommend putting filenames, and directories in quotes.\n\n")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  aq <file>                # print whether file has com.apple.quarantine")
	fmt.Fprintln(os.Stderr, "  aq -r <file>             # remove com.apple.quarantine from file")
	fmt.Fprintln(os.Stderr, "  aq -f <directory>        # list files in directory with com.apple.quarantine")
	fmt.Fprintln(os.Stderr, "  aq -rf <directory>       # remove com.apple.quarantine from all files in directory (recursive)")
	fmt.Fprintln(os.Stderr, "  aq -v, --version         # print version")
	fmt.Fprintln(os.Stderr, "  aq -h, --help            # show help")
	fmt.Fprintln(os.Stderr, "\nNotes:")
	fmt.Fprintln(os.Stderr, "  This tool calls the system \"xattr\" command to detect and remove the")
	fmt.Fprintln(os.Stderr, "  com.apple.quarantine extended attribute. It must be run on macOS.")
}

// parseArgs handles short and long flags and supports combined -rf
func parseArgs(in []string) (remove bool, folder string, ver bool, help bool, positional []string, err error) {
	i := 0
	for i < len(in) {
		a := in[i]
		if a == "--remove" {
			remove = true
			i++
			continue
		}
		if a == "--folder" {
			if i+1 >= len(in) {
				return false, "", false, false, nil, errors.New("--folder requires an argument")
			}
			folder = in[i+1]
			i += 2
			continue
		}
		if a == "--version" {
			ver = true
			i++
			continue
		}
		if a == "--help" {
			help = true
			i++
			continue
		}

		if strings.HasPrefix(a, "--folder=") {
			folder = strings.TrimPrefix(a, "--folder=")
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			// short or combined
			// handle -rf, -fr, -r, -f <arg>, -v, -h
			// If exactly -rf or -fr, next token is folder
			if a == "-rf" || a == "-fr" {
				remove = true
				if i+1 >= len(in) {
					return false, "", false, false, nil, errors.New("-f requires a directory argument")
				}
				folder = in[i+1]
				i += 2
				continue
			}
			// handle -r, -v, -h
			if a == "-r" {
				remove = true
				i++
				continue
			}
			if a == "-v" {
				ver = true
				i++
				continue
			}
			if a == "-h" {
				help = true
				i++
				continue
			}
			if strings.HasPrefix(a, "-f=") {
				folder = strings.TrimPrefix(a, "-f=")
				i++
				continue
			}
			if a == "-f" {
				if i+1 >= len(in) {
					return false, "", false, false, nil, errors.New("-f requires a directory argument")
				}
				folder = in[i+1]
				i += 2
				continue
			}
			// unknown short flag
			return false, "", false, false, nil, fmt.Errorf("unknown flag: %s", a)
		}

		// positional
		positional = append(positional, a)
		i++
	}
	return remove, folder, ver, help, positional, nil
}

func ensureExists(path string) error {
	_, err := os.Stat(path)
	return err
}

func hasQuarantine(path string) (bool, error) {
	// Use `xattr <path>` and check output for com.apple.quarantine
	out, err := exec.Command("xattr", path).Output()
	if err != nil {
		// xattr may exit with non-zero on error; if it's exit status 1 and output empty,
		// treat as no attributes. But we'll return the error only if it's unexpected.
		// To be conservative: if output contains the attr, return true; else return false without error when exit code indicates no attrs.
		s := strings.TrimSpace(string(out))
		if s == "" {
			// No attributes
			return false, nil
		}
		// otherwise return error
		return strings.Contains(s, "com.apple.quarantine"), nil
	}
	return strings.Contains(string(out), "com.apple.quarantine"), nil
}

func removeQuarantine(path string) error {
	fmt.Println("Removing quarantine from:", path)
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xattr failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func listQuarantinedInFolder(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// If we can't access an entry, report and continue
			fmt.Fprintln(os.Stderr, "skipping (access error):", path, "->", err)
			return nil
		}
		// check every file/dir
		has, err := hasQuarantine(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error checking xattr for:", path, "->", err)
			return nil
		}
		if has {
			fmt.Println(path)
		}
		return nil
	})
}

func removeQuarantineRecursive(root string) error {
	var hadErr bool
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, "skipping (access error):", path, "->", err)
			return nil
		}
		has, err := hasQuarantine(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error checking xattr for:", path, "->", err)
			hadErr = true
			return nil
		}
		if has {
			if err := removeQuarantine(path); err != nil {
				fmt.Fprintln(os.Stderr, "failed to remove quarantine from:", path, "->", err)
				hadErr = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if hadErr {
		return errors.New("some files failed to update; see stderr for details")
	}
	return nil
}
