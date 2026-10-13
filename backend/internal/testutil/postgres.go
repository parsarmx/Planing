package testutil

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/url"
	"os"
	"planningpoker/internal/database/postgres"
	"planningpoker/pkg"
	"testing"
)

// Each test uses an isolated schema in an explicitly supplied disposable database.
func Postgres(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + pkg.ID()
	if _, err = conn.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		conn.Close(ctx)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	// Add search_path to the URL used by both pgxpool and Goose's sql.DB.
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	testDSN := u.String()
	db, err := postgres.Open(ctx, testDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m, err := postgres.NewMigration(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	return testDSN
}
