package config

import (
	"bufio"
	"fmt"
	"log/slog"
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

	// Auth & sessions (SPEC-003).
	SessionTTL     time.Duration // how long a login session stays valid
	AuthCookieName string        // name of the session cookie

	// Observability (SPEC-004). Telemetry is never required to run: with kind "none"
	// (the default when no endpoint is set) the OTel pipeline is a no-op (BR-401).
	OTELServiceName      string  // resource service.name
	OTELExporterKind     string  // otlp | stdout | none
	OTELExporterEndpoint string  // OTLP endpoint URL; empty => kind defaults to none
	OTELExporterHeaders  string  // OTLP headers (secret), e.g. "authorization=Bearer xyz"
	OTELTraceSampleRatio float64 // sampling probability 0.0..1.0 (NOT a financial rate)

	// AI / Insighter (SPEC-005). The LLM provider is swappable by config; nothing
	// invokes it until the AI feature engine (SPEC-104) wires it in.
	InsighterProvider      string        // ollama | groq | fake
	InsighterOllamaBaseURL string        // local Ollama base URL
	InsighterOllamaModel   string        // Ollama model name
	InsighterGroqBaseURL   string        // Groq OpenAI-compatible base URL
	InsighterGroqAPIKey    string        // Groq API key (secret) — required iff provider=groq
	InsighterGroqModel     string        // Groq model name
	InsighterTimeout       time.Duration // per-generation request timeout
	InsighterCacheTTL      time.Duration // result-cache entry TTL
	InsighterCacheSize     int           // max cached entries (in-memory LRU)

	// Market data (SPEC-006). FII quotes + macro indicators ingested by a background
	// worker; the provider is swappable and the MVP sources (Fundamentus/Yahoo/BCB) need
	// no API key. Market data is global — none of this is per-user.
	MarketDataProvider           string        // fake | live
	MarketDataFundamentusBaseURL string        // FII fundamentals (price/DY/P-VP/segment)
	MarketDataYahooBaseURL       string        // FII last dividend (.SA)
	MarketDataBCBBaseURL         string        // BCB-SGS macro series (SELIC/CDI/IPCA)
	MarketDataWatchlist          []string      // FII tickers to refresh (MVP; holdings-backed later)
	MarketDataRefreshInterval    time.Duration // scheduler cadence (freshness ≤ 24h target)
	MarketDataTimeout            time.Duration // per provider request timeout
	MarketDataSchedulerEnabled   bool          // run the in-process scheduler (off for multi-replica)

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

	// Auth defaults (SPEC-003).
	defaultSessionTTL     = 168 * time.Hour // 7 days
	defaultAuthCookieName = "yf_session"

	// Observability defaults (SPEC-004).
	defaultOTELServiceName      = "yield-forge"
	defaultOTELTraceSampleRatio = 1.0

	// AI / Insighter defaults (SPEC-005).
	defaultInsighterProvider      = "ollama"
	defaultInsighterOllamaBaseURL = "http://localhost:11434"
	defaultInsighterOllamaModel   = "llama3.1"
	defaultInsighterGroqBaseURL   = "https://api.groq.com/openai/v1"
	defaultInsighterGroqModel     = "llama-3.1-8b-instant"
	defaultInsighterTimeout       = 30 * time.Second
	defaultInsighterCacheTTL      = 30 * time.Minute
	defaultInsighterCacheSize     = 256

	// Market data defaults (SPEC-006). Default provider is the deterministic Fake so the
	// zero-config app and CI never hit the network.
	defaultMarketDataProvider           = "fake"
	defaultMarketDataFundamentusBaseURL = "https://www.fundamentus.com.br"
	defaultMarketDataYahooBaseURL       = "https://query1.finance.yahoo.com"
	defaultMarketDataBCBBaseURL         = "https://api.bcb.gov.br/dados/serie"
	defaultMarketDataRefreshInterval    = 24 * time.Hour
	defaultMarketDataTimeout            = 15 * time.Second
	defaultMarketDataSchedulerEnabled   = true
)

var validOTELExporterKinds = map[string]bool{"otlp": true, "stdout": true, "none": true}

var validInsighterProviders = map[string]bool{"ollama": true, "groq": true, "fake": true}

