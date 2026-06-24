package insight

import (
	"regexp"
	"strings"
)

// orderPatterns detect the "order signature" the non-advice gate rejects (SPEC-005
// FR-014 / BR-506): a directive to TRANSACT, not the mere mention of an asset. Each
// pattern targets a high-confidence signal, so a ticker or sector named as a
// *consideration* — or a restatement of a current holding ("you hold 100 cotas") —
// passes. The signal is an imperative buy/sell verb (compre/venda/buy/sell) near a
// quantity/ticker/price, a price/entry-exit target, or a guaranteed-return claim.
// Matching is case-insensitive over the combined insight text.
//
// Note: a bare "100 cotas" is deliberately NOT an order — it reads identically to a
// holding. The transaction VERB is what distinguishes an order from a fact.
var orderPatterns = []*regexp.Regexp{
	// Imperative buy/sell near a quantity, a B3 ticker, or shares/units.
	regexp.MustCompile(`(?i)\b(compre|venda|vende|buy|sell)\b[^.!?\n]{0,30}?(\b\d+\b|\b[a-z]{4}\d{1,2}\b|ações|açao|cotas?|shares?)`),
	// Buy/sell at a specific price: "compre a R$ 160", "buy at $100", "venda a 120".
	regexp.MustCompile(`(?i)\b(compre|venda|vende|buy|sell)\b[^.!?\n]{0,20}?\b(a|por|at|@)\b\s*(r\$|us\$|\$)?\s*\d`),
	// Price / entry-exit target.
	regexp.MustCompile(`(?i)(pre[çc]o[- ]?alvo|target\s+price|price\s+target|ponto\s+de\s+(entrada|sa[íi]da)|(entry|exit)\s+(point|price))`),
	// Guaranteed / certain return claims (both word orders).
	regexp.MustCompile(`(?i)(retorno|ganho|lucro|rentabilidade|return|profit|yield|gain)s?\s+\w*\s*(garantid[oa]s?|cert[oa]s?|guaranteed|assured|certain)\b`),
	regexp.MustCompile(`(?i)\b(garantid[oa]s?|guaranteed|assured)\b[^.!?\n]{0,20}?(retorno|ganho|lucro|rentabilidade|return|profit|yield|\d+\s*%)`),
}

// containsOrder reports whether text contains an order signature (FR-014).
func containsOrder(text string) bool {
	for _, p := range orderPatterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

// insightText concatenates an insight's user-facing fields for validation. Category is
// included too, so an order can't slip through a field that isn't an enum yet (the
// allow-list lands with SPEC-104).
func insightText(in Insight) string {
	return strings.Join([]string{in.Category, in.Title, in.Detail, in.Explanation}, "\n")
}
