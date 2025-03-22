package swarmcd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"time"
)

type stackMetadata struct {
	repoRevision          string
	deployedStackRevision string
	deployedAt            time.Time
	hash                  string
}

func newStackMetadata(repoRevision string, stackRevision string, hash string, time time.Time) *stackMetadata {
	return &stackMetadata{
		repoRevision:          repoRevision,
		deployedStackRevision: stackRevision,
		hash:                  hash,
		deployedAt:            time,
	}
}

func newStackMetadataFromStackData(repoRevision string, stackRevision string, stackData []byte) *stackMetadata {
	return &stackMetadata{
		repoRevision:          repoRevision,
		deployedStackRevision: stackRevision,
		hash:                  computeHash(stackData),
		deployedAt:            time.Now(),
	}
}

func (stackMetadata *stackMetadata) fmtHash() string {
	return fmtHash(stackMetadata.hash)
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
		repo_revision TEXT, 
		deployed_stack_revision TEXT, 
		hash TEXT, 
		deployed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}

// Save last deployed stackMetadata
func saveLastDeployedMetadata(db *sql.DB, stackName string, stackMetadata *stackMetadata) error {
	_, err := db.Exec(`
		INSERT INTO revisions (stack, repo_revision, deployed_stack_revision, hash, deployed_at) 
		VALUES (?, ?, ?, ?, ?) 
		ON CONFLICT(stack) DO UPDATE SET 
			repo_revision = excluded.repo_revision, 
			deployed_stack_revision = excluded.deployed_stack_revision, 
			hash = excluded.hash,
			deployed_at = excluded.deployed_at
	`, stackName, stackMetadata.repoRevision, stackMetadata.deployedStackRevision, stackMetadata.hash, stackMetadata.deployedAt)

	if err != nil {
		return fmt.Errorf("failed to save revision: %w", err)
	}

	return nil
}

// Load a stack's stackMetadata
func loadLastDeployedMetadata(db *sql.DB, stackName string) (*stackMetadata, error) {
	var repoRevision, deployedStackRevision, hash string
	var deployedAt time.Time

	err := db.QueryRow(`
		SELECT repo_revision, deployed_stack_revision, hash, deployed_at 
		FROM revisions 
		WHERE stack = ?`, stackName).Scan(&repoRevision, &deployedStackRevision, &hash, &deployedAt)

	if err == sql.ErrNoRows {
		return newStackMetadata("", "", "", time.Now()), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revision: %w", err)
	}

	return &stackMetadata{
		repoRevision:          repoRevision,
		deployedStackRevision: deployedStackRevision,
		hash:                  hash,
		deployedAt:            deployedAt,
	}, nil
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
