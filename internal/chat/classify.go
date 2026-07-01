package chat

import (
	"regexp"
	"strconv"
	"strings"
)

// The classifier is a deterministic keyword/amount heuristic (SPEC-108 FR-1083 / §15 tech debt): it
// routes a turn to the engine whose facts ground it, and extracts a contribution amount / horizon.
// It never trusts intent from the client and never invents numbers — an unparseable amount/horizon
// falls back to the general fact set.

var (
	// A value in "R$ 1.500", "R$1.500,00", "2 mil", "2,5 mil", "2000 reais".
	milRe   = regexp.MustCompile(`(\d[\d.]*(?:,\d{1,2})?)\s*mil\b`)
	reaisRe = regexp.MustCompile(`(?:r\$\s*|)(\d[\d.]*(?:,\d{1,2})?)\s*reais\b`)
	brlRe   = regexp.MustCompile(`r\$\s*(\d[\d.]*(?:,\d{1,2})?)`)
	// A horizon in "daqui a 10 anos", "em 5 anos".
	horizonRe = regexp.MustCompile(`(\d{1,2})\s*anos?\b`)
)

// Classify determines the turn's Intent and extracts the contribution amount (centavos) and horizon
// (years). Deterministic: the same text always classifies identically.
func Classify(text string) (intent Intent, amountCentavos int64, horizonYears int) {
	lower := strings.ToLower(text)
	amountCentavos = parseAmountCentavos(lower)
	horizonYears = parseHorizonYears(lower)

	switch {
	case amountCentavos > 0 && hasContributionSignal(lower):
		return IntentContribution, amountCentavos, horizonYears
	case hasProjectionSignal(lower):
		return IntentProjection, amountCentavos, horizonYears
	default:
		return IntentGeneral, 0, 0
	}
}

func hasContributionSignal(s string) bool {
	return strings.Contains(s, "aport") || // aportar / aporte
		strings.Contains(s, "contribu") || // contribuir / contribuição
		(strings.Contains(s, "investir") && (strings.Contains(s, "mês") || strings.Contains(s, "mes")))
}

func hasProjectionSignal(s string) bool {
	return strings.Contains(s, "daqui a") ||
		strings.Contains(s, "renda passiva") ||
		strings.Contains(s, "projeç") || // projeção / projeções
		strings.Contains(s, "patrimôni") || // patrimônio
		(strings.Contains(s, "anos") && (strings.Contains(s, "quanto") || strings.Contains(s, "futuro")))
}

// parseAmountCentavos extracts a BRL amount as int64 centavos ("mil" → ×1000), or 0 if none.
func parseAmountCentavos(s string) int64 {
	if m := milRe.FindStringSubmatch(s); m != nil {
		return brlToCentavos(m[1]) * 1000
	}
	if m := reaisRe.FindStringSubmatch(s); m != nil {
		return brlToCentavos(m[1])
	}
	if m := brlRe.FindStringSubmatch(s); m != nil {
		return brlToCentavos(m[1])
	}
	return 0
}

// parseHorizonYears extracts a horizon in years (1..40), or 0 if none/out of range.
func parseHorizonYears(s string) int {
	m := horizonRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 || n > 40 {
		return 0
	}
	return n
}

// maxAmountReais bounds a parsed amount well beyond any real monthly contribution — it guards the
// centavos (and the ×1000 "mil") arithmetic against int64 overflow on absurd input (SPEC-108 §15).
const maxAmountReais = 1_000_000_000 // R$1 billion

// brlToCentavos parses a Brazilian-formatted number ("1.500", "1.500,00", "2,50", "2") to int64
// centavos. Thousands separators (".") are dropped; the fraction after "," is the cents. Invalid or
// out-of-range → 0.
func brlToCentavos(num string) int64 {
	num = strings.ReplaceAll(num, ".", "") // drop thousands separators
	parts := strings.SplitN(num, ",", 2)
	reais, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || reais < 0 || reais > maxAmountReais {
		return 0
	}
	cents := int64(0)
	if len(parts) == 2 {
		c, err := strconv.ParseInt((parts[1] + "00")[:2], 10, 64) // pad/truncate to 2 digits
		if err != nil {
			return 0
		}
		cents = c
	}
	return reais*100 + cents
}
