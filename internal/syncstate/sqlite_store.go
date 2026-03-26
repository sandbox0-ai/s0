package syncstate

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

const sqliteStateSchema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS attachments (
	id TEXT PRIMARY KEY,
	workspace_root TEXT NOT NULL,
	volume_id TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	payload_json BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sync_attachments_workspace_root
	ON attachments (workspace_root);

CREATE INDEX IF NOT EXISTS idx_sync_attachments_volume_id
	ON attachments (volume_id);

CREATE TABLE IF NOT EXISTS manifests (
	attachment_id TEXT PRIMARY KEY,
	payload_json BLOB NOT NULL,
	FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
);
`

var (
	stateDBMu   sync.Mutex
	stateDBPath string
	stateDB     *sql.DB
)

func listAttachmentsFromStore() ([]Attachment, error) {
	db, err := openStateDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT payload_json FROM attachments ORDER BY workspace_root`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]Attachment, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		attachment, err := decodeAttachment(payload)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, *attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attachments, nil
}

func loadAttachmentFromStore(id string) (*Attachment, error) {
	db, err := openStateDB()
	if err != nil {
		return nil, err
	}

	var payload []byte
	err = db.QueryRow(`SELECT payload_json FROM attachments WHERE id = ?`, strings.TrimSpace(id)).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return decodeAttachment(payload)
}

func saveAttachmentToStore(attachment *Attachment) error {
	if attachment == nil {
		return errors.New("attachment is nil")
	}
	db, err := openStateDB()
	if err != nil {
		return err
	}

	return withTx(db, func(tx *sql.Tx) error {
		return upsertAttachmentTx(tx, attachment)
	})
}

func updateAttachmentInStore(id string, update func(*Attachment) error) (*Attachment, error) {
	if update == nil {
		return nil, errors.New("update function is nil")
	}
	db, err := openStateDB()
	if err != nil {
		return nil, err
	}

	var updated *Attachment
	err = withTx(db, func(tx *sql.Tx) error {
		attachment, err := loadAttachmentTx(tx, id)
		if err != nil {
			return err
		}
		if attachment.LastSync == nil {
			attachment.LastSync = &SyncCheckpoint{}
		}
		if err := update(attachment); err != nil {
			return err
		}
		if err := upsertAttachmentTx(tx, attachment); err != nil {
			return err
		}
		updated = attachment
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func deleteAttachmentFromStore(id string) error {
	db, err := openStateDB()
	if err != nil {
		return err
	}
	trimmedID := strings.TrimSpace(id)
	if err := withTx(db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM attachments WHERE id = ?`, trimmedID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func loadManifestFromStore(id string) (*Manifest, error) {
	db, err := openStateDB()
	if err != nil {
		return nil, err
	}

	var payload []byte
	err = db.QueryRow(`SELECT payload_json FROM manifests WHERE attachment_id = ?`, strings.TrimSpace(id)).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &Manifest{Entries: map[string]ManifestEntry{}}, nil
		}
		return nil, err
	}
	return decodeManifest(payload)
}

func saveManifestToStore(id string, manifest *Manifest) error {
	if manifest == nil {
		return errors.New("manifest is nil")
	}
	db, err := openStateDB()
	if err != nil {
		return err
	}

	return withTx(db, func(tx *sql.Tx) error {
		return upsertManifestTx(tx, id, manifest)
	})
}

func deleteManifestFromStore(id string) error {
	db, err := openStateDB()
	if err != nil {
		return err
	}
	if err := withTx(db, func(tx *sql.Tx) error {
		_, err := tx.Exec(`DELETE FROM manifests WHERE attachment_id = ?`, strings.TrimSpace(id))
		return err
	}); err != nil {
		return err
	}
	return nil
}

func openStateDB() (*sql.DB, error) {
	path := stateDatabasePath()
	stateDBMu.Lock()
	defer stateDBMu.Unlock()

	if stateDB != nil && stateDBPath == path {
		return stateDB, nil
	}
	if stateDB != nil {
		_ = stateDB.Close()
		stateDB = nil
		stateDBPath = ""
	}

	if err := os.MkdirAll(rootDir(), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := initializeStateDB(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	stateDB = db
	stateDBPath = path
	return stateDB, nil
}

func initializeStateDB(db *sql.DB) error {
	if _, err := db.Exec(sqliteStateSchema); err != nil {
		return err
	}
	return nil
}

func withTx(db *sql.DB, fn func(*sql.Tx) error) error {
	if fn == nil {
		return errors.New("transaction callback is nil")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func loadAttachmentTx(tx *sql.Tx, id string) (*Attachment, error) {
	var payload []byte
	err := tx.QueryRow(`SELECT payload_json FROM attachments WHERE id = ?`, strings.TrimSpace(id)).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return decodeAttachment(payload)
}

func upsertAttachmentTx(tx *sql.Tx, attachment *Attachment) error {
	if attachment == nil {
		return errors.New("attachment is nil")
	}
	if attachment.LastSync == nil {
		attachment.LastSync = &SyncCheckpoint{}
	}
	attachment.UpdatedAt = time.Now().UTC()
	payload, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO attachments (id, workspace_root, volume_id, updated_at, payload_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_root = excluded.workspace_root,
			volume_id = excluded.volume_id,
			updated_at = excluded.updated_at,
			payload_json = excluded.payload_json
	`, attachment.ID, attachment.WorkspaceRoot, attachment.VolumeID, attachment.UpdatedAt.Format(time.RFC3339Nano), payload)
	return err
}

func upsertManifestTx(tx *sql.Tx, id string, manifest *Manifest) error {
	if manifest.Entries == nil {
		manifest.Entries = map[string]ManifestEntry{}
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO manifests (attachment_id, payload_json)
		VALUES (?, ?)
		ON CONFLICT(attachment_id) DO UPDATE SET
			payload_json = excluded.payload_json
	`, strings.TrimSpace(id), payload)
	return err
}

func decodeAttachment(payload []byte) (*Attachment, error) {
	var attachment Attachment
	if err := json.Unmarshal(payload, &attachment); err != nil {
		return nil, err
	}
	if attachment.LastSync == nil {
		attachment.LastSync = &SyncCheckpoint{}
	}
	return &attachment, nil
}

func decodeManifest(payload []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, err
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]ManifestEntry{}
	}
	return &manifest, nil
}

func stateDatabasePath() string {
	return filepath.Join(rootDir(), "state.db")
}
