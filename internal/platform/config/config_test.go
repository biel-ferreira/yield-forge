package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testDatabaseURL is a syntactically valid DSN used to satisfy the now-required
// DATABASE_URL (SPEC-002 D1) on success-path tests. It is never connected to.
const testDatabaseURL = "postgres://user:pass@localhost:5432/yieldforge_test?sslmode=disable"

// clearConfigEnv forces every config variable to the "unset" state (empty string,
// which Load treats as unset) so each test starts from defaults deterministically.
// DATABASE_URL is then set to a valid placeholder, since it is required (D1) and an
// unset value would otherwise make every success-path Load() fail.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"APP_ENV", "APP_PORT", "LOG_LEVEL", "LOG_FORMAT",
		"HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT", "SHUTDOWN_TIMEOUT",
		"DATABASE_URL",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS",
		"DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME", "DB_CONNECT_TIMEOUT",
		"SESSION_TTL", "AUTH_COOKIE_NAME",
		"OTEL_SERVICE_NAME", "OTEL_EXPORTER_KIND", "OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_HEADERS", "OTEL_TRACE_SAMPLE_RATIO",
		"INSIGHTER_PROVIDER", "INSIGHTER_OLLAMA_BASE_URL", "INSIGHTER_OLLAMA_MODEL",
		"INSIGHTER_GROQ_BASE_URL", "INSIGHTER_GROQ_API_KEY", "INSIGHTER_GROQ_MODEL",
		"INSIGHTER_TIMEOUT", "INSIGHTER_CACHE_TTL", "INSIGHTER_CACHE_SIZE",
		"MARKETDATA_PROVIDER", "MARKETDATA_FUNDAMENTUS_BASE_URL", "MARKETDATA_YAHOO_BASE_URL",
		"MARKETDATA_BCB_BASE_URL", "MARKETDATA_WATCHLIST", "MARKETDATA_REFRESH_INTERVAL",
		"MARKETDATA_TIMEOUT", "MARKETDATA_SCHEDULER_ENABLED",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("DATABASE_URL", testDatabaseURL)
}

func TestLoad_Defaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != "dev" {
		t.Errorf("AppEnv = %q, want dev", cfg.AppEnv)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text (dev default)", cfg.LogFormat)
	}
	if cfg.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", cfg.ReadTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", cfg.Warnings)
	}
}

func TestLoad_Overrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("HTTP_READ_TIMEOUT", "5s")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppEnv != "prod" {
		t.Errorf("AppEnv = %q, want prod", cfg.AppEnv)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json", cfg.LogFormat)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want 5s", cfg.ReadTimeout)
	}
	if cfg.ShutdownTimeout != 3*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 3s", cfg.ShutdownTimeout)
	}
}

func TestLoad_LogFormatDefaultsToJSONInProd(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "prod") // LOG_FORMAT left unset

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json (prod default)", cfg.LogFormat)
	}
}

func TestLoad_InvalidLogLevelFallsBackWithWarning(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info fallback", cfg.LogLevel)
	}
	if len(cfg.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want exactly one", cfg.Warnings)
	}
	if !strings.Contains(cfg.Warnings[0], "LOG_LEVEL") {
		t.Errorf("warning %q should mention LOG_LEVEL", cfg.Warnings[0])
	}
}

func TestLoad_InvalidLogFormatFallsBackWithWarning(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("LOG_FORMAT", "xml")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text fallback (dev)", cfg.LogFormat)
	}
	if len(cfg.Warnings) != 1 || !strings.Contains(cfg.Warnings[0], "LOG_FORMAT") {
		t.Errorf("Warnings = %v, want one mentioning LOG_FORMAT", cfg.Warnings)
	}
}

