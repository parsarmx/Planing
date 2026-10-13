package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/lock"
	"io/fs"
	"os"
	"regexp"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	db := stdlib.OpenDB(*cfg)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// An empty directory uses embedded migrations, including inside the release image.
func NewMigration(db *sql.DB, dir string) (*goose.Provider, error) {
	var source fs.FS
	if dir == "" {
		var err error
		source, err = fs.Sub(migrations, "migrations")
		if err != nil {
			return nil, err
		}
	} else {
		source = os.DirFS(dir)
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, err
	}
	// Qualify the version table: Goose's existence check otherwise assumes public
	// even when the connection has a different PostgreSQL search_path.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var schema string
	if err = db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
		return nil, err
	}
	if !regexp.MustCompile(`^[a-z_][a-z0-9_]*$`).MatchString(schema) {
		return nil, fmt.Errorf("migration schema must use lowercase letters, digits and underscores")
	}
	store, err := database.NewStore(database.DialectPostgres, schema+".goose_db_version")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider("", db, source, goose.WithStore(store), goose.WithSessionLocker(locker), goose.WithDisableGlobalRegistry(true))
}
