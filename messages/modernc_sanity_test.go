package messages

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

func TestModerncDriverSanity(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sanity.db")
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// PRAGMA foreign_keys is per-connection; every pooled connection is
	// created from the same DSN, so the pragma must hold on any of them.
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	// sqlstore.New runs Upgrade, which errors out when foreign keys are off.
	if _, err := sqlstore.New(context.Background(), "sqlite", dsn, waLog.Noop); err != nil {
		t.Fatalf("sqlstore.New: %v", err)
	}
}
