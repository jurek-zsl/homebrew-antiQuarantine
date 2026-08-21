package walker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"antiQuarantine/internal/quarantine"
	"golang.org/x/sys/unix"
)

// Result contains metrics and outcomes of a walk/strip operation
type Result struct {
	TotalScanned      int64             `json:"total_scanned"`
	TotalQuarantined  int64             `json:"total_quarantined"`
	TotalProcessed    int64             `json:"total_processed"`
	QuarantinedPaths  []string          `json:"quarantined_paths"`
	ProcessedPaths    []string          `json:"processed_paths"`
	Errors            map[string]string `json:"errors,omitempty"`
}

// Options configures the directory walking engine
type Options struct {
	Recursive      bool
	FollowSymlinks bool
	CrossDevice    bool
	Workers        int
	DryRun         bool
	Strip          bool
	OnQuarantined  func(path string, info *quarantine.FileInfo)
	OnProcessed    func(path string, action string)
	OnError        func(path string, err error)
}

// Walk scans one or more target paths (files or directories)
func Walk(targets []string, opts Options) (*Result, error) {
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU() * 2
		if opts.Workers < 4 {
			opts.Workers = 4
		}
		if opts.Workers > 32 {
			opts.Workers = 32
		}
	}

	res := &Result{
		QuarantinedPaths: make([]string, 0),
		ProcessedPaths:   make([]string, 0),
		Errors:           make(map[string]string),
	}

	var (
		mu          sync.Mutex
		scanned     int64
		quarantined int64
		processed   int64
	)

	workChan := make(chan string, 1024)
	var wg sync.WaitGroup

	// Launch worker pool
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range workChan {
				atomic.AddInt64(&scanned, 1)

				has, err := quarantine.HasQuarantine(path)
				if err != nil {
					if !os.IsNotExist(err) {
						mu.Lock()
						res.Errors[path] = err.Error()
						mu.Unlock()
						if opts.OnError != nil {
							opts.OnError(path, err)
						}
					}
					continue
				}

				if !has {
					continue
				}

				atomic.AddInt64(&quarantined, 1)
				mu.Lock()
				res.QuarantinedPaths = append(res.QuarantinedPaths, path)
				mu.Unlock()

				if opts.OnQuarantined != nil {
					info, _ := quarantine.InspectFile(path)
					opts.OnQuarantined(path, info)
				}

				if opts.Strip {
					if opts.DryRun {
						atomic.AddInt64(&processed, 1)
						mu.Lock()
						res.ProcessedPaths = append(res.ProcessedPaths, path)
						mu.Unlock()
						if opts.OnProcessed != nil {
							opts.OnProcessed(path, "dry-run (would remove)")
						}
					} else {
						err := quarantine.RemoveQuarantine(path)
						if err != nil {
							mu.Lock()
							res.Errors[path] = err.Error()
							mu.Unlock()
							if opts.OnError != nil {
								opts.OnError(path, err)
							}
						} else {
							atomic.AddInt64(&processed, 1)
							mu.Lock()
							res.ProcessedPaths = append(res.ProcessedPaths, path)
							mu.Unlock()
							if opts.OnProcessed != nil {
								opts.OnProcessed(path, "removed")
							}
						}
					}
				}
			}
		}()
	}

	// Producer walks the filesystem
	for _, target := range targets {
		absPath, err := filepath.Abs(target)
		if err != nil {
			absPath = target
		}

		fi, err := os.Lstat(absPath)
		if err != nil {
			mu.Lock()
			res.Errors[absPath] = err.Error()
			mu.Unlock()
			if opts.OnError != nil {
				opts.OnError(absPath, err)
			}
			continue
		}

		// If target is a file or symlink, send directly
		if !fi.IsDir() || !opts.Recursive {
			workChan <- absPath
			continue
		}

		// Target is a directory and Recursive is true
		var rootDev uint64
		if !opts.CrossDevice {
			rootDev, err = getDeviceID(absPath)
			if err != nil {
				mu.Lock()
				res.Errors[absPath] = err.Error()
				mu.Unlock()
			}
		}

		// Also check the directory itself
		workChan <- absPath

		_ = filepath.WalkDir(absPath, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if opts.OnError != nil {
					opts.OnError(p, walkErr)
				}
				return nil
			}

			if p == absPath {
				return nil
			}

			if d.IsDir() && !opts.CrossDevice && rootDev != 0 {
				dev, err := getDeviceID(p)
				if err == nil && dev != rootDev {
					return filepath.SkipDir
				}
			}

			// Do not traverse into symlinked directories unless explicit
			if d.Type()&os.ModeSymlink != 0 && !opts.FollowSymlinks {
				workChan <- p
				return nil
			}

			workChan <- p
			return nil
		})
	}

	close(workChan)
	wg.Wait()

	res.TotalScanned = atomic.LoadInt64(&scanned)
	res.TotalQuarantined = atomic.LoadInt64(&quarantined)
	res.TotalProcessed = atomic.LoadInt64(&processed)

	return res, nil
}

func getDeviceID(path string) (uint64, error) {
	var stat unix.Stat_t
	err := unix.Lstat(path, &stat)
	if err != nil {
		return 0, err
	}
	return uint64(stat.Dev), nil
}

// GetTotalQuarantinedCount returns count of items matching com.apple.quarantine
func (r *Result) SummaryString() string {
	return fmt.Sprintf("Scanned: %d | Quarantined: %d | Processed: %d | Errors: %d",
		r.TotalScanned, r.TotalQuarantined, r.TotalProcessed, len(r.Errors))
}
