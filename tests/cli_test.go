package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "aq")
	
	// Get project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	projectRoot := wd
	if filepath.Base(wd) == "tests" {
		projectRoot = filepath.Dir(wd)
	}

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/aq")
	cmd.Env = append(os.Environ(), "GOPATH=/tmp/gopath", "GOCACHE=/tmp/gocache")
	cmd.Dir = projectRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build aq binary for integration test: %v, output: %s", err, string(out))
	}
	return binPath
}

func TestCLILifecycle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	bin := buildBinary(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_cli.zip")
	_ = os.WriteFile(testFile, []byte("dummy zip content"), 0644)

	// Set quarantine using /usr/bin/xattr
	cmd := exec.Command("/usr/bin/xattr", "-w", "com.apple.quarantine", sampleQuarantine, testFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to set synthetic xattr: %v", err)
	}

	// 1. Test aq check
	checkCmd := exec.Command(bin, "check", testFile)
	out, err := checkCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aq check failed: %v, out: %s", err, string(out))
	}
	if !strings.Contains(string(out), "HAS com.apple.quarantine") {
		t.Errorf("expected 'HAS com.apple.quarantine', got: %s", string(out))
	}

	// 2. Test aq inspect --json
	inspectCmd := exec.Command(bin, "inspect", "--json", testFile)
	out, err = inspectCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aq inspect --json failed: %v, out: %s", err, string(out))
	}
	var inspectData []map[string]interface{}
	if err := json.Unmarshal(out, &inspectData); err != nil {
		t.Fatalf("failed to parse inspect JSON: %v, raw: %s", err, string(out))
	}
	if len(inspectData) == 0 || inspectData[0]["has_quarantine"] != true {
		t.Errorf("expected JSON has_quarantine: true, got: %v", inspectData)
	}

	// 3. Test legacy flag aq -rf on directory
	legacyDir := filepath.Join(tmpDir, "legacy_dir")
	_ = os.MkdirAll(legacyDir, 0755)
	legacyFile := filepath.Join(legacyDir, "nested.pkg")
	_ = os.WriteFile(legacyFile, []byte("dummy pkg"), 0644)
	_ = exec.Command("/usr/bin/xattr", "-w", "com.apple.quarantine", sampleQuarantine, legacyFile).Run()

	stripLegacyCmd := exec.Command(bin, "-rf", legacyDir)
	out, err = stripLegacyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aq -rf failed: %v, out: %s", err, string(out))
	}
	if !strings.Contains(string(out), "Removed quarantine from:") {
		t.Errorf("expected 'Removed quarantine from:', got: %s", string(out))
	}

	// Verify legacy file is clean
	checkCleanCmd := exec.Command(bin, legacyFile)
	out, _ = checkCleanCmd.CombinedOutput()
	if !strings.Contains(string(out), "does NOT have com.apple.quarantine") {
		t.Errorf("expected clean verification, got: %s", string(out))
	}

	// 4. Test aq strip on testFile with vault record
	stripCmd := exec.Command(bin, "strip", testFile)
	out, err = stripCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aq strip failed: %v, out: %s", err, string(out))
	}

	// 5. Test aq restore
	restoreCmd := exec.Command(bin, "restore", testFile)
	out, err = restoreCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aq restore failed: %v, out: %s", err, string(out))
	}
	if !strings.Contains(string(out), "Restored quarantine attribute") {
		t.Errorf("expected 'Restored quarantine attribute', got: %s", string(out))
	}

	// Verify restored
	checkRestoredCmd := exec.Command(bin, "check", testFile)
	out, _ = checkRestoredCmd.CombinedOutput()
	if !strings.Contains(string(out), "HAS com.apple.quarantine") {
		t.Errorf("expected file to have quarantine after restore, got: %s", string(out))
	}
}

func TestShellCompletionGeneration(t *testing.T) {
	bin := buildBinary(t)
	for _, shell := range []string{"bash", "zsh", "fish"} {
		cmd := exec.Command(bin, "completion", shell)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to generate %s completion: %v", shell, err)
		}
		if out.Len() == 0 {
			t.Errorf("expected non-empty completion for %s", shell)
		}
	}
}
