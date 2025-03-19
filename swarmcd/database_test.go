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

	stackName := "test-stack"
	revision := "v1.0.0"
	stackContent := []byte("test content")

	err = saveLastDeployedRevision(db, stackName, revision, stackContent)
	if err != nil {
		t.Fatalf("Failed to save revision: %v", err)
	}

	loadedRevision, loadedHash, err := loadLastDeployedRevision(db, stackName)
	if err != nil {
		t.Fatalf("Failed to load revision: %v", err)
	}

	expectedHash := computeHash(stackContent)

	if loadedRevision != revision {
		t.Errorf("Expected revision %s, got %s", revision, loadedRevision)
	}

	if loadedHash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, loadedHash)
	}
}
