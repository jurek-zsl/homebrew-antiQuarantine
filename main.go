package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const version = "1.2.0"
const quarantineAttribute = "com.apple.quarantine"

func main() {
	// Manual arg parse to support long flags and combined -rf
	// Add hidden cat2gether flag (not shown in help) — parse it here
	removeFlag, folderArg, versionFlag, helpFlag, cat2getherFlag, positional, parseErr := parseArgs(os.Args[1:])
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "Error parsing args:", parseErr)
		printUsage()
		os.Exit(1)
	}

	if cat2getherFlag {
		printCat2getherAd()
		os.Exit(0)
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
		absPath, err := filepath.Abs(folderArg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error getting absolute path:", err)
			os.Exit(1)
		}
		if err := ensureExists(absPath); err != nil {
			fmt.Fprintln(os.Stderr, "Directory not found:", absPath)
			os.Exit(2)
		}
		if removeFlag {
			if err := processPathParallel(absPath, true); err != nil {
				fmt.Fprintln(os.Stderr, "Errors occurred during removal:", err)
				os.Exit(1)
			}
		} else {
			if err := processPathParallel(absPath, false); err != nil {
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
	// No need for ensureExists, the xattr calls will fail if the file doesn't exist.

	if removeFlag {
		if err := removeQuarantine(target, false); err != nil {
			// Check if the error is because the file doesn't exist
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "File not found:", target)
				os.Exit(2)
			}
			fmt.Fprintln(os.Stderr, "Failed to remove quarantine:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	has, err := hasQuarantine(target)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "File not found:", target)
			os.Exit(2)
		}
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
	fmt.Fprintln(os.Stderr, "  This tool uses native macOS APIs to detect and remove the")
	fmt.Fprintln(os.Stderr, "  com.apple.quarantine extended attribute for maximum performance.")
}

// Hidden promotional ad for Cat2gether — triggered by -c2g / --cat2gether (not shown in help)
func printCat2getherAd() {
	// Two header lines
	fmt.Println()
	fmt.Println("cat2gether — best dating app for geeks & nerds!")
	fmt.Println("Find your purr-fect match!")
	fmt.Println()
	// ASCII cat block (user provided)
	fmt.Println("   |\\---/|")
	fmt.Println("   | ,_, |")
	fmt.Println("    \\_`_/-..----.")
	fmt.Println(" ___/ `   ' ,\"\"+ \\  ")
	fmt.Println("(__...'   __\\    |`.___.';")
	fmt.Println("  (_,...'(_,.`__)/'.....+")
	fmt.Println()
	// Find out more line
	fmt.Println("Find out more: https://cat2gether.com")
	fmt.Println()
}

// parseArgs handles short and long flags and supports combined -rf
func parseArgs(in []string) (remove bool, folder string, ver bool, help bool, cat2gether bool, positional []string, err error) {
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
				return false, "", false, false, false, nil, errors.New("--folder requires an argument")
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
		if a == "--cat2gether" {
			cat2gether = true
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
			// handle -rf, -fr, -r, -f <arg>, -v, -h, -c2g
			// If exactly -rf or -fr, next token is folder
			if a == "-rf" || a == "-fr" {
				remove = true
				if i+1 >= len(in) {
					return false, "", false, false, false, nil, errors.New("-f requires a directory argument")
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
			if a == "-c2g" {
				cat2gether = true
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
					return false, "", false, false, false, nil, errors.New("-f requires a directory argument")
				}
				folder = in[i+1]
				i += 2
				continue
			}
			// unknown short flag
			return false, "", false, false, false, nil, fmt.Errorf("unknown flag: %s", a)
		}

		// positional
		positional = append(positional, a)
		i++
	}
	return remove, folder, ver, help, cat2gether, positional, nil
}

func ensureExists(path string) error {
	_, err := os.Stat(path)
	return err
}

func hasQuarantine(path string) (bool, error) {
	// Pass a nil buffer to just check for existence and get the size.
	_, err := unix.Getxattr(path, quarantineAttribute, nil)
	if err == nil {
		return true, nil // Attribute exists
	}
	if err == unix.ENOATTR || err == unix.ENODATA {
		return false, nil // Attribute does not exist, not an error
	}
	return false, err // Another error occurred
}

func removeQuarantine(path string, quiet bool) error {
	err := unix.Removexattr(path, quarantineAttribute)
	if err == nil {
		if !quiet {
			fmt.Println("Removed quarantine from:", path)
		}
		return nil
	}
	if err == unix.ENOATTR || err == unix.ENODATA {
		return nil // Attribute was not there, which is fine
	}
	return fmt.Errorf("xattr remove failed on %s: %w", path, err)
}

func getDeviceID(path string) (uint64, error) {
	var stat unix.Stat_t
	err := unix.Stat(path, &stat)
	if err != nil {
		return 0, err
	}
	return uint64(stat.Dev), nil
}

func processPathParallel(root string, remove bool) error {
	rootDeviceID, err := getDeviceID(root)
	if err != nil {
		return fmt.Errorf("could not get device ID for root: %w", err)
	}

	paths := make(chan string, 100)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				has, err := hasQuarantine(path)
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Sprintf("error checking xattr for %s: %v", path, err))
					mu.Unlock()
					continue
				}
				if has {
					if remove {
						if err := removeQuarantine(path, true); err != nil {
							mu.Lock()
							errors = append(errors, fmt.Sprintf("failed to remove quarantine from %s: %v", path, err))
							mu.Unlock()
						} else {
							fmt.Println("Removed quarantine from:", path)
						}
					} else {
						fmt.Println(path)
					}
				}
			}
		}()
	}

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, "skipping (access error):", path, "->", err)
			return nil // Continue walking
		}

		// Check for filesystem boundary
		if d.IsDir() {
			deviceID, err := getDeviceID(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, "skipping (stat error):", path, "->", err)
				return filepath.SkipDir
			}
			if deviceID != rootDeviceID {
				fmt.Fprintln(os.Stderr, "skipping (filesystem boundary):", path)
				return filepath.SkipDir
			}
		}

		paths <- path
		return nil
	})

	close(paths)
	wg.Wait()

	if walkErr != nil {
		return walkErr
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors:\n%s", len(errors), strings.Join(errors, "\n"))
	}

	return nil
}
