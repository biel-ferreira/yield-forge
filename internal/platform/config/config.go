package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration, resolved from the environment.
//
// Every field has a sensible default, so the zero-config case still produces a
// runnable app. Secrets (added by later specs) must come from the environment and
// must never be given a hardcoded default.
type Config struct {
	AppEnv          string        // dev | staging | prod
	Port            int           // HTTP listen port
	LogLevel        string        // normalized: debug | info | warn | error
	LogFormat       string        // json | text
	ReadTimeout     time.Duration // HTTP server read timeout
	WriteTimeout    time.Duration // HTTP server write timeout
	IdleTimeout     time.Duration // HTTP server idle (keep-alive) timeout
	ShutdownTimeout time.Duration // graceful shutdown budget

	// Database (SPEC-002). DatabaseURL is a required secret with no default — the
	// app fails fast if it is unset. The remaining knobs tune the connection pool.
	DatabaseURL       string        // postgres://user:pass@host:5432/db?sslmode=...
	DBMaxOpenConns    int           // pool: max open connections
	DBMaxIdleConns    int           // pool: max idle connections
	DBConnMaxLifetime time.Duration // pool: max lifetime of a connection
	DBConnMaxIdleTime time.Duration // pool: max idle time before a connection is closed
	DBConnectTimeout  time.Duration // bounded timeout for the initial connect/ping

	// Warnings holds non-fatal configuration notes (e.g. a value that was invalid
	// and replaced by a default). They should be logged once the logger exists.
	Warnings []string
}

// Defaults for every configurable value.
const (
	defaultAppEnv          = "dev"
	defaultPort            = 8080
	defaultLogLevel        = "info"
	defaultReadTimeout     = 10 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 10 * time.Second

	// Database pool defaults — deliberately conservative to stay within free-tier
	// Postgres connection caps (ADR-0003); tune once a host is chosen.
	defaultDBMaxOpenConns    = 10
	defaultDBMaxIdleConns    = 5
	defaultDBConnMaxLifetime = 30 * time.Minute
	defaultDBConnMaxIdleTime = 5 * time.Minute
	defaultDBConnectTimeout  = 5 * time.Second
)

var validLogLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

// IsDev reports whether the app is running in the development environment.
func (c Config) IsDev() bool { return c.AppEnv == "dev" }

// Addr returns the server listen address (e.g. ":8080").
func (c Config) Addr() string { return fmt.Sprintf(":%d", c.Port) }

// RedactedDatabaseURL returns the database target with all credentials stripped,
// safe for logging. It keeps only scheme, host, port and database name — never the
// user or password. On a parse failure it returns a constant marker rather than
// risk leaking the raw DSN.
func (c Config) RedactedDatabaseURL() string {
	if c.DatabaseURL == "" {
		return ""
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return "[unparseable DATABASE_URL]"
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}

// Load resolves configuration from the environment, applying defaults and
// validation. A local .env file (if present) seeds variables that are not already
// set in the real environment — real environment variables always win.
//
// Fatal problems (a non-numeric port, an unparseable duration) return an error.
// Non-fatal problems (an unknown LOG_LEVEL or LOG_FORMAT) are normalized to a
// default and recorded in Config.Warnings.
func Load() (Config, error) {
	loadDotEnvIfPresent(".env")

	var cfg Config
	var errs []string

	cfg.AppEnv = getString("APP_ENV", defaultAppEnv)

	// Port — fatal if non-numeric or out of range.
	if port, err := getInt("APP_PORT", defaultPort); err != nil {
		errs = append(errs, err.Error())
	} else if port < 1 || port > 65535 {
		errs = append(errs, fmt.Sprintf("APP_PORT must be between 1 and 65535, got %d", port))
	} else {
		cfg.Port = port
	}

	// Log level — non-fatal: an unknown value falls back to info with a warning.
	level := strings.ToLower(getString("LOG_LEVEL", defaultLogLevel))
	if !validLogLevels[level] {
		cfg.Warnings = append(cfg.Warnings,
			fmt.Sprintf("invalid LOG_LEVEL %q; falling back to %q", level, defaultLogLevel))
		level = defaultLogLevel
	}
	cfg.LogLevel = level

	// Log format — default depends on environment; an unknown value is non-fatal.
	defaultFormat := "json"
	if cfg.AppEnv == defaultAppEnv {
		defaultFormat = "text"
	}
	format := strings.ToLower(getString("LOG_FORMAT", defaultFormat))
	if format != "json" && format != "text" {
		cfg.Warnings = append(cfg.Warnings,
			fmt.Sprintf("invalid LOG_FORMAT %q; falling back to %q", format, defaultFormat))
		format = defaultFormat
	}
	cfg.LogFormat = format

	// Database URL — required secret (no default). Missing/empty is fatal (D1).
	cfg.DatabaseURL = getString("DATABASE_URL", "")
	if cfg.DatabaseURL == "" {
		errs = append(errs,
			"DATABASE_URL is required (e.g. postgres://user:pass@host:5432/db?sslmode=disable)")
	}

	// Pool sizes — fatal if non-numeric or negative.
	for _, p := range []struct {
		key string
		def int
		dst *int
	}{
		{"DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns, &cfg.DBMaxOpenConns},
		{"DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns, &cfg.DBMaxIdleConns},
	} {
		n, err := getInt(p.key, p.def)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if n < 0 {
			errs = append(errs, fmt.Sprintf("%s must be >= 0, got %d", p.key, n))
			continue
		}
		*p.dst = n
	}

	// Timeouts and pool durations — fatal if unparseable.
	for _, t := range []struct {
		key string
		def time.Duration
		dst *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", defaultReadTimeout, &cfg.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", defaultWriteTimeout, &cfg.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", defaultIdleTimeout, &cfg.IdleTimeout},
		{"SHUTDOWN_TIMEOUT", defaultShutdownTimeout, &cfg.ShutdownTimeout},
		{"DB_CONN_MAX_LIFETIME", defaultDBConnMaxLifetime, &cfg.DBConnMaxLifetime},
		{"DB_CONN_MAX_IDLE_TIME", defaultDBConnMaxIdleTime, &cfg.DBConnMaxIdleTime},
		{"DB_CONNECT_TIMEOUT", defaultDBConnectTimeout, &cfg.DBConnectTimeout},
	} {
		d, err := getDuration(t.key, t.def)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		*t.dst = d
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(errs, "; "))
	}
	return cfg, nil
}

// getString returns the env value for key, or def when it is unset or empty.
func getString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// getInt returns the integer env value for key, or def when unset/empty. A
// non-numeric value is a fatal error.
func getInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
	}
	return n, nil
}

// getDuration returns the duration env value for key, or def when unset/empty. An
// unparseable value is a fatal error. Accepts Go duration syntax (e.g. 10s, 1m).
func getDuration(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 10s or 1m, got %q", key, v)
	}
	return d, nil
}

// loadDotEnvIfPresent loads KEY=VALUE pairs from path into the process
// environment without overriding variables that are already set. A missing or
// unreadable file is ignored. This is a minimal development convenience, not a
// full dotenv implementation.
func loadDotEnvIfPresent(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
