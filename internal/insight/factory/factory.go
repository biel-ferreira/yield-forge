package factory

import (
	"log/slog"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/insight/groq"
	"github.com/biel-ferreira/yield-forge/internal/insight/ollama"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// New builds the Insighter from config as observed(cached(gated(provider))):
//   - provider — the LLM adapter selected by INSIGHTER_PROVIDER (ollama | groq | fake);
//   - gated    — the binding-guard gates (explainability + non-advice), so the guards
//     are unavoidable regardless of provider (SPEC-005 BR-501);
//   - cached   — the in-memory result cache (stores gated results, avoids LLM calls);
//   - observed — AI telemetry (span + generation counter, no prompt/PII).
//
// Nothing invokes the returned Insighter until the AI feature engine (SPEC-104) wires it
// into a feature.
func New(cfg config.Config, logger *slog.Logger, clk clock.Clock) insight.Insighter {
	provider, model := newProvider(cfg)

	chain := insight.Gated(provider, logger)
	chain = cached{next: chain, cache: newMemCache(cfg.InsighterCacheSize, cfg.InsighterCacheTTL, clk)}
	return newObserved(chain, cfg.InsighterProvider, model)
}

// newProvider selects the LLM adapter and the model label (for telemetry) from config.
func newProvider(cfg config.Config) (insight.Insighter, string) {
	switch cfg.InsighterProvider {
	case "groq":
		return groq.New(cfg.InsighterGroqBaseURL, cfg.InsighterGroqAPIKey, cfg.InsighterGroqModel, cfg.InsighterTimeout),
			cfg.InsighterGroqModel
	case "fake":
		return insight.Fake{}, "fake"
	default: // ollama
		return ollama.New(cfg.InsighterOllamaBaseURL, cfg.InsighterOllamaModel, cfg.InsighterTimeout),
			cfg.InsighterOllamaModel
	}
}
