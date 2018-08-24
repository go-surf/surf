package sqldb

import "testing"

func TestEnsurePostgres(t *testing.T) {
	db, cleanup := EnsurePostgres(t, nil)
	defer cleanup()

	if err := db.Ping(); err != nil {
		t.Fatalf("cannot ping: %s", err)
	}
}

func TestClonePosgres(t *testing.T) {
	db, cleanup := ClonePostgres(t, "postgres", nil)
	defer cleanup()

	if err := db.Ping(); err != nil {
		t.Fatalf("cannot ping: %s", err)
	}
}
