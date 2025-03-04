package swarmcd

import (
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
)

const dbFile = "/data/revisions.db"

// Ensure database and table exist
func initDB() error {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS revisions (stack TEXT PRIMARY KEY, revision TEXT)`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

func saveLastDeployedRevision(stackName, revision string) error {
	err := initDB() // Ensure DB is initialized
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO revisions (stack, revision) VALUES (?, ?) ON CONFLICT(stack) DO UPDATE SET revision = excluded.revision`, stackName, revision)

	if err != nil {
		return fmt.Errorf("failed to save revision: %w", err)
	}

	return nil
}

// Load a stack's revision
func loadLastDeployedRevision(stackName string) (revision string, err error) {
	err = initDB() // Ensure DB is initialized
	if err != nil {
		return "", err
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	err = db.QueryRow(`SELECT revision FROM revisions WHERE stack = ?`, stackName).Scan(&revision)

	if errors.Is(err, sql.ErrNoRows) {
		// No existing revision found
		return "", nil
	} else if err != nil {
		// Unexpected error
		return "", fmt.Errorf("failed to query revision: %w", err)
	}

	return revision, nil
}
