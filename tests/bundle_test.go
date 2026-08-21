package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"antiQuarantine/internal/bundle"
	"antiQuarantine/internal/quarantine"
)

func TestAppBundleSanitizer(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	tmpDir := t.TempDir()
	appPath := filepath.Join(tmpDir, "TestApp.app")
	contents := filepath.Join(appPath, "Contents")
	macOS := filepath.Join(contents, "MacOS")
	frameworks := filepath.Join(contents, "Frameworks")

	_ = os.MkdirAll(macOS, 0755)
	_ = os.MkdirAll(frameworks, 0755)

	binPath := filepath.Join(macOS, "TestApp")
	dylibPath := filepath.Join(frameworks, "libhelper.dylib")
	plistPath := filepath.Join(contents, "Info.plist")

	_ = os.WriteFile(binPath, []byte("#!/bin/sh\necho test"), 0755)
	_ = os.WriteFile(dylibPath, []byte("dylib content"), 0644)
	_ = os.WriteFile(plistPath, []byte("<plist></plist>"), 0644)

	// Set quarantine on bundle root, binary, and dylib
	_ = quarantine.SetRawQuarantine(appPath, sampleQuarantine)
	_ = quarantine.SetRawQuarantine(binPath, sampleQuarantine)
	_ = quarantine.SetRawQuarantine(dylibPath, sampleQuarantine)

	if !bundle.IsAppBundle(appPath) {
		t.Fatalf("expected IsAppBundle to be true for TestApp.app")
	}

	// Run FixBundle (without codesign on mock files)
	rep, err := bundle.FixBundle(appPath, bundle.Options{
		AdHocCodesign: false,
		CheckSpctl:    false,
		DryRun:        false,
	})
	if err != nil {
		t.Fatalf("FixBundle failed: %v", err)
	}

	if rep.StrippedCount != 3 {
		t.Errorf("expected 3 files stripped inside bundle, got %d", rep.StrippedCount)
	}

	// Verify all items are now clean
	hasRoot, _ := quarantine.HasQuarantine(appPath)
	hasBin, _ := quarantine.HasQuarantine(binPath)
	hasDylib, _ := quarantine.HasQuarantine(dylibPath)

	if hasRoot || hasBin || hasDylib {
		t.Fatalf("expected all files in bundle to be clean: root=%v, bin=%v, dylib=%v", hasRoot, hasBin, hasDylib)
	}
}
