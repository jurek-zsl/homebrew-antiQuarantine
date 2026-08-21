package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"antiQuarantine/internal/quarantine"
	_ "modernc.org/sqlite"
)

// Record represents a single recorded quarantine strip event
type Record struct {
	ID            int64      `json:"id"`
	FilePath      string     `json:"file_path"`
	RawQuarantine string     `json:"raw_quarantine"`
	StrippedAt    time.Time  `json:"stripped_at"`
	RestoredAt    *time.Time `json:"restored_at,omitempty"`
}

var (
	vaultMu   sync.Mutex
	vaultOnce sync.Once
	vaultDB   *sql.DB
	vaultErr  error
)

func getVaultDir() (string, error) {
	if customDir := os.Getenv("AQ_VAULT_DIR"); customDir != "" {
		if err := os.MkdirAll(customDir, 0755); err != nil {
			return "", err
		}
		return customDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "antiQuarantine")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ResetDBForTesting closes the active DB and resets the singleton for test isolation
func ResetDBForTesting() {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	if vaultDB != nil {
		_ = vaultDB.Close()
		vaultDB = nil
	}
	vaultOnce = sync.Once{}
	vaultErr = nil
}

// GetDB returns a lazily initialized singleton connection to the history vault
func GetDB() (*sql.DB, error) {
	vaultMu.Lock()
	defer vaultMu.Unlock()
	vaultOnce.Do(func() {
		dir, err := getVaultDir()
		if err != nil {
			vaultErr = err
			return
		}
		dbPath := filepath.Join(dir, "history.db")
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			vaultErr = err
			return
		}

		schema := `
		CREATE TABLE IF NOT EXISTS quarantine_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT NOT NULL,
			raw_quarantine TEXT NOT NULL,
			stripped_at TIMESTAMP NOT NULL,
			restored_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_history_path ON quarantine_history(file_path);
		CREATE INDEX IF NOT EXISTS idx_history_stripped ON quarantine_history(stripped_at);
		`
		if _, err := db.Exec(schema); err != nil {
			vaultErr = err
			return
		}
		vaultDB = db
	})

	return vaultDB, vaultErr
}

// RecordStrip saves a stripped xattr into the vault
func RecordStrip(path string, rawQuarantine string) error {
	if rawQuarantine == "" {
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	db, err := GetDB()
	if err != nil {
		return err
	}

	query := `INSERT INTO quarantine_history (file_path, raw_quarantine, stripped_at) VALUES (?, ?, ?)`
	_, err = db.Exec(query, absPath, rawQuarantine, time.Now().UTC())
	return err
}

// ListRecent retrieves the latest N stripped records
func ListRecent(limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 20
	}
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, file_path, raw_quarantine, stripped_at, restored_at 
	          FROM quarantine_history 
	          ORDER BY id DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var restoredNull sql.NullTime
		if err := rows.Scan(&r.ID, &r.FilePath, &r.RawQuarantine, &r.StrippedAt, &restoredNull); err != nil {
			return nil, err
		}
		if restoredNull.Valid {
			r.RestoredAt = &restoredNull.Time
		}
		records = append(records, r)
	}
	return records, nil
}

// RestoreLast restores the most recently stripped file from the vault
func RestoreLast() (*Record, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, file_path, raw_quarantine, stripped_at, restored_at 
	          FROM quarantine_history 
	          WHERE restored_at IS NULL
	          ORDER BY id DESC LIMIT 1`
	var r Record
	var restoredNull sql.NullTime
	err = db.QueryRow(query).Scan(&r.ID, &r.FilePath, &r.RawQuarantine, &r.StrippedAt, &restoredNull)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no un-restored history records found in vault")
		}
		return nil, err
	}

	// Reapply xattr
	if err := quarantine.SetRawQuarantine(r.FilePath, r.RawQuarantine); err != nil {
		return nil, fmt.Errorf("failed to restore xattr to %s: %w", r.FilePath, err)
	}

	// Mark as restored
	now := time.Now().UTC()
	_, _ = db.Exec(`UPDATE quarantine_history SET restored_at = ? WHERE id = ?`, now, r.ID)
	r.RestoredAt = &now

	return &r, nil
}

// RestorePath restores quarantine to a specific path using the most recent record
func RestorePath(path string) (*Record, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, file_path, raw_quarantine, stripped_at, restored_at 
	          FROM quarantine_history 
	          WHERE file_path = ? 
	          ORDER BY id DESC LIMIT 1`
	var r Record
	var restoredNull sql.NullTime
	err = db.QueryRow(query, absPath).Scan(&r.ID, &r.FilePath, &r.RawQuarantine, &r.StrippedAt, &restoredNull)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no history record found for %s", path)
		}
		return nil, err
	}

	// Reapply xattr
	if err := quarantine.SetRawQuarantine(r.FilePath, r.RawQuarantine); err != nil {
		return nil, fmt.Errorf("failed to restore xattr to %s: %w", r.FilePath, err)
	}

	now := time.Now().UTC()
	_, _ = db.Exec(`UPDATE quarantine_history SET restored_at = ? WHERE id = ?`, now, r.ID)
	r.RestoredAt = &now

	return &r, nil
}

// ClearHistory empties the vault
func ClearHistory() error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM quarantine_history`)
	return err
}
