package swarmcd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
)

var dbFile = getDBFilePath()

func getDBFilePath() string {
	if path := os.Getenv("SWARMCD_DB"); path != "" {
		return path
	}
	return "/data/revisions.db" // Default path
}

// Ensure database and table exist
func initDB() (*sql.DB, error) {
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
func saveLastDeployedRevision(stackName, revision string, stackContent []byte) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()

	hash := computeHash(stackContent)

	_, err = db.Exec(`
		INSERT INTO revisions (stack, revision, hash) 
		VALUES (?, ?, ?) 
		ON CONFLICT(stack) DO UPDATE SET 
			revision = excluded.revision, 
			hash = excluded.hash
	`, stackName, revision, hash)

	if err != nil {
		return fmt.Errorf("failed to save revision: %w", err)
	}

	return nil
}

// Load a stack's revision and hash
func loadLastDeployedRevision(stackName string) (revision string, hash string, err error) {
	db, err := initDB()
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	err = db.QueryRow(`SELECT revision, hash FROM revisions WHERE stack = ?`, stackName).Scan(&revision, &hash)
	if err == sql.ErrNoRows {
		return "", "", nil
	} else if err != nil {
		return "", "", fmt.Errorf("failed to query revision: %w", err)
	}

	return revision, hash, nil
}

// Compute a SHA-256 hash of the stack content
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}