func TestLoad_FatalErrors(t *testing.T) {
	cases := []struct {
		name       string
		key, val   string
		wantSubstr string
	}{
		{"non-numeric port", "APP_PORT", "abc", "APP_PORT"},
		{"port out of range", "APP_PORT", "70000", "APP_PORT"},
		{"bad read timeout", "HTTP_READ_TIMEOUT", "nope", "HTTP_READ_TIMEOUT"},
		{"duration without unit", "SHUTDOWN_TIMEOUT", "10", "SHUTDOWN_TIMEOUT"},
		{"non-numeric max open conns", "DB_MAX_OPEN_CONNS", "abc", "DB_MAX_OPEN_CONNS"},
		{"negative max idle conns", "DB_MAX_IDLE_CONNS", "-1", "DB_MAX_IDLE_CONNS"},
		{"bad conn lifetime", "DB_CONN_MAX_LIFETIME", "nope", "DB_CONN_MAX_LIFETIME"},
		{"bad connect timeout", "DB_CONNECT_TIMEOUT", "5", "DB_CONNECT_TIMEOUT"},
		{"invalid otel exporter kind", "OTEL_EXPORTER_KIND", "kafka", "OTEL_EXPORTER_KIND"},
		{"non-numeric sample ratio", "OTEL_TRACE_SAMPLE_RATIO", "lots", "OTEL_TRACE_SAMPLE_RATIO"},
		{"sample ratio out of range", "OTEL_TRACE_SAMPLE_RATIO", "1.5", "OTEL_TRACE_SAMPLE_RATIO"},
		{"invalid insighter provider", "INSIGHTER_PROVIDER", "claude", "INSIGHTER_PROVIDER"},
		{"non-numeric cache size", "INSIGHTER_CACHE_SIZE", "lots", "INSIGHTER_CACHE_SIZE"},
		{"cache size below one", "INSIGHTER_CACHE_SIZE", "0", "INSIGHTER_CACHE_SIZE"},
		{"bad insighter timeout", "INSIGHTER_TIMEOUT", "30", "INSIGHTER_TIMEOUT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(tc.key, tc.val)

			_, err := Load()
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q should mention %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}

func TestLoad_DatabaseDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != testDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, testDatabaseURL)
	}
	if cfg.DBMaxOpenConns != 10 {
		t.Errorf("DBMaxOpenConns = %d, want 10", cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns != 5 {
		t.Errorf("DBMaxIdleConns = %d, want 5", cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime != 30*time.Minute {
		t.Errorf("DBConnMaxLifetime = %v, want 30m", cfg.DBConnMaxLifetime)
	}
	if cfg.DBConnMaxIdleTime != 5*time.Minute {
		t.Errorf("DBConnMaxIdleTime = %v, want 5m", cfg.DBConnMaxIdleTime)
	}
	if cfg.DBConnectTimeout != 5*time.Second {
		t.Errorf("DBConnectTimeout = %v, want 5s", cfg.DBConnectTimeout)
	}
}

func TestLoad_DatabasePoolOverrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "20")
	t.Setenv("DB_MAX_IDLE_CONNS", "0")
	t.Setenv("DB_CONN_MAX_LIFETIME", "1h")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "90s")
	t.Setenv("DB_CONNECT_TIMEOUT", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBMaxOpenConns != 20 {
		t.Errorf("DBMaxOpenConns = %d, want 20", cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns != 0 {
		t.Errorf("DBMaxIdleConns = %d, want 0 (zero is valid)", cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime != time.Hour {
		t.Errorf("DBConnMaxLifetime = %v, want 1h", cfg.DBConnMaxLifetime)
	}
	if cfg.DBConnMaxIdleTime != 90*time.Second {
		t.Errorf("DBConnMaxIdleTime = %v, want 90s", cfg.DBConnMaxIdleTime)
	}
	if cfg.DBConnectTimeout != 2*time.Second {
		t.Errorf("DBConnectTimeout = %v, want 2s", cfg.DBConnectTimeout)
	}
}

func TestLoad_AuthDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Errorf("SessionTTL = %v, want 168h", cfg.SessionTTL)
	}
	if cfg.AuthCookieName != "yf_session" {
		t.Errorf("AuthCookieName = %q, want yf_session", cfg.AuthCookieName)
	}
	// dev (default APP_ENV) → cookie not Secure; prod → Secure.
	if cfg.CookieSecure() {
		t.Error("CookieSecure() = true in dev, want false")
	}
	if !(Config{AppEnv: "prod"}).CookieSecure() {
		t.Error("CookieSecure() = false in prod, want true")
	}
}

func TestLoad_OTELDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OTELServiceName != "yield-forge" {
		t.Errorf("OTELServiceName = %q, want yield-forge", cfg.OTELServiceName)
	}
	// No endpoint => kind "none" => telemetry disabled (no-op).
	if cfg.OTELExporterKind != "none" {
		t.Errorf("OTELExporterKind = %q, want none", cfg.OTELExporterKind)
	}
	if cfg.TelemetryEnabled() {
		t.Error("TelemetryEnabled() = true with no endpoint, want false")
	}
	if cfg.OTELTraceSampleRatio != 1.0 {
		t.Errorf("OTELTraceSampleRatio = %v, want 1.0", cfg.OTELTraceSampleRatio)
	}
}

func TestLoad_OTELEndpointDefaultsToOTLP(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OTELExporterKind != "otlp" {
		t.Errorf("OTELExporterKind = %q, want otlp (derived from endpoint)", cfg.OTELExporterKind)
	}
	if !cfg.TelemetryEnabled() {
		t.Error("TelemetryEnabled() = false with an endpoint, want true")
	}
}

func TestLoad_InsighterDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InsighterProvider != "ollama" {
		t.Errorf("InsighterProvider = %q, want ollama", cfg.InsighterProvider)
	}
	if cfg.InsighterOllamaBaseURL != "http://localhost:11434" {
		t.Errorf("InsighterOllamaBaseURL = %q, want http://localhost:11434", cfg.InsighterOllamaBaseURL)
	}
	if cfg.InsighterTimeout != 30*time.Second {
		t.Errorf("InsighterTimeout = %v, want 30s", cfg.InsighterTimeout)
	}
	if cfg.InsighterCacheTTL != 30*time.Minute {
		t.Errorf("InsighterCacheTTL = %v, want 30m", cfg.InsighterCacheTTL)
	}
	if cfg.InsighterCacheSize != 256 {
		t.Errorf("InsighterCacheSize = %d, want 256", cfg.InsighterCacheSize)
	}
}

func TestLoad_GroqRequiresAPIKey(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("INSIGHTER_PROVIDER", "groq") // no key set

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when provider=groq without an API key")
	}
	if !strings.Contains(err.Error(), "INSIGHTER_GROQ_API_KEY") {
		t.Errorf("error %q should mention INSIGHTER_GROQ_API_KEY", err.Error())
	}
}

