package watcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"antiQuarantine/internal/history"
	"antiQuarantine/internal/quarantine"
	"github.com/fsnotify/fsnotify"
)

// Config configures the background folder watcher
type Config struct {
	Directories   []string
	AutoStrip     bool
	Notify        bool
	Extensions    []string
	Quiet         bool
	DebounceDelay time.Duration
}

// Watcher manages active file monitoring
type Watcher struct {
	cfg       Config
	fsw       *fsnotify.Watcher
	mu        sync.Mutex
	debounces map[string]*time.Timer
}

// New creates a configured Watcher instance
func New(cfg Config) (*Watcher, error) {
	if cfg.DebounceDelay <= 0 {
		cfg.DebounceDelay = 400 * time.Millisecond
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fsnotify watcher: %w", err)
	}

	return &Watcher{
		cfg:       cfg,
		fsw:       fsw,
		debounces: make(map[string]*time.Timer),
	}, nil
}

// Start begins the watcher event loop until the context is cancelled
func (w *Watcher) Start(ctx context.Context) error {
	defer w.fsw.Close()

	for _, dir := range w.cfg.Directories {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			absDir = dir
		}
		if err := w.fsw.Add(absDir); err != nil {
			return fmt.Errorf("failed to watch %s: %w", absDir, err)
		}
		if !w.cfg.Quiet {
			fmt.Printf("👀 Watching directory: %s (auto-strip: %v, notify: %v)\n", absDir, w.cfg.AutoStrip, w.cfg.Notify)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			if !w.cfg.Quiet {
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			}

		case event, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}

			// We are interested in file creations, writes, and metadata changes
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Chmod) {
				w.scheduleCheck(event.Name)
			}
		}
	}
}

func (w *Watcher) scheduleCheck(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// If extension filter is set, check it
	if len(w.cfg.Extensions) > 0 {
		match := false
		ext := strings.ToLower(filepath.Ext(path))
		for _, targetExt := range w.cfg.Extensions {
			if strings.EqualFold(ext, targetExt) || strings.EqualFold(ext, "."+targetExt) {
				match = true
				break
			}
		}
		if !match {
			return
		}
	}

	if timer, exists := w.debounces[path]; exists {
		timer.Stop()
	}

	w.debounces[path] = time.AfterFunc(w.cfg.DebounceDelay, func() {
		w.handleFile(path)
		w.mu.Lock()
		delete(w.debounces, path)
		w.mu.Unlock()
	})
}

func (w *Watcher) handleFile(path string) {
	has, err := quarantine.HasQuarantine(path)
	if err != nil || !has {
		return
	}

	raw, _ := quarantine.GetRawQuarantine(path)
	meta, _ := quarantine.ParseQuarantineString(raw)

	agentName := "Unknown"
	if meta != nil && meta.Agent != "" {
		agentName = meta.Agent
	}

	filename := filepath.Base(path)

	if w.cfg.AutoStrip {
		// Snapshot before stripping
		_ = history.RecordStrip(path, raw)
		err := quarantine.RemoveQuarantine(path)
		if err == nil {
			if !w.cfg.Quiet {
				fmt.Printf("⚡ [Auto-Stripped] %s (Agent: %s)\n", path, agentName)
			}
			if w.cfg.Notify {
				sendNotification(
					fmt.Sprintf("Quarantine Removed: %s", filename),
					fmt.Sprintf("Downloaded by %s. Gatekeeper flag automatically stripped.", agentName),
				)
			}
		}
	} else {
		if !w.cfg.Quiet {
			fmt.Printf("⚠️  [Quarantined File Detected] %s (Agent: %s)\n", path, agentName)
		}
		if w.cfg.Notify {
			sendNotification(
				fmt.Sprintf("Quarantine Detected: %s", filename),
				fmt.Sprintf("Downloaded by %s. Run 'aq strip %s' to unquarantine.", agentName, filename),
			)
		}
	}
}

func sendNotification(title, message string) {
	// Native macOS notification via AppleScript
	script := fmt.Sprintf(`display notification %q with title %q subtitle "antiQuarantine"`, message, title)
	_ = exec.Command("osascript", "-e", script).Start()
}