var validMarketDataProviders = map[string]bool{"fake": true, "live": true}

var validLogLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

// IsDev reports whether the app is running in the development environment.
func (c Config) IsDev() bool { return c.AppEnv == "dev" }

// Addr returns the server listen address (e.g. ":8080").
func (c Config) Addr() string { return fmt.Sprintf(":%d", c.Port) }

// CookieSecure reports whether the session cookie should carry the Secure flag.
// True everywhere except local development, where plain HTTP is used (SPEC-003 §10).
func (c Config) CookieSecure() bool { return !c.IsDev() }

// TelemetryEnabled reports whether telemetry is exported. False ("none") means the
// OTel pipeline is a no-op — the app runs identically with no backend (SPEC-004 BR-401).
func (c Config) TelemetryEnabled() bool { return c.OTELExporterKind != "none" }

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

// LogValue implements slog.LogValuer so that logging a Config — even by accident, e.g.
// logger.Info("config", "cfg", cfg) — never leaks a secret: the DSN is redacted and the
// Groq API key and OTLP headers are masked. Defense-in-depth (SPEC-005 security
// hardening); the loader already avoids logging secrets directly.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("app_env", c.AppEnv),
		slog.Int("port", c.Port),
		slog.String("log_level", c.LogLevel),
		slog.String("log_format", c.LogFormat),
		slog.String("database_url", c.RedactedDatabaseURL()),
		slog.String("otel_exporter_kind", c.OTELExporterKind),
		slog.String("otel_exporter_endpoint", c.OTELExporterEndpoint),
		slog.String("otel_exporter_headers", maskSecret(c.OTELExporterHeaders)),
		slog.String("insighter_provider", c.InsighterProvider),
		slog.String("insighter_ollama_base_url", c.InsighterOllamaBaseURL),
		slog.String("insighter_ollama_model", c.InsighterOllamaModel),
		slog.String("insighter_groq_base_url", c.InsighterGroqBaseURL),
		slog.String("insighter_groq_api_key", maskSecret(c.InsighterGroqAPIKey)),
		slog.String("insighter_groq_model", c.InsighterGroqModel),
		slog.String("marketdata_provider", c.MarketDataProvider),
		slog.Bool("marketdata_scheduler_enabled", c.MarketDataSchedulerEnabled),
		slog.Int("marketdata_watchlist_size", len(c.MarketDataWatchlist)),
	)
}

