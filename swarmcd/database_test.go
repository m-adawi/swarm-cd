package swarmcd

import (
	"testing"
	"time"
)

import (
	_ "modernc.org/sqlite"
)

func TestSaveAndLoadLastDeployedRevision(t *testing.T) {
	const dbFile = ":memory:" // Use in-memory database for tests
	db, err := initStackDB(dbFile, "test-stack")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.close()

	repoRevision := "abcdefgh"
	stackRevision := "12345678"
	stackContent := []byte("test content")

	version := newStackMetadataFromStackData(repoRevision, stackRevision, stackContent)
	now := time.Now()
	version.deployedAt = now

	err = db.saveLastDeployedMetadata(version)
	if err != nil {
		t.Fatalf("Failed to save repoRevision: %v", err)
	}

	loadedVersion, err := db.loadLastDeployedMetadata()
	if err != nil {
		t.Fatalf("Failed to load repoRevision: %v", err)
	}

	expectedHash := computeHash(stackContent)

	if loadedVersion.repoRevision != repoRevision {
		t.Errorf("Expected repoRevision %s, got %s", repoRevision, loadedVersion.repoRevision)
	}

	if loadedVersion.deployedStackRevision != stackRevision {
		t.Errorf("Expected repoRevision %s, got %s", repoRevision, loadedVersion.deployedStackRevision)
	}

	if !isRoughlyEqual(loadedVersion.deployedAt, now, 1*time.Microsecond) {
		t.Errorf("Expected time %s, got %s", now, loadedVersion.deployedAt)
	}

	if loadedVersion.hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, loadedVersion.hash)
	}
}

func isRoughlyEqual(t1, t2 time.Time, tolerance time.Duration) bool {
	diff := t2.Sub(t1)
	// Check if the difference is within the tolerance
	if diff < 0 {
		diff = -diff // Handle negative difference
	}
	return diff <= tolerance
}
