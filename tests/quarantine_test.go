package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"antiQuarantine/internal/quarantine"
	"golang.org/x/sys/unix"
)

const sampleQuarantine = "0081;65d8ab12;Safari;B8C27D56-5B81-4C3D-B9AC-06D76D38B1C8"

func TestQuarantineSyscalls(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin xattr syscalls only supported on macOS")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 1. Check clean file
	has, err := quarantine.HasQuarantine(testFile)
	if err != nil {
		t.Fatalf("unexpected error on clean file: %v", err)
	}
	if has {
		t.Fatalf("expected clean file to have no quarantine")
	}

	// 2. Set synthetic quarantine
	err = quarantine.SetRawQuarantine(testFile, sampleQuarantine)
	if err != nil {
		t.Fatalf("failed to set raw quarantine: %v", err)
	}

	// 3. Verify detection
	has, err = quarantine.HasQuarantine(testFile)
	if err != nil {
		t.Fatalf("unexpected error checking quarantine: %v", err)
	}
	if !has {
		t.Fatalf("expected file to have quarantine attribute")
	}

	// 4. Retrieve raw value
	raw, err := quarantine.GetRawQuarantine(testFile)
	if err != nil {
		t.Fatalf("failed to get raw quarantine: %v", err)
	}
	if raw != sampleQuarantine {
		t.Fatalf("quarantine mismatch. Got %q, want %q", raw, sampleQuarantine)
	}

	// 5. List attributes
	attrs, err := quarantine.ListAttributes(testFile)
	if err != nil {
		t.Fatalf("failed to list attributes: %v", err)
	}
	found := false
	for _, a := range attrs {
		if a == quarantine.AttributeName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s in attribute list, got %v", quarantine.AttributeName, attrs)
	}

	// 6. Remove attribute
	err = quarantine.RemoveQuarantine(testFile)
	if err != nil {
		t.Fatalf("failed to remove quarantine: %v", err)
	}

	// 7. Verify removed
	has, err = quarantine.HasQuarantine(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Fatalf("expected quarantine to be removed")
	}
}

func TestSymlinkSafety(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin xattr syscalls only supported on macOS")
	}

	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	symlinkFile := filepath.Join(tmpDir, "symlink.txt")

	_ = os.WriteFile(targetFile, []byte("target"), 0644)
	_ = os.Symlink(targetFile, symlinkFile)

	// Set quarantine on target
	_ = quarantine.SetRawQuarantine(targetFile, sampleQuarantine)

	// HasQuarantine on symlink itself should check the symlink inode (XATTR_NOFOLLOW)
	hasOnLink, _ := quarantine.HasQuarantine(symlinkFile)
	hasOnTarget, _ := quarantine.HasQuarantine(targetFile)

	if !hasOnTarget {
		t.Fatalf("target should have quarantine")
	}

	// Setting xattr directly on symlink with unix.Lsetxattr
	_ = unix.Lsetxattr(symlinkFile, quarantine.AttributeName, []byte(sampleQuarantine), 0)
	hasOnLink, _ = quarantine.HasQuarantine(symlinkFile)
	if !hasOnLink {
		t.Fatalf("symlink should have quarantine")
	}

	// Remove from symlink only
	_ = quarantine.RemoveQuarantine(symlinkFile)
	hasOnLink, _ = quarantine.HasQuarantine(symlinkFile)
	hasOnTarget, _ = quarantine.HasQuarantine(targetFile)

	if hasOnLink {
		t.Fatalf("symlink should no longer have quarantine")
	}
	if !hasOnTarget {
		t.Fatalf("target should still retain quarantine because remove on symlink did not follow link")
	}
}