func TestLoad_MarketDataDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MarketDataProvider != "fake" {
		t.Errorf("MarketDataProvider = %q, want fake", cfg.MarketDataProvider)
	}
	if cfg.MarketDataFundamentusBaseURL != "https://www.fundamentus.com.br" {
		t.Errorf("MarketDataFundamentusBaseURL = %q", cfg.MarketDataFundamentusBaseURL)
	}
	if cfg.MarketDataRefreshInterval != 24*time.Hour {
		t.Errorf("MarketDataRefreshInterval = %v, want 24h", cfg.MarketDataRefreshInterval)
	}
	if cfg.MarketDataTimeout != 15*time.Second {
		t.Errorf("MarketDataTimeout = %v, want 15s", cfg.MarketDataTimeout)
	}
	if !cfg.MarketDataSchedulerEnabled {
		t.Errorf("MarketDataSchedulerEnabled = false, want true (default)")
	}
	if len(cfg.MarketDataWatchlist) != 0 {
		t.Errorf("MarketDataWatchlist = %v, want empty", cfg.MarketDataWatchlist)
	}
}

func TestLoad_MarketDataWatchlistParsed(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("MARKETDATA_WATCHLIST", " HGLG11 , KNRI11 ,, MXRF11 ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"HGLG11", "KNRI11", "MXRF11"}
	if len(cfg.MarketDataWatchlist) != len(want) {
		t.Fatalf("watchlist = %v, want %v", cfg.MarketDataWatchlist, want)
	}
	for i, w := range want {
		if cfg.MarketDataWatchlist[i] != w {
			t.Errorf("watchlist[%d] = %q, want %q", i, cfg.MarketDataWatchlist[i], w)
		}
	}
}

func TestLoad_MarketDataInvalidProvider(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("MARKETDATA_PROVIDER", "brapi")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "MARKETDATA_PROVIDER") {
		t.Fatalf("expected a MARKETDATA_PROVIDER error, got %v", err)
	}
}

