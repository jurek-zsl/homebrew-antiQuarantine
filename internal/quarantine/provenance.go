package quarantine

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Provenance represents origin download details fetched from macOS LaunchServices DB
type Provenance struct {
	EventUUID            string    `json:"event_uuid"`
	Timestamp            time.Time `json:"timestamp,omitempty"`
	AgentBundleID        string    `json:"agent_bundle_id,omitempty"`
	AgentName            string    `json:"agent_name,omitempty"`
	DataURL              string    `json:"data_url,omitempty"`
	OriginURL            string    `json:"origin_url,omitempty"`
	SenderName           string    `json:"sender_name,omitempty"`
	SenderAddress        string    `json:"sender_address,omitempty"`
	TypeNumber           int       `json:"type_number,omitempty"`
}

var (
	quarantineDBPathOnce sync.Once
	quarantineDBPath     string
)

func getQuarantineDBPath() string {
	quarantineDBPathOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err == nil {
			quarantineDBPath = filepath.Join(home, "Library", "Preferences", "com.apple.LaunchServices.QuarantineEventsV2")
		}
	})
	return quarantineDBPath
}

// LookupProvenance queries the macOS QuarantineEventsV2 SQLite database by UUID
func LookupProvenance(uuid string) (*Provenance, error) {
	if uuid == "" {
		return nil, fmt.Errorf("empty UUID")
	}

	dbPath := getQuarantineDBPath()
	if dbPath == "" {
		return nil, fmt.Errorf("unable to determine user home directory")
	}

	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("quarantine events DB not found: %w", err)
	}

	// Open read-only with URI mode to avoid locking
	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open quarantine DB: %w", err)
	}
	defer db.Close()

	query := `SELECT 
		LSQuarantineEventIdentifier,
		LSQuarantineTimeStamp,
		COALESCE(LSQuarantineAgentBundleIdentifier, ''),
		COALESCE(LSQuarantineAgentName, ''),
		COALESCE(LSQuarantineDataURLString, ''),
		COALESCE(LSQuarantineOriginURLString, ''),
		COALESCE(LSQuarantineSenderName, ''),
		COALESCE(LSQuarantineSenderAddress, ''),
		COALESCE(LSQuarantineTypeNumber, 0)
	FROM LSQuarantineEvent
	WHERE LSQuarantineEventIdentifier = ?
	LIMIT 1`

	var (
		id            string
		tsEpoch       sql.NullFloat64
		agentBundleID string
		agentName     string
		dataURL       string
		originURL     string
		senderName    string
		senderAddress string
		typeNum       int
	)

	row := db.QueryRow(query, uuid)
	err = row.Scan(&id, &tsEpoch, &agentBundleID, &agentName, &dataURL, &originURL, &senderName, &senderAddress, &typeNum)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No record found for this UUID
		}
		return nil, fmt.Errorf("querying quarantine DB failed: %w", err)
	}

	prov := &Provenance{
		EventUUID:     id,
		AgentBundleID: agentBundleID,
		AgentName:     agentName,
		DataURL:       dataURL,
		OriginURL:     originURL,
		SenderName:    senderName,
		SenderAddress: senderAddress,
		TypeNumber:    typeNum,
	}

	if tsEpoch.Valid && tsEpoch.Float64 > 0 {
		// macOS Cocoa CoreData timestamps are seconds since Jan 1 2001 (978307200), or unix timestamps
		// LaunchServices typically uses standard Cocoa timestamp or Unix timestamp.
		// If < 1000000000, it's Cocoa Epoch (offset + 978307200).
		epochVal := int64(tsEpoch.Float64)
		if epochVal < 1000000000 && epochVal > 0 {
			prov.Timestamp = time.Unix(epochVal+978307200, 0).UTC()
		} else {
			prov.Timestamp = time.Unix(epochVal, 0).UTC()
		}
	}

	return prov, nil
}