// maskSecret renders a secret for logging — distinguishing unset from set without ever
// revealing the value.
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	return "[REDACTED]"
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

	// Session cookie name (SPEC-003).
	cfg.AuthCookieName = getString("AUTH_COOKIE_NAME", defaultAuthCookieName)

	// Observability (SPEC-004).
	cfg.OTELServiceName = getString("OTEL_SERVICE_NAME", defaultOTELServiceName)
	cfg.OTELExporterEndpoint = getString("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	cfg.OTELExporterHeaders = getString("OTEL_EXPORTER_OTLP_HEADERS", "")

	// Exporter kind — defaults to otlp when an endpoint is set, otherwise none (no-op).
	kind := strings.ToLower(getString("OTEL_EXPORTER_KIND", ""))
	if kind == "" {
		if cfg.OTELExporterEndpoint != "" {
			kind = "otlp"
		} else {
			kind = "none"
		}
	}
	if !validOTELExporterKinds[kind] {
		errs = append(errs, fmt.Sprintf("OTEL_EXPORTER_KIND must be otlp, stdout, or none, got %q", kind))
	}
	cfg.OTELExporterKind = kind

	// Sample ratio — fatal if unparseable or out of [0,1].
	if ratio, err := getFloat("OTEL_TRACE_SAMPLE_RATIO", defaultOTELTraceSampleRatio); err != nil {
		errs = append(errs, err.Error())
	} else if ratio < 0 || ratio > 1 {
		errs = append(errs, fmt.Sprintf("OTEL_TRACE_SAMPLE_RATIO must be between 0 and 1, got %v", ratio))
	} else {
		cfg.OTELTraceSampleRatio = ratio
	}

	// AI / Insighter (SPEC-005).
	cfg.InsighterProvider = strings.ToLower(getString("INSIGHTER_PROVIDER", defaultInsighterProvider))
	if !validInsighterProviders[cfg.InsighterProvider] {
		errs = append(errs, fmt.Sprintf("INSIGHTER_PROVIDER must be ollama, groq, or fake, got %q", cfg.InsighterProvider))
	}
	cfg.InsighterOllamaBaseURL = getString("INSIGHTER_OLLAMA_BASE_URL", defaultInsighterOllamaBaseURL)
	cfg.InsighterOllamaModel = getString("INSIGHTER_OLLAMA_MODEL", defaultInsighterOllamaModel)
	cfg.InsighterGroqBaseURL = getString("INSIGHTER_GROQ_BASE_URL", defaultInsighterGroqBaseURL)
	cfg.InsighterGroqAPIKey = getString("INSIGHTER_GROQ_API_KEY", "")
	cfg.InsighterGroqModel = getString("INSIGHTER_GROQ_MODEL", defaultInsighterGroqModel)
	if cfg.InsighterProvider == "groq" && cfg.InsighterGroqAPIKey == "" {
		errs = append(errs, "INSIGHTER_GROQ_API_KEY is required when INSIGHTER_PROVIDER=groq")
	}
	// Insighter base URLs must be valid http(s) (a typo'd or non-http endpoint silently
	// fails at call time). A hosted (Groq) cleartext endpoint would send portfolio facts
	// over the wire unencrypted, so the active provider is warned about that (SPEC-005 §10).
	for _, u := range []struct {
		key, val string
		groq     bool
	}{
		{"INSIGHTER_OLLAMA_BASE_URL", cfg.InsighterOllamaBaseURL, false},
		{"INSIGHTER_GROQ_BASE_URL", cfg.InsighterGroqBaseURL, true},
	} {
		parsed, err := url.Parse(u.val)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			errs = append(errs, fmt.Sprintf("%s must be a valid http(s) URL, got %q", u.key, u.val))
			continue
		}
		if u.groq && cfg.InsighterProvider == "groq" && parsed.Scheme != "https" {
			cfg.Warnings = append(cfg.Warnings,
				fmt.Sprintf("%s is not https — portfolio facts would be sent in cleartext", u.key))
		}
	}
	// Cache size — fatal if non-numeric or < 1.
	if n, err := getInt("INSIGHTER_CACHE_SIZE", defaultInsighterCacheSize); err != nil {
		errs = append(errs, err.Error())
	} else if n < 1 {
		errs = append(errs, fmt.Sprintf("INSIGHTER_CACHE_SIZE must be >= 1, got %d", n))
	} else {
		cfg.InsighterCacheSize = n
	}

	// Market data (SPEC-006).
	cfg.MarketDataProvider = strings.ToLower(getString("MARKETDATA_PROVIDER", defaultMarketDataProvider))
	if !validMarketDataProviders[cfg.MarketDataProvider] {
		errs = append(errs, fmt.Sprintf("MARKETDATA_PROVIDER must be fake or live, got %q", cfg.MarketDataProvider))
	}
	cfg.MarketDataFundamentusBaseURL = getString("MARKETDATA_FUNDAMENTUS_BASE_URL", defaultMarketDataFundamentusBaseURL)
	cfg.MarketDataYahooBaseURL = getString("MARKETDATA_YAHOO_BASE_URL", defaultMarketDataYahooBaseURL)
	cfg.MarketDataBCBBaseURL = getString("MARKETDATA_BCB_BASE_URL", defaultMarketDataBCBBaseURL)
	cfg.MarketDataWatchlist = splitAndTrim(getString("MARKETDATA_WATCHLIST", ""))
	// Provider base URLs must be valid http(s) (a typo fails silently at call time). Ticker
	// validation is the marketdata package's job, not config's (keeps platform free of feature imports).
	for _, u := range []struct{ key, val string }{
		{"MARKETDATA_FUNDAMENTUS_BASE_URL", cfg.MarketDataFundamentusBaseURL},
		{"MARKETDATA_YAHOO_BASE_URL", cfg.MarketDataYahooBaseURL},
		{"MARKETDATA_BCB_BASE_URL", cfg.MarketDataBCBBaseURL},
	} {
		if parsed, err := url.Parse(u.val); err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			errs = append(errs, fmt.Sprintf("%s must be a valid http(s) URL, got %q", u.key, u.val))
		}
	}
	if b, err := getBool("MARKETDATA_SCHEDULER_ENABLED", defaultMarketDataSchedulerEnabled); err != nil {
		errs = append(errs, err.Error())
	} else {
		cfg.MarketDataSchedulerEnabled = b
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
		{"SESSION_TTL", defaultSessionTTL, &cfg.SessionTTL},
		{"INSIGHTER_TIMEOUT", defaultInsighterTimeout, &cfg.InsighterTimeout},
		{"INSIGHTER_CACHE_TTL", defaultInsighterCacheTTL, &cfg.InsighterCacheTTL},
		{"MARKETDATA_REFRESH_INTERVAL", defaultMarketDataRefreshInterval, &cfg.MarketDataRefreshInterval},
		{"MARKETDATA_TIMEOUT", defaultMarketDataTimeout, &cfg.MarketDataTimeout},
	} {
		d, err := getDuration(t.key, t.def)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		*t.dst = d
	}
	// The Insighter timeout and cache TTL must be positive: a non-positive timeout drops
	// the FR-506 bound, and a non-positive TTL silently disables the cache (SPEC-005).
	if cfg.InsighterTimeout <= 0 {
		errs = append(errs, fmt.Sprintf("INSIGHTER_TIMEOUT must be > 0, got %s", cfg.InsighterTimeout))
	}
	if cfg.InsighterCacheTTL <= 0 {
		errs = append(errs, fmt.Sprintf("INSIGHTER_CACHE_TTL must be > 0, got %s", cfg.InsighterCacheTTL))
	}
	// The market-data interval and timeout must be positive (SPEC-006 FR-608): a non-positive
	// timeout drops the request bound and a non-positive interval would busy-loop the scheduler.
	if cfg.MarketDataRefreshInterval <= 0 {
		errs = append(errs, fmt.Sprintf("MARKETDATA_REFRESH_INTERVAL must be > 0, got %s", cfg.MarketDataRefreshInterval))
	}
	if cfg.MarketDataTimeout <= 0 {
		errs = append(errs, fmt.Sprintf("MARKETDATA_TIMEOUT must be > 0, got %s", cfg.MarketDataTimeout))
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

// getFloat returns the float env value for key, or def when unset/empty. A
// non-numeric value is a fatal error.
func getFloat(key string, def float64) (float64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", key, v)
	}
	return f, nil
}

