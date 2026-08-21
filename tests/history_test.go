package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"antiQuarantine/internal/history"
	"antiQuarantine/internal/quarantine"
)

func TestHistoryVaultAndRestore(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "vault_test.bin")
	_ = os.WriteFile(testFile, []byte("data"), 0644)

	// Clear history for clean test run
	_ = history.ClearHistory()

	// Set quarantine
	_ = quarantine.SetRawQuarantine(testFile, sampleQuarantine)

	// Read & record
	raw, _ := quarantine.GetRawQuarantine(testFile)
	err := history.RecordStrip(testFile, raw)
	if err != nil {
		t.Fatalf("failed to record strip: %v", err)
	}

	// Remove quarantine
	_ = quarantine.RemoveQuarantine(testFile)

	has, _ := quarantine.HasQuarantine(testFile)
	if has {
		t.Fatalf("quarantine should be removed")
	}

	// Check history list
	records, err := history.ListRecent(5)
	if err != nil {
		t.Fatalf("failed to list history: %v", err)
	}
	if len(records) == 0 {
		t.Fatalf("expected at least 1 history record")
	}

	// Restore last
	rec, err := history.RestoreLast()
	if err != nil {
		t.Fatalf("restore last failed: %v", err)
	}
	if rec.FilePath != testFile {
		t.Errorf("restored file mismatch: got %s, want %s", rec.FilePath, testFile)
	}

	// Verify file is now quarantined again
	hasAfter, _ := quarantine.HasQuarantine(testFile)
	if !hasAfter {
		t.Fatalf("expected file to have quarantine restored")
	}

	restoredRaw, _ := quarantine.GetRawQuarantine(testFile)
	if restoredRaw != sampleQuarantine {
		t.Errorf("restored value mismatch: got %q, want %q", restoredRaw, sampleQuarantine)
	}
}
