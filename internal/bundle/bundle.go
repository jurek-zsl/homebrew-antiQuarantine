package bundle

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"antiQuarantine/internal/walker"
)

// Report provides details of bundle analysis, quarantine stripping, and codesigning
type Report struct {
	BundlePath      string   `json:"bundle_path"`
	BundleID        string   `json:"bundle_id,omitempty"`
	BundleName      string   `json:"bundle_name,omitempty"`
	StrippedCount   int      `json:"stripped_count"`
	QuarantinedList []string `json:"quarantined_files,omitempty"`
	Codesigned      bool     `json:"codesigned"`
	CodesignOutput  string   `json:"codesign_output,omitempty"`
	SpctlAssessment string   `json:"spctl_assessment,omitempty"`
	Success         bool     `json:"success"`
	Error           string   `json:"error,omitempty"`
}

// Options controls bundle fixing behavior
type Options struct {
	AdHocCodesign bool
	CheckSpctl    bool
	DryRun        bool
	Verbose       bool
}

// FixBundle sanitizes a macOS .app or .framework bundle
func FixBundle(bundlePath string, opts Options) (*Report, error) {
	absPath, err := filepath.Abs(bundlePath)
	if err != nil {
		absPath = bundlePath
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return &Report{BundlePath: bundlePath, Error: err.Error()}, err
	}

	if !fi.IsDir() {
		return &Report{BundlePath: bundlePath, Error: "target is not a directory/bundle"}, fmt.Errorf("target is not a directory")
	}

	rep := &Report{
		BundlePath: absPath,
		BundleName: filepath.Base(absPath),
	}

	// 1. Walk entire bundle and strip quarantine attributes
	walkRes, err := walker.Walk([]string{absPath}, walker.Options{
		Recursive:      true,
		FollowSymlinks: false,
		CrossDevice:    false,
		DryRun:         opts.DryRun,
		Strip:          true,
	})
	if err != nil {
		rep.Error = err.Error()
		return rep, err
	}

	rep.StrippedCount = int(walkRes.TotalProcessed)
	rep.QuarantinedList = walkRes.QuarantinedPaths

	// 2. Perform Ad-hoc code signing if requested and not in dry-run mode
	if opts.AdHocCodesign && !opts.DryRun {
		cmd := exec.Command("codesign", "--force", "--deep", "--sign", "-", absPath)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			rep.CodesignOutput = strings.TrimSpace(out.String())
			rep.Error = fmt.Sprintf("codesign failed: %v (%s)", err, rep.CodesignOutput)
			return rep, nil
		}
		rep.Codesigned = true
		rep.CodesignOutput = strings.TrimSpace(out.String())
	}

	// 3. Spctl assessment check
	if opts.CheckSpctl {
		cmd := exec.Command("spctl", "--assess", "--type", "execute", "-v", absPath)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		_ = cmd.Run()
		rep.SpctlAssessment = strings.TrimSpace(out.String())
	}

	rep.Success = true
	return rep, nil
}

// IsAppBundle checks if a given directory path is an Apple Bundle
func IsAppBundle(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".app" || ext == ".framework" || ext == ".plugin" || ext == ".bundle" || ext == ".xpc" || ext == ".kext" {
		return true
	}
	// Check if Contents/Info.plist exists
	plist := filepath.Join(path, "Contents", "Info.plist")
	if _, err := os.Stat(plist); err == nil {
		return true
	}
	return false
}
