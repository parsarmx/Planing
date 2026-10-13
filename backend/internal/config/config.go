package config

import (
	"fmt"
	"github.com/spf13/viper"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ServerConfig struct {
	Host           string   `mapstructure:"host"`
	Port           string   `mapstructure:"port"`
	FrontendPath   string   `mapstructure:"frontend_path"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	SecureCookies  bool     `mapstructure:"secure_cookies"`
}
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
	MaxConns int32  `mapstructure:"max_conns"`
	MinConns int32  `mapstructure:"min_conns"`
	URL      string `mapstructure:"url"`
}
type SMSConfig struct {
	URL   string `mapstructure:"webhook_url"`
	Token string `mapstructure:"webhook_token"`
}
type RealtimeConfig struct {
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
	Heartbeat   time.Duration `mapstructure:"heartbeat"`
}
type Config struct {
	Environment string         `mapstructure:"environment"`
	Server      ServerConfig   `mapstructure:"server"`
	DB          DatabaseConfig `mapstructure:"db"`
	SMS         SMSConfig      `mapstructure:"sms"`
	Realtime    RealtimeConfig `mapstructure:"realtime"`
	// Resolved values consumed by the application and transport adapters.
	Addr, DatabaseURL, FrontendPath, SMSURL, SMSToken string   `mapstructure:"-"`
	Origins                                           []string `mapstructure:"-"`
	SecureCookies                                     bool     `mapstructure:"-"`
}

func defaults() *viper.Viper {
	v := viper.New()
	v.SetDefault("environment", "development")
	for k, value := range map[string]any{
		"server.host": "0.0.0.0", "server.port": "8080", "server.frontend_path": "../../frontend/dist",
		"server.allowed_origins": []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://192.168.15.87:8080", "http://localhost:5173", "http://127.0.0.1:5173", "http://192.168.15.87:5173"},
		"server.secure_cookies":  false,
		"db.host":                "127.0.0.1", "db.port": "55432", "db.user": "poker", "db.password": "local-poker-password", "db.dbname": "poker", "db.sslmode": "disable", "db.max_conns": 20, "db.min_conns": 1, "db.url": "",
		"sms.webhook_url": "", "sms.webhook_token": "", "realtime.idle_timeout": "24h", "realtime.heartbeat": "10s",
	} {
		v.SetDefault(k, value)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	for key, env := range map[string]string{
		"environment": "APP_ENV", "server.allowed_origins": "ALLOWED_ORIGINS", "server.secure_cookies": "COOKIE_SECURE",
		"server.frontend_path": "FRONTEND_PATH", "db.url": "DATABASE_URL", "sms.webhook_url": "SMS_WEBHOOK_URL", "sms.webhook_token": "SMS_WEBHOOK_TOKEN",
	} {
		_ = v.BindEnv(key, env, strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
	}
	return v
}

// Load provides defaults for in-memory tests. The CLI always uses LoadConfig.
func Load() (*Config, error) { return decode(defaults(), "") }
func LoadConfig(path string) (*Config, error) {
	v := defaults()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read configuration: %w", err)
	}
	return decode(v, path)
}
func decode(v *viper.Viper, path string) (*Config, error) {
	c := &Config{}
	if err := v.Unmarshal(c); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}
	c.Addr = net.JoinHostPort(c.Server.Host, c.Server.Port)
	if addr := os.Getenv("APP_ADDR"); addr != "" {
		c.Addr = addr
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return nil, fmt.Errorf("invalid server address")
	}
	c.Origins = c.Server.AllowedOrigins
	for i := range c.Origins {
		c.Origins[i] = strings.TrimSpace(c.Origins[i])
		u, err := url.Parse(c.Origins[i])
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" {
			return nil, fmt.Errorf("invalid allowed origin")
		}
	}
	c.FrontendPath = c.Server.FrontendPath
	if path != "" && !filepath.IsAbs(c.FrontendPath) && os.Getenv("FRONTEND_PATH") == "" && os.Getenv("SERVER_FRONTEND_PATH") == "" {
		c.FrontendPath = filepath.Join(filepath.Dir(path), c.FrontendPath)
	}
	c.FrontendPath, _ = filepath.Abs(c.FrontendPath)
	c.DatabaseURL = c.DB.URL
	if c.DatabaseURL == "" {
		c.DatabaseURL = GetDSN(&c.DB)
	}
	c.SMSURL, c.SMSToken = c.SMS.URL, c.SMS.Token
	c.SecureCookies = c.Server.SecureCookies
	if c.Environment != "development" && c.Environment != "production" {
		return nil, fmt.Errorf("environment must be development or production")
	}
	if c.DB.MaxConns < 1 || c.DB.MinConns < 0 || c.DB.MinConns > c.DB.MaxConns {
		return nil, fmt.Errorf("invalid database pool limits")
	}
	if c.Realtime.IdleTimeout <= 0 || c.Realtime.Heartbeat <= 0 {
		return nil, fmt.Errorf("realtime durations must be positive")
	}
	if c.Environment == "production" && (!c.SecureCookies || c.DatabaseURL == "" || !strings.HasPrefix(c.SMSURL, "https://") || c.SMSToken == "") {
		return nil, fmt.Errorf("production requires PostgreSQL, secure cookies and HTTPS SMS webhook with a token")
	}
	return c, nil
}
func GetDSN(c *DatabaseConfig) string {
	u := url.URL{Scheme: "postgres", User: url.UserPassword(c.User, c.Password), Host: net.JoinHostPort(c.Host, c.Port), Path: "/" + c.DBName}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
