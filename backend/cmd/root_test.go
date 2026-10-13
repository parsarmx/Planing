package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandArgumentsAndConfigErrors(t *testing.T) {
	for _, args := range [][]string{{"pg_makemigration"}, {"pg_makemigration", "../bad"}, {"pg_import"}, {"pg_import", "--file", "state.json", "--from-jsonb"}, {"serve", "extra"}, {"serve", "--config", "/missing-poker-config.yml"}} {
		root := NewRoot()
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Fatalf("invalid command accepted: %v", args)
		}
	}
	root := NewRoot()
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"serve", "pg_create", "pg_migrate", "pg_rollback", "pg_status", "pg_makemigration", "pg_import"} {
		if !strings.Contains(out.String(), command) {
			t.Fatal("missing command", command)
		}
	}
}
func TestMakeMigrationCreatesDirectoryAndGooseSQL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrations")
	root := NewRoot()
	root.SetArgs([]string{"pg_makemigration", "qa_test", "--dir", path})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(path)
	if err != nil || len(files) != 1 {
		t.Fatal("migration file not created", err)
	}
	b, err := os.ReadFile(filepath.Join(path, files[0].Name()))
	if err != nil || !strings.Contains(string(b), "-- +goose Up") || !strings.Contains(string(b), "-- +goose Down") {
		t.Fatal("invalid Goose file", err)
	}
}
