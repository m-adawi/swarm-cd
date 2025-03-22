package swarmcd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
)

type version struct {
	revision string
	hash     string
}

func newVersion(revision string, hash string) *version {
	return &version{
		revision: revision,
		hash:     hash,
	}
}

func newVersionFromData(revision string, data []byte) *version {
	return &version{
		revision: revision,
		hash:     computeHash(data),
	}
}

func (version *version) fmtHash() string {
	return fmtHash(version.hash)
}

func getDBFilePath() string {
	if path := os.Getenv("SWARMCD_DB"); path != "" {
		return path
	}
	return "/data/revisions.db" // Default path
}

// Ensure database and table exist
func initDB(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS revisions (
		stack TEXT PRIMARY KEY, 
		revision TEXT, 
		hash TEXT
	)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}

// Save last deployed revision and hash
func saveLastDeployedRevision(db *sql.DB, stackName string, version *version) error {

	_, err := db.Exec(`
		INSERT INTO revisions (stack, revision, hash) 
		VALUES (?, ?, ?) 
		ON CONFLICT(stack) DO UPDATE SET 
			revision = excluded.revision, 
			hash = excluded.hash
	`, stackName, version.revision, version.hash)

	if err != nil {
		return fmt.Errorf("failed to save revision: %w", err)
	}

	return nil
}

// Load a stack's revision and hash
func loadLastDeployedRevision(db *sql.DB, stackName string) (version *version, err error) {
	var revision, hash string
	err = db.QueryRow(`SELECT revision, hash FROM revisions WHERE stack = ?`, stackName).Scan(&revision, &hash)
	if err == sql.ErrNoRows {
		return newVersion("", ""), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revision: %w", err)
	}

	return newVersion(revision, hash), nil
}

// Compute a SHA-256 hash of the stack content
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func fmtHash(hash string) string {
	if len(hash) >= 8 {
		return hash[:8]
	}
	return "<empty-hash>"
}
