package cmd

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"os"
	"planningpoker/app"
	"planningpoker/internal/config"
	"planningpoker/internal/database/postgres"
	"planningpoker/internal/repositories"
	"regexp"
	"time"
)

func NewRoot() *cobra.Command {
	path := "config/config.yml"
	root := &cobra.Command{Use: "poker", Short: "Planning Poker server and PostgreSQL tools", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&path, "config", path, "YAML configuration file")
	load := func() (*config.Config, error) { return config.LoadConfig(path) }
	root.AddCommand(&cobra.Command{Use: "serve", Short: "Serve the API and React app", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := load()
		if err != nil {
			return err
		}
		application := app.New(cfg)
		if err := application.Err(); err != nil {
			return err
		}
		application.Run()
		return nil
	}})
	for _, action := range []string{"pg_migrate", "pg_rollback", "pg_status"} {
		action := action
		var dir string
		var version int64
		c := &cobra.Command{Use: action, Short: map[string]string{"pg_migrate": "Apply Goose migrations", "pg_rollback": "Roll back Goose migrations", "pg_status": "Show migration status"}[action], Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if version < 0 {
				return fmt.Errorf("version cannot be negative")
			}
			cfg, err := load()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()
			db, err := postgres.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer db.Close()
			m, err := postgres.NewMigration(db, dir)
			if err != nil {
				return err
			}
			switch action {
			case "pg_migrate":
				var results []*goose.MigrationResult
				if version == 0 {
					results, err = m.Up(ctx)
				} else {
					results, err = m.UpTo(ctx, version)
				}
				for _, r := range results {
					cmd.Println(r.String())
				}
			case "pg_rollback":
				if cmd.Flags().Changed("version") {
					var results []*goose.MigrationResult
					results, err = m.DownTo(ctx, version)
					for _, r := range results {
						cmd.Println(r.String())
					}
				} else {
					var result *goose.MigrationResult
					result, err = m.Down(ctx)
					if result != nil {
						cmd.Println(result.String())
					}
				}
			case "pg_status":
				var statuses []*goose.MigrationStatus
				statuses, err = m.Status(ctx)
				for _, s := range statuses {
					cmd.Printf("%d\t%s\t%s\n", s.Source.Version, s.State, s.Source.Path)
				}
			}
			return err
		}}
		c.Flags().StringVar(&dir, "dir", "", "Migration directory (default: embedded migrations)")
		if action != "pg_status" {
			c.Flags().Int64Var(&version, "version", 0, "Target migration version")
		}
		root.AddCommand(c)
	}
	var migrationDir string
	createMigration := &cobra.Command{Use: "pg_makemigration [name]", Short: "Create a Goose SQL migration", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(args[0]) {
			return fmt.Errorf("migration name must use lowercase letters, digits and underscores")
		}
		if err := os.MkdirAll(migrationDir, 0755); err != nil {
			return err
		}
		return goose.Create(nil, migrationDir, args[0], "sql")
	}}
	createMigration.Flags().StringVar(&migrationDir, "dir", "internal/database/postgres/migrations", "Migration directory")
	root.AddCommand(createMigration)
	root.AddCommand(&cobra.Command{Use: "pg_create", Short: "Create the configured PostgreSQL database", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := load()
		if err != nil {
			return err
		}
		pc, err := pgx.ParseConfig(cfg.DatabaseURL)
		if err != nil {
			return err
		}
		name := pc.Database
		pc.Database = "postgres"
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		conn, err := pgx.ConnectConfig(ctx, pc)
		if err != nil {
			return err
		}
		defer conn.Close(ctx)
		var exists bool
		if err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", name).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
			if err != nil {
				return err
			}
		}
		cmd.Printf("Database %s is ready.\n", name)
		return nil
	}})
	var file string
	var legacyDB bool
	importCmd := &cobra.Command{Use: "pg_import", Short: "Import legacy JSON into an empty PostgreSQL database", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if (file == "" && !legacyDB) || (file != "" && legacyDB) {
			return fmt.Errorf("choose exactly one of --file or --from-jsonb")
		}
		cfg, err := load()
		if err != nil {
			return err
		}
		store, err := repositories.NewPostgres(cfg)
		if err != nil {
			return err
		}
		defer store.Close()
		var b []byte
		if legacyDB {
			b, err = store.LegacyJSON(cmd.Context())
		} else {
			b, err = os.ReadFile(file)
		}
		if err != nil {
			return err
		}
		result, err := store.Import(cmd.Context(), b)
		if err != nil {
			return err
		}
		cmd.Printf("Imported users=%d rooms=%d sessions=%d OTPs=%d already_imported=%t\n", result.Users, result.Rooms, result.Sessions, result.OTPs, result.AlreadyImported)
		return nil
	}}
	importCmd.Flags().StringVar(&file, "file", "", "Legacy state.json file")
	importCmd.Flags().BoolVar(&legacyDB, "from-jsonb", false, "Read the retired poker_state JSONB table")
	root.AddCommand(importCmd)
	return root
}
func Execute() {
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
