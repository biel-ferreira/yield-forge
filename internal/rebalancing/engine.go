package rebalancing

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
)

// Options carries per-request rebalancing flags (SPEC-105 FR-1056).
type Options struct {
	// IncludeAssetShares opts into the illustrative per-candidate within-area share (D6). Default
	// off — the natural caller that sets it is the Conversational Copilot (SPEC-108).
	IncludeAssetShares bool
}

// Service is the rebalancing engine (SPEC-105 §6). It reuses the Fact Builder seam, computes the
// split deterministically, and emits guidance ONLY through the gated Insighter — explainability
// (FR-013) and non-advice (FR-014) hold by construction (BR-1054). The computed numbers are never
// produced by the LLM; the grounding guard drops any hallucinated candidate ticker (BR-1053).
type Service struct {
	facts     FactSource
	universe  UniverseReader
	insighter insight.Insighter
	tracer    trace.Tracer
}

// NewService builds the engine over the Fact Builder seam, the FII universe reader, and the
// (gated) Insighter.
func NewService(facts FactSource, universe UniverseReader, insighter insight.Insighter) *Service {
	return &Service{facts: facts, universe: universe, insighter: insighter, tracer: observability.Tracer("rebalancing")}
}

// Rebalance produces the contribution guidance for userID. It builds the facts (reusing
// BuildFacts), computes the split, then asks the Insighter once per area for that area's gated
// explanation and once for grounded candidates. An area whose generation fails is skipped; a
// candidate whose ticker is not in the universe is dropped. Available is false only when every
// area failed (full outage); an empty portfolio still produces guidance (FR-1053).
func (s *Service) Rebalance(ctx context.Context, userID string, contribution Contribution, opts Options) (Rebalancing, error) {
	// Span over fact-building (reuse seam + universe + computed split) for latency visibility. It
	// carries NO content — no contribution amount, figures, or generated text (BR-505/FR-1058).
	facts, split, universe, err := s.buildFacts(ctx, userID, contribution)
	if err != nil {
		return Rebalancing{}, err
	}
	return s.generate(ctx, userID, facts, split, universe, opts)
}

// BuildContributionFacts returns the DETERMINISTIC grounding facts for a contribution — the base
// portfolio facts + the FII universe + the computed split — WITHOUT any LLM call. It is the seam the
// Conversational Copilot (SPEC-108) uses to ground a "tenho R$X pra aportar" chat turn without
// re-running the per-area rebalancing LLM (no double-generate). A non-positive amount is an error.
func (s *Service) BuildContributionFacts(ctx context.Context, userID string, amountCentavos int64) (insight.Facts, error) {
	contribution, err := ParseContribution(amountCentavos)
	if err != nil {
		return nil, fmt.Errorf("contribution facts: %w", err)
	}
	facts, _, _, err := s.buildFacts(ctx, userID, contribution)
	return facts, err
}

// buildFacts reuses the published seam + the FII universe and computes the split, all inside the
// rebalancing.facts span (FR-1058).
func (s *Service) buildFacts(ctx context.Context, userID string, contribution Contribution) (insight.Facts, []AreaShare, []marketdata.FIIQuote, error) {
	factCtx, span := s.tracer.Start(ctx, "rebalancing.facts")
	defer span.End()

	base, err := s.facts.BuildFacts(factCtx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("rebalance: %w", err)
	}
	universe, err := s.universe.ListFIIUniverse(factCtx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("rebalance: %w", err)
	}
	facts, split := assembleFacts(base, contribution, universe)
	return facts, split, universe, nil
}

// generate runs the gated Insighter over the assembled facts (areas then candidates).
func (s *Service) generate(ctx context.Context, userID string, facts insight.Facts, split []AreaShare, universe []marketdata.FIIQuote, opts Options) (Rebalancing, error) {
	areas, succeeded, err := s.buildAreas(ctx, userID, facts, split)
	if err != nil {
		return Rebalancing{}, err
	}
	candidates, dropped, err := s.buildCandidates(ctx, userID, facts, universe, areaShareBps(split, areaClassFII), opts)
	if err != nil {
		return Rebalancing{}, err
	}

	return Rebalancing{
		Areas:             areas,
		Candidates:        candidates,
		DroppedCandidates: dropped,
		Disclaimer:        insight.Disclaimer,
		Available:         succeeded > 0, // false only when every area failed (full outage)
	}, nil
}

const areaClassFII = "fii"

