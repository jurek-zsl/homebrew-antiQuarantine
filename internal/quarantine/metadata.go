package quarantine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Flag constants representing bitmasks in com.apple.quarantine
const (
	FlagQuarantined       uint16 = 0x0001 // File is quarantined / untrusted
	FlagAgentAssessed     uint16 = 0x0002 // Assessed by an anti-malware agent / XProtect
	FlagUserApproved      uint16 = 0x0040 // User explicitly authorized execution
	FlagDownloadedFromWeb uint16 = 0x0080 // Originates from network / browser download
	FlagSandboxed         uint16 = 0x0100 // Generated in sandbox
)

// Metadata represents the parsed 4-field structure of macOS com.apple.quarantine
type Metadata struct {
	Raw         string    `json:"raw"`
	Flags       uint16    `json:"flags"`
	FlagsHex    string    `json:"flags_hex"`
	FlagLabels  []string  `json:"flag_labels"`
	Timestamp   time.Time `json:"timestamp"`
	Agent       string    `json:"agent"`
	EventUUID   string    `json:"event_uuid"`
	IsMalware   bool      `json:"is_quarantined"`
	IsApproved  bool      `json:"is_user_approved"`
	IsFromWeb   bool      `json:"is_downloaded_from_web"`
}

// FileInfo holds full quarantine and filesystem inspection details for a target
type FileInfo struct {
	Path          string      `json:"path"`
	AbsolutePath  string      `json:"absolute_path"`
	IsDir         bool        `json:"is_dir"`
	IsSymlink     bool        `json:"is_symlink"`
	HasQuarantine bool        `json:"has_quarantine"`
	Metadata      *Metadata   `json:"metadata,omitempty"`
	Provenance    *Provenance `json:"provenance,omitempty"`
	Error         string      `json:"error,omitempty"`
}

// ParseQuarantineString decodes the 4-part semicolon-delimited string
func ParseQuarantineString(raw string) (*Metadata, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty quarantine string")
	}

	parts := strings.Split(raw, ";")
	meta := &Metadata{
		Raw: raw,
	}

	if len(parts) >= 1 && parts[0] != "" {
		meta.FlagsHex = parts[0]
		parsedFlags, err := strconv.ParseUint(parts[0], 16, 16)
		if err == nil {
			meta.Flags = uint16(parsedFlags)
			meta.FlagLabels = decodeFlags(meta.Flags)
			meta.IsMalware = (meta.Flags & FlagQuarantined) != 0
			meta.IsApproved = (meta.Flags & FlagUserApproved) != 0
			meta.IsFromWeb = (meta.Flags & FlagDownloadedFromWeb) != 0
		}
	}

	if len(parts) >= 2 && parts[1] != "" {
		parsedTs, err := strconv.ParseInt(parts[1], 16, 64)
		if err == nil {
			meta.Timestamp = time.Unix(parsedTs, 0).UTC()
		}
	}

	if len(parts) >= 3 {
		meta.Agent = parts[2]
	}

	if len(parts) >= 4 {
		meta.EventUUID = parts[3]
	}

	return meta, nil
}

func decodeFlags(flags uint16) []string {
	var labels []string
	if flags&FlagQuarantined != 0 {
		labels = append(labels, "Quarantined/Untrusted (0x0001)")
	}
	if flags&FlagAgentAssessed != 0 {
		labels = append(labels, "Agent Assessed (0x0002)")
	}
	if flags&FlagUserApproved != 0 {
		labels = append(labels, "User Approved (0x0040)")
	}
	if flags&FlagDownloadedFromWeb != 0 {
		labels = append(labels, "Network Download (0x0080)")
	}
	if flags&FlagSandboxed != 0 {
		labels = append(labels, "Sandboxed (0x0100)")
	}
	if len(labels) == 0 {
		labels = append(labels, fmt.Sprintf("Unknown (0x%04X)", flags))
	}
	return labels
}

// InspectFile inspects a target file, reading attributes and resolving provenance
func InspectFile(path string) (*FileInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	fi, err := os.Lstat(path)
	if err != nil {
		return &FileInfo{
			Path:         path,
			AbsolutePath: absPath,
			Error:        err.Error(),
		}, err
	}

	info := &FileInfo{
		Path:         path,
		AbsolutePath: absPath,
		IsDir:        fi.IsDir(),
		IsSymlink:    fi.Mode()&os.ModeSymlink != 0,
	}

	has, err := HasQuarantine(path)
	if err != nil {
		info.Error = err.Error()
		return info, err
	}
	info.HasQuarantine = has

	if has {
		raw, err := GetRawQuarantine(path)
		if err == nil && raw != "" {
			meta, err := ParseQuarantineString(raw)
			if err == nil {
				info.Metadata = meta
				if meta.EventUUID != "" {
					prov, _ := LookupProvenance(meta.EventUUID)
					info.Provenance = prov
				}
			}
		}
	}

	return info, nil
}

// ToJSON converts FileInfo to formatted JSON
func (f *FileInfo) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
