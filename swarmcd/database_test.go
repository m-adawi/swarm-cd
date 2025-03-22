package swarmcd

import (
	"testing"
)

import (
	_ "modernc.org/sqlite"
)

func TestSaveAndLoadLastDeployedRevision(t *testing.T) {
	const dbFile = ":memory:" // Use in-memory database for tests
	db, err := initDB(dbFile)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	stackName := "test-stack"
	revision := "abcdefgh"
	stackContent := []byte("test content")

	version := newVersionFromData(revision, stackContent)
	err = saveLastDeployedRevision(db, stackName, version)
	if err != nil {
		t.Fatalf("Failed to save revision: %v", err)
	}

	loadedVersion, err := loadLastDeployedRevision(db, stackName)
	if err != nil {
		t.Fatalf("Failed to load revision: %v", err)
	}

	expectedHash := computeHash(stackContent)

	if loadedVersion.revision != revision {
		t.Errorf("Expected revision %s, got %s", revision, loadedVersion.revision)
	}

	if loadedVersion.hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, loadedVersion.hash)
	}
}