func TestLoad_InsighterDurationsMustBePositive(t *testing.T) {
	for _, key := range []string{"INSIGHTER_TIMEOUT", "INSIGHTER_CACHE_TTL"} {
		t.Run(key, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(key, "0s")

			_, err := Load()
			if err == nil {
				t.Fatalf("expected an error when %s=0s", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error %q should mention %s", err.Error(), key)
			}
		})
	}
}

func TestLoad_InsighterBaseURLMustBeValid(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("INSIGHTER_OLLAMA_BASE_URL", "not-a-valid-url")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for an invalid INSIGHTER_OLLAMA_BASE_URL")
	}
	if !strings.Contains(err.Error(), "INSIGHTER_OLLAMA_BASE_URL") {
		t.Errorf("error %q should mention INSIGHTER_OLLAMA_BASE_URL", err.Error())
	}
}

func TestConfig_LogValueMasksSecrets(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("INSIGHTER_PROVIDER", "groq")
	t.Setenv("INSIGHTER_GROQ_API_KEY", "gsk_supersecret")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer topsecret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("config", "cfg", cfg)
	out := buf.String()

	for _, secret := range []string{"gsk_supersecret", "topsecret"} {
		if strings.Contains(out, secret) {
			t.Errorf("logged config leaked a secret %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("logged config should mask secrets with [REDACTED]: %s", out)
	}
}

func TestLoad_DatabaseURLRequired(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "") // explicitly unset the required secret

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is unset, got nil")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q should mention DATABASE_URL", err.Error())
	}
}

func TestRedactedDatabaseURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strips user and password",
			in:   "postgres://user:s3cr3t@db.example.com:5432/yieldforge?sslmode=require",
			want: "postgres://db.example.com:5432/yieldforge",
		},
		{"empty stays empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Config{DatabaseURL: tc.in}.RedactedDatabaseURL()
			if got != tc.want {
				t.Errorf("RedactedDatabaseURL() = %q, want %q", got, tc.want)
			}
			if strings.Contains(got, "s3cr3t") {
				t.Errorf("redacted URL %q must never contain the password", got)
			}
		})
	}
}

func TestConfigHelpers(t *testing.T) {
	c := Config{AppEnv: "dev", Port: 8080}
	if !c.IsDev() {
		t.Error("IsDev() = false, want true for dev")
	}
	if c.Addr() != ":8080" {
		t.Errorf("Addr() = %q, want :8080", c.Addr())
	}
	if (Config{AppEnv: "prod"}).IsDev() {
		t.Error("IsDev() = true, want false for prod")
	}
}

func TestLoadDotEnvIfPresent_SeedsWithoutOverriding(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	content := "# a comment\n" +
		"YF_TEST_NEW=fromfile\n" +
		"YF_TEST_EXISTING=fromfile\n" +
		"\n" +
		"YF_TEST_QUOTED=\"quoted value\"\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// EXISTING is already set in the real env and must NOT be overridden.
	t.Setenv("YF_TEST_EXISTING", "fromenv")
	// NEW and QUOTED start unset; clean them up afterwards (loadDotEnv uses os.Setenv).
	os.Unsetenv("YF_TEST_NEW")
	os.Unsetenv("YF_TEST_QUOTED")
	t.Cleanup(func() {
		os.Unsetenv("YF_TEST_NEW")
		os.Unsetenv("YF_TEST_QUOTED")
	})

	loadDotEnvIfPresent(envPath)

	if got := os.Getenv("YF_TEST_NEW"); got != "fromfile" {
		t.Errorf("YF_TEST_NEW = %q, want fromfile", got)
	}
	if got := os.Getenv("YF_TEST_EXISTING"); got != "fromenv" {
		t.Errorf("YF_TEST_EXISTING = %q, want fromenv (must not override real env)", got)
	}
	if got := os.Getenv("YF_TEST_QUOTED"); got != "quoted value" {
		t.Errorf("YF_TEST_QUOTED = %q, want 'quoted value' (quotes stripped)", got)
	}
}

func TestLoadDotEnvIfPresent_MissingFileIsNoop(t *testing.T) {
	// Must not panic or error on a missing file.
	loadDotEnvIfPresent(filepath.Join(t.TempDir(), "does-not-exist.env"))
}
