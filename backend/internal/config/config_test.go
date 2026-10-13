package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProductionFailsClosedWithoutSMSAndDatabase(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SMS_WEBHOOK_URL", "")
	t.Setenv("COOKIE_SECURE", "false")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted insecure development configuration")
	}
}
func TestYAMLConfigAndEnvironmentOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("environment: development\nserver:\n  host: 127.0.0.1\n  port: '8089'\n  frontend_path: ../web\ndb:\n  dbname: yaml_database\nrealtime:\n  heartbeat: 2s\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_ADDR", "")
	t.Setenv("FRONTEND_PATH", "")
	t.Setenv("DATABASE_URL", "")
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != "127.0.0.1:8089" || c.DB.DBName != "yaml_database" || c.Realtime.IdleTimeout != 24*time.Hour || c.Realtime.Heartbeat != 2*time.Second {
		t.Fatalf("YAML/default configuration incorrect: %+v", c.Realtime)
	}
	if c.FrontendPath != filepath.Join(filepath.Dir(filepath.Dir(path)), "web") {
		t.Fatal("frontend path not relative to config")
	}
	t.Setenv("DB_DBNAME", "override")
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:8089,http://127.0.0.1:8089")
	t.Setenv("REALTIME_IDLE_TIMEOUT", "48h")
	c, err = LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.DB.DBName != "override" || len(c.Origins) != 2 || c.Realtime.IdleTimeout != 48*time.Hour {
		t.Fatal("environment did not override YAML")
	}
	if _, err = LoadConfig(path + ".missing"); err == nil {
		t.Fatal("missing YAML silently ignored")
	}
	t.Setenv("REALTIME_IDLE_TIMEOUT", "0s")
	if _, err = LoadConfig(path); err == nil {
		t.Fatal("unbounded/zero idle timeout accepted")
	}
}
func TestDatabasePasswordIsURLEncoded(t *testing.T) {
	dsn := GetDSN(&DatabaseConfig{User: "poker", Password: "a@b:c/d", Host: "127.0.0.1", Port: "55432", DBName: "poker", SSLMode: "disable"})
	if !strings.Contains(dsn, "a%40b%3Ac%2Fd") {
		t.Fatal("password was not encoded")
	}
}

func TestDefaultLANOriginsAndOverride(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOWED_ORIGINS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"http://192.168.15.87:8080", "http://192.168.15.87:5173", "http://127.0.0.1:8080", "http://localhost:5173"} {
		found := false
		for _, origin := range cfg.Origins {
			if origin == expected {
				found = true
			}
		}
		if !found {
			t.Errorf("missing origin %s", expected)
		}
	}
	t.Setenv("ALLOWED_ORIGINS", "https://poker.example")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Origins) != 1 || cfg.Origins[0] != "https://poker.example" {
		t.Fatal("explicit allowed origins not preserved")
	}
}
