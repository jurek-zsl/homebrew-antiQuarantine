package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"antiQuarantine/internal/quarantine"
	"antiQuarantine/internal/walker"
)

func setupTestDirectoryTree(t *testing.T) (string, int) {
	tmpDir := t.TempDir()

	// Structure:
	// tmpDir/
	//   file1.txt (quarantined)
	//   file2.txt (clean)
	//   sub1/
	//     file3.dylib (quarantined)
	//     file4.txt (clean)
	//     sub2/
	//       binary (quarantined)

	f1 := filepath.Join(tmpDir, "file1.txt")
	f2 := filepath.Join(tmpDir, "file2.txt")
	s1 := filepath.Join(tmpDir, "sub1")
	s2 := filepath.Join(s1, "sub2")
	f3 := filepath.Join(s1, "file3.dylib")
	f4 := filepath.Join(s1, "file4.txt")
	f5 := filepath.Join(s2, "binary")

	_ = os.MkdirAll(s2, 0755)
	_ = os.WriteFile(f1, []byte("f1"), 0644)
	_ = os.WriteFile(f2, []byte("f2"), 0644)
	_ = os.WriteFile(f3, []byte("f3"), 0644)
	_ = os.WriteFile(f4, []byte("f4"), 0644)
	_ = os.WriteFile(f5, []byte("f5"), 0755)

	_ = quarantine.SetRawQuarantine(f1, sampleQuarantine)
	_ = quarantine.SetRawQuarantine(f3, sampleQuarantine)
	_ = quarantine.SetRawQuarantine(f5, sampleQuarantine)

	return tmpDir, 3 // 3 files quarantined
}

func TestWalkerRecursiveScan(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	root, expectedQuarantined := setupTestDirectoryTree(t)

	res, err := walker.Walk([]string{root}, walker.Options{
		Recursive:   true,
		CrossDevice: true,
		Strip:       false,
	})
	if err != nil {
		t.Fatalf("unexpected walk error: %v", err)
	}

	if res.TotalQuarantined != int64(expectedQuarantined) {
		t.Errorf("expected %d quarantined files, got %d", expectedQuarantined, res.TotalQuarantined)
	}
	if len(res.QuarantinedPaths) != expectedQuarantined {
		t.Errorf("expected %d paths in list, got %d", expectedQuarantined, len(res.QuarantinedPaths))
	}
}

func TestWalkerDryRun(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	root, expectedQuarantined := setupTestDirectoryTree(t)

	res, err := walker.Walk([]string{root}, walker.Options{
		Recursive:   true,
		CrossDevice: true,
		DryRun:      true,
		Strip:       true,
	})
	if err != nil {
		t.Fatalf("unexpected walk error: %v", err)
	}

	if res.TotalProcessed != int64(expectedQuarantined) {
		t.Errorf("expected %d processed in dry-run, got %d", expectedQuarantined, res.TotalProcessed)
	}

	// Verify that attributes were NOT actually removed
	res2, _ := walker.Walk([]string{root}, walker.Options{
		Recursive: true,
		Strip:     false,
	})
	if res2.TotalQuarantined != int64(expectedQuarantined) {
		t.Errorf("expected files to still be quarantined after dry-run, got %d", res2.TotalQuarantined)
	}
}

func TestWalkerActualStrip(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	root, expectedQuarantined := setupTestDirectoryTree(t)

	res, err := walker.Walk([]string{root}, walker.Options{
		Recursive:   true,
		CrossDevice: true,
		DryRun:      false,
		Strip:       true,
	})
	if err != nil {
		t.Fatalf("unexpected walk error: %v", err)
	}

	if res.TotalProcessed != int64(expectedQuarantined) {
		t.Errorf("expected %d processed, got %d", expectedQuarantined, res.TotalProcessed)
	}

	// Verify that all are now clean
	resAfter, _ := walker.Walk([]string{root}, walker.Options{
		Recursive: true,
		Strip:     false,
	})
	if resAfter.TotalQuarantined != 0 {
		t.Errorf("expected 0 quarantined files after strip, got %d", resAfter.TotalQuarantined)
	}
}
