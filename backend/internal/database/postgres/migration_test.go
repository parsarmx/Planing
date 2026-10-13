package postgres_test

import (
	"context"
	"planningpoker/internal/database/postgres"
	"planningpoker/internal/testutil"
	"testing"
)

func TestMigrationRollbackReapplyAndProviderReopen(t *testing.T) {
	dsn := testutil.Postgres(t)
	ctx := context.Background()
	db, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m, err := postgres.NewMigration(db, "")
	if err != nil {
		t.Fatal(err)
	}
	version, err := m.GetDBVersion(ctx)
	if err != nil || version != 2 {
		t.Fatal("schema version", version, err)
	}
	if _, err = m.Down(ctx); err != nil {
		t.Fatal(err)
	}
	version, err = m.GetDBVersion(ctx)
	if err != nil || version != 1 {
		t.Fatal("one-step rollback", version, err)
	}
	if _, err = m.DownTo(ctx, 0); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err = db.QueryRowContext(ctx, "SELECT to_regclass('rooms') IS NOT NULL").Scan(&exists); err != nil || exists {
		t.Fatal("tables survived rollback", err)
	}
	m, err = postgres.NewMigration(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	status, err := m.Status(ctx)
	if err != nil || len(status) != 2 {
		t.Fatal("migration status", err)
	}
	if _, err = m.Up(ctx); err != nil {
		t.Fatal("repeated migration not idempotent", err)
	}
}
