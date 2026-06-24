package insight

import (
	"regexp"
	"strings"
)

// The non-advice gate (SPEC-005 FR-014 / BR-506) rejects an "order signature": a directive
// to TRANSACT — buy/sell/allocate a specific asset, a price or entry/exit target, a
// guaranteed return — as opposed to naming an asset or sector as a *consideration* or
// restating a holding. We match the natural ways an LLM phrases a recommendation
// (imperative AND advisory/infinitive moods, with or without an explicit quantity) while
// letting considerations ("X is worth researching", "increase your fixed-income exposure")
// and holdings ("you hold 100 cotas") through — over-rejection silently neuters legitimate
// portfolio intelligence (PRD FR-019), so the true-negative corpus matters as much as the
// positives.
//
// Recall is biased over precision on purpose: FR-014 is a binding product guard, so a
// borderline directive is better rejected than emitted. Known residual: an order whose verb
// and asset are split across two *sentences* ("Compre. HGLG11 está barato.") can evade the
// proximity windows — the system prompt is the first defense there, and pattern 7 catches
// the common "verb alone on its own line" form.
var (
	// Transaction verbs, imperative + infinitive. Past tense ("comprou"/"vendeu") is
	// deliberately excluded so a holding history is not read as an order.
	buySellVerb = `compre|comprar|venda|vender|vende|adquira|adquirir|buy|buying|sell|selling|purchase`
	// Tokens that turn a transaction verb into an order: a quantity, a B3-style ticker,
	// shares/units, or a whole-position word ("sell all/half", "venda tudo").
	assetToken = `\b\d+\b|\b[a-z]{3,5}\d{1,2}\b|ações|açao|cotas?|shares?|posi[çc][ãa]o|position|tudo|metade|everything|\bhalf\b|\ball\b`
	// A concrete *security* quantity (not a bare number) — the strong signal that an
	// advisory/sizing verb is aimed at a specific holding rather than an asset class.
	qtyToken = `\b[a-z]{3,5}\d{1,2}\b|cotas?|shares?|ações`
	// Advisory verbs (recommend/suggest/advise) and position-sizing verbs (increase/reduce/
	// add/allocate). These only signal an order when aimed at a concrete security (qtyToken)
	// or chained to a transaction verb — so "increase your fixed-income exposure" passes.
	advisoryVerb = `recomend\w*|sugir\w*|suger\w*|sugest\w*|aconselh\w*|recommend\w*|suggest\w*`
	allocVerb    = `aument\w*|reduz\w*|increas\w*|reduc\w*|boost\w*|allocate|\badd\b|\badding\b`
)

// orderPatterns are the order-signature detectors, matched case-insensitively over the
// combined insight text.
var orderPatterns = []*regexp.Regexp{
	// 1. Buy/sell/acquire near an asset, quantity, or whole position (incl. "sell all/half").
	regexp.MustCompile(`(?i)\b(` + buySellVerb + `)\b[^.!?\n]{0,40}?(` + assetToken + `)`),
	// 2. Advisory or position-sizing verb aimed at a specific security quantity/ticker.
	regexp.MustCompile(`(?i)\b(` + advisoryVerb + `|` + allocVerb + `)\b[^.!?\n]{0,40}?(` + qtyToken + `)`),
	// 3. "Recommend/suggest buying/selling/investing" — advisory verb chained to a transaction verb.
	regexp.MustCompile(`(?i)\b(` + advisoryVerb + `)\b[^.!?\n]{0,25}?\b(compr\w*|vend\w*|adquir\w*|buy|buying|sell|selling|purchas\w*|invest\w*|aport\w*)\b`),
	// 4. Buy/sell at a specific price: "compre a R$ 160", "buy at $100", "venda a 120".
	regexp.MustCompile(`(?i)\b(compre|comprar|venda|vender|vende|buy|sell)\b[^.!?\n]{0,25}?\b(a|por|at|@)\b\s*(r\$|us\$|\$)?\s*\d`),
	// 5. Price / fair-value / entry-exit target.
	regexp.MustCompile(`(?i)(pre[çc]o[- ]?alvo|target\s+price|price\s+target|valor\s+justo|fair\s+value|ponto\s+de\s+(entrada|sa[íi]da)|(entry|exit)\s+(point|price))`),
	// 6. Guaranteed / certain return claims (both word orders).
	regexp.MustCompile(`(?i)(retorno|ganho|lucro|rentabilidade|return|profit|yield|gain)s?\s+\w*\s*(garantid[oa]s?|cert[oa]s?|guaranteed|assured|certain)\b`),
	regexp.MustCompile(`(?i)\b(garantid[oa]s?|guaranteed|assured)\b[^.!?\n]{0,20}?(retorno|ganho|lucro|rentabilidade|return|profit|yield|\d+\s*%)`),
	// 7. Standalone imperative buy/sell at a sentence start — catches an order split onto its
	//    own line ("Compre.\nHGLG11 ..."). Only unambiguous imperative forms (no noun homographs).
	regexp.MustCompile(`(?i)(?:^|[.\n]\s*)(compre|comprem|vendam)\b`),
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
