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
	factCtx, span := s.tracer.Start(ctx, "rebalancing.facts")
	base, err := s.facts.BuildFacts(factCtx, userID)
	if err == nil {
		var universe []marketdata.FIIQuote
		universe, err = s.universe.ListFIIUniverse(factCtx)
		if err == nil {
			facts, split := assembleFacts(base, contribution, universe)
			span.End()
			return s.generate(ctx, userID, facts, split, universe, opts)
		}
	}
	span.End()
	return Rebalancing{}, fmt.Errorf("rebalance: %w", err)
}

// generate runs the gated Insighter over the assembled facts (areas then candidates).
func (s *Service) generate(ctx context.Context, userID string, facts insight.Facts, split []AreaShare, universe []marketdata.FIIQuote, opts Options) (Rebalancing, error) {
	areas, succeeded, err := s.buildAreas(ctx, userID, facts, split)
	if err != nil {
		return Rebalancing{}, err
	}
	candidates, err := s.buildCandidates(ctx, userID, facts, universe, areaShareBps(split, areaClassFII), opts)
	if err != nil {
		return Rebalancing{}, err
	}

	return Rebalancing{
		Areas:      areas,
		Candidates: candidates,
		Disclaimer: insight.Disclaimer,
		Available:  succeeded > 0, // false only when every area failed (full outage)
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
		succeeded++
		area := Area{
			Class:                   string(sh.Class),
			SuggestedShareBps:       sh.SuggestedShareBps,
			SuggestedAmountCentavos: sh.SuggestedAmountCentavos,
		}
		if len(res.Insights) > 0 {
			in := res.Insights[0]
			area.Title, area.Detail, area.Explanation = in.Title, in.Detail, in.Explanation
		}
		areas = append(areas, area)
	}
	return areas, succeeded, nil
}

// buildCandidates asks the Insighter for named candidates and applies the grounding guard: only a
// ticker present in the universe survives (BR-1053). When opted in, the FII area's share is split
// evenly across the surfaced candidates as an illustrative within-area consideration (D6).
func (s *Service) buildCandidates(ctx context.Context, userID string, facts insight.Facts, universe []marketdata.FIIQuote, fiiAreaBps int, opts Options) ([]Candidate, error) {
	known := tickerIndex(universe)
	res, err := s.insighter.Generate(ctx, insight.InsightRequest{
		Facts:  withFocus(facts, "candidates", ""),
		Task:   TaskRebalancing,
		UserID: userID,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("rebalance: %w", ctx.Err())
		}
		return nil, nil // candidates are optional — a degraded candidates call is not fatal
	}

	var candidates []Candidate
	for _, in := range res.Insights {
		q, ok := known[normalizeTicker(in.Title)]
		if !ok {
			continue // grounding guard: the assistant never names a ticker the system doesn't know
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
	return candidates, nil
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
// remainder to the first), as an illustrative within-area consideration — never an order (D6).
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