// buildAreas asks the Insighter for each computed area's gated explanation, joining the
// authoritative computed numbers onto the LLM's text (the LLM explains, never produces, the split).
func (s *Service) buildAreas(ctx context.Context, userID string, facts insight.Facts, split []AreaShare) ([]Area, int, error) {
	var areas []Area
	succeeded := 0
	for _, sh := range split {
		res, err := s.insighter.Generate(ctx, insight.InsightRequest{
			Facts:  withFocus(facts, "area", string(sh.Class)),
			Task:   TaskRebalancing,
			UserID: userID,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, 0, fmt.Errorf("rebalance: %w", ctx.Err())
			}
			continue // this area degraded or was gate-rejected — skip it (FR-1057)
		}
		// The gate only guarantees an explanation for insights PRESENT — a successful-but-empty
		// result (or a blank explanation) would leak an unexplained area, so guard here (FR-013).
		if !explained(res.Insights) {
			continue
		}
		succeeded++
		in := res.Insights[0]
		areas = append(areas, Area{
			Class:                   string(sh.Class),
			SuggestedShareBps:       sh.SuggestedShareBps,
			SuggestedAmountCentavos: sh.SuggestedAmountCentavos,
			Title:                   in.Title,
			Detail:                  in.Detail,
			Explanation:             in.Explanation,
		})
	}
	return areas, succeeded, nil
}

// buildCandidates asks the Insighter for named candidates and applies the grounding guard: only a
// ticker present in the universe survives (BR-1053). When opted in, the FII area's share is split
// evenly across the surfaced candidates as an illustrative within-area consideration (D6).
// buildCandidates returns the grounded candidates plus the count of candidates the grounding guard
// dropped for naming a ticker the system does not know (a hallucination signal surfaced for
// edge-level telemetry, BR-1053).
func (s *Service) buildCandidates(ctx context.Context, userID string, facts insight.Facts, universe []marketdata.FIIQuote, fiiAreaBps int, opts Options) ([]Candidate, int, error) {
	known := tickerIndex(universe)
	res, err := s.insighter.Generate(ctx, insight.InsightRequest{
		Facts:  withFocus(facts, "candidates", ""),
		Task:   TaskRebalancing,
		UserID: userID,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, fmt.Errorf("rebalance: %w", ctx.Err())
		}
		return nil, 0, nil // candidates are optional — a degraded candidates call is not fatal
	}

	var candidates []Candidate
	dropped := 0
	for _, in := range res.Insights {
		q, ok := known[normalizeTicker(in.Title)]
		if !ok {
			dropped++ // grounding guard: the assistant never names a ticker the system doesn't know
			continue
		}
		if strings.TrimSpace(in.Explanation) == "" {
			continue // no unexplained candidate reaches the user (FR-013)
		}
		candidates = append(candidates, Candidate{
			Ticker:      q.Ticker.String(),
			Sector:      string(q.Sector),
			Title:       in.Title,
			Detail:      in.Detail,
			Explanation: in.Explanation,
		})
	}
	if opts.IncludeAssetShares {
		assignIllustrativeShares(candidates, fiiAreaBps)
	}
	return candidates, dropped, nil
}

// withFocus returns a shallow copy of facts with the focus keys added, so each Insighter call has
// a distinct cache key and the prompt knows what to reason about. The originals are read-only.
func withFocus(facts insight.Facts, kind, area string) insight.Facts {
	out := make(insight.Facts, len(facts)+2)
	for k, v := range facts {
		out[k] = v
	}
	out["focus_kind"] = kind
	if area != "" {
		out["focus_area"] = area
	}
	return out
}

// tickerIndex maps normalized ticker → quote for the grounding guard.
func tickerIndex(universe []marketdata.FIIQuote) map[string]marketdata.FIIQuote {
	out := make(map[string]marketdata.FIIQuote, len(universe))
	for _, q := range universe {
		out[normalizeTicker(q.Ticker.String())] = q
	}
	return out
}

func normalizeTicker(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// explained reports whether the gated result carries a usable explanation. The gate guarantees an
// explanation only for insights PRESENT, so a successful-but-empty result must be treated as
// having no explanation — never leak an unexplained suggestion to the user (FR-013).
func explained(insights []insight.Insight) bool {
	return len(insights) > 0 && strings.TrimSpace(insights[0].Explanation) != ""
}

// areaShareBps returns the suggested share (bps) of the named area class, or 0 if absent.
func areaShareBps(split []AreaShare, class string) int {
	for _, a := range split {
		if string(a.Class) == class {
			return a.SuggestedShareBps
		}
	}
	return 0
}

// assignIllustrativeShares distributes the FII area's bps evenly across the candidates (largest
// remainder to the first), as an illustrative within-area consideration — never an order (D6). The
// candidate order is the LLM's, so which candidate absorbs the remainder is not stable across
// providers; that is acceptable for an explicitly illustrative figure.
func assignIllustrativeShares(candidates []Candidate, fiiAreaBps int) {
	n := len(candidates)
	if n == 0 || fiiAreaBps <= 0 {
		return
	}
	base := fiiAreaBps / n
	remainder := fiiAreaBps - base*n
	for i := range candidates {
		candidates[i].IllustrativeShareBps = base
		if i < remainder {
			candidates[i].IllustrativeShareBps++
		}
	}
}
