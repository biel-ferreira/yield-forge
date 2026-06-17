package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// clearConfigEnv forces every config variable to the "unset" state (empty string,
// which Load treats as unset) so each test starts from defaults deterministically.
func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"APP_ENV", "APP_PORT", "LOG_LEVEL", "LOG_FORMAT",
		"HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT", "SHUTDOWN_TIMEOUT",
	} {
		t.Setenv(k, "")
	}
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