// getBool returns the boolean env value for key, or def when unset/empty. Accepts the
// strconv.ParseBool forms (1, t, true, 0, f, false, …); anything else is a fatal error.
func getBool(key string, def bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean (true/false), got %q", key, v)
	}
	return b, nil
}

// splitAndTrim splits a comma-separated env value into trimmed, non-empty items. An empty
// or all-whitespace input yields nil.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
		val = parseDotEnvValue(val)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

// parseDotEnvValue extracts the value from the right-hand side of a KEY=VALUE line.
// A quoted value ("..." or '...') is taken verbatim between the quotes, and anything
// after the closing quote (e.g. an inline comment) is dropped. An unquoted value has
// any inline comment — a '#' introduced by whitespace — stripped, then is trimmed.
// This lets `.env.example`'s annotated lines (`KEY=10   # default: 10`) be copied
// verbatim into a working `.env`.
func parseDotEnvValue(raw string) string {
	v := strings.TrimLeft(raw, " \t")
	if v == "" {
		return ""
	}
	if q := v[0]; q == '"' || q == '\'' {
		if end := strings.IndexByte(v[1:], q); end >= 0 {
			return v[1 : 1+end]
		}
		// Unterminated quote: fall back to treating it as an unquoted value.
	}
	// Unquoted: an inline comment starts at the first '#' preceded by whitespace
	// (so a '#' inside the value itself, e.g. a URL fragment, is preserved).
	for i := 1; i < len(v); i++ {
		if v[i] == '#' && (v[i-1] == ' ' || v[i-1] == '\t') {
			v = v[:i]
			break
		}
	}
	return strings.TrimRight(v, " \t")
}
