package swarmcd

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

const dbFile = "/data/revisions.db"

func saveRevisionDB(stackName, revision string) error {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS revisions (stack TEXT PRIMARY KEY, revision TEXT)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO revisions (stack, revision) VALUES (?, ?) ON CONFLICT(stack) DO UPDATE SET revision = excluded.revision`, stackName, revision)
	return err
}

func loadRevisionDB(stackName string) string {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return ""
	}
	defer db.Close()

	var revision string
	err = db.QueryRow(`SELECT revision FROM revisions WHERE stack = ?`, stackName).Scan(&revision)
	if err != nil {
		return ""
	}
	return revision
}
