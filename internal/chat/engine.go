package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
)

// Engine bounds (SPEC-108 D4/§9 — zero-cost + cost-safety). Tunable; documented.
const (
	maxThreads          = 50 // per-user rolling cap (FR-1086)
	priorWindow         = 10 // bounded prior-turn context sent to the LLM
	maxTitleLen         = 60 // thread title = first message, truncated
	defaultHorizonYears = 10 // projection turn without an explicit horizon
)

// Service is the chat engine (SPEC-108). Per turn it resolves the thread, persists the user message,
// classifies the intent, grounds it with the matching engine's DETERMINISTIC facts, calls the gated
// Insighter ONCE, and persists + returns the gated reply — so explainability (FR-013) and non-advice
// (FR-014) hold by construction (BR-1082). Identity comes from the caller's context.
type Service struct {
	repo         Repository
	facts        FactSource
	contribution ContributionFactSource
	projection   ProjectionFactSource
	insighter    insight.Insighter
	clock        clock.Clock
	tracer       trace.Tracer
}

// NewService builds the chat engine over the repository, the grounding fact sources (SPEC-104/105/107),
// the gated Insighter (SPEC-005), and the Clock.
func NewService(repo Repository, facts FactSource, contribution ContributionFactSource, projection ProjectionFactSource, insighter insight.Insighter, clk clock.Clock) *Service {
	return &Service{repo: repo, facts: facts, contribution: contribution, projection: projection, insighter: insighter, clock: clk, tracer: observability.Tracer("chat")}
}

// Send handles one turn: resolve/create the thread, persist the user message, ground the turn, call
// the gated Insighter, and persist + return the gated assistant reply. A degraded (LLM outage) or
// gate-rejected turn returns a safe reply and is NOT persisted (the thread stays readable, the user
// can retry). An unowned/unknown thread → ErrThreadNotFound (→ 404).
func (s *Service) Send(ctx context.Context, userID, threadID, content string) (Reply, error) {
	// Span over the turn for latency visibility. It carries NO content — no message text, facts, or
	// generated reply reach telemetry (BR-505/FR-1089). The Insighter records its own generate span.
	ctx, span := s.tracer.Start(ctx, "chat.turn")
	defer span.End()

	now := s.clock.Now()

	thread, err := s.resolveThread(ctx, userID, threadID, content, now)
	if err != nil {
		return Reply{}, err // ErrThreadNotFound propagates
	}
	if _, err := s.repo.AppendMessage(ctx, Message{ThreadID: thread.ID, Role: RoleUser, Content: content, CreatedAt: now}); err != nil {
		return Reply{}, fmt.Errorf("chat send: %w", err)
	}

	intent, amount, horizon := Classify(content)
	grounding, err := s.ground(ctx, userID, intent, amount, horizon)
	if err != nil {
		return Reply{}, fmt.Errorf("chat send: %w", err)
	}
	prior, err := s.priorContext(ctx, userID, thread.ID)
	if err != nil {
		return Reply{}, fmt.Errorf("chat send: %w", err)
	}

	res, err := s.insighter.Generate(ctx, insight.InsightRequest{
		Facts:  turnFacts(grounding, prior, content),
		Task:   TaskChat,
		UserID: userID,
	})
	if err != nil {
		if ctx.Err() != nil {
			return Reply{}, fmt.Errorf("chat send: %w", ctx.Err())
		}
		return degradedReply(err), nil // not persisted — the thread stays readable, user can retry
	}
	if !explained(res.Insights) {
		return degradedReply(insight.ErrInsightsUnavailable), nil
	}

	assistant, err := s.repo.AppendMessage(ctx, Message{
		ThreadID:    thread.ID,
		Role:        RoleAssistant,
		Content:     replyContent(res.Insights[0]),
		Explanation: strings.TrimSpace(res.Insights[0].Explanation),
		CreatedAt:   s.clock.Now(),
	})
	if err != nil {
		return Reply{}, fmt.Errorf("chat send: %w", err)
	}

	return Reply{Message: assistant, Disclaimer: firstNonEmpty(res.Disclaimer, insight.Disclaimer), Available: true}, nil
}

// resolveThread returns the owned thread for threadID, or creates a fresh one for the caller and
// enforces the rolling cap immediately — so a new-thread turn stays bounded even if it later
// degrades (LLM outage) and never reaches the reply (FR-1086 / BR-1085).
func (s *Service) resolveThread(ctx context.Context, userID, threadID, content string, now time.Time) (Thread, error) {
	if threadID != "" {
		return s.repo.GetThreadByID(ctx, userID, threadID)
	}
	thread, err := s.repo.CreateThread(ctx, Thread{UserID: userID, Title: title(content), CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return Thread{}, err
	}
	if err := s.repo.EnforceCap(ctx, userID, maxThreads); err != nil {
		return Thread{}, err
	}
	return thread, nil
}

// ground routes the turn to the matching engine's deterministic facts, degrading to the general
// SPEC-104 facts when the intent's source is absent or errors at runtime (resilience, FR-1083). No
// grounding path invokes the LLM (D5) — the single LLM call is the Insighter.Generate above.
func (s *Service) ground(ctx context.Context, userID string, intent Intent, amountCentavos int64, horizonYears int) (insight.Facts, error) {
	switch intent {
	case IntentContribution:
		if s.contribution != nil {
			if f, err := s.contribution.BuildContributionFacts(ctx, userID, amountCentavos); err == nil {
				return f, nil
			}
		}
	case IntentProjection:
		if s.projection != nil {
			h := horizonYears
			if h == 0 {
				h = defaultHorizonYears
			}
			if f, err := s.projection.BuildProjectionFacts(ctx, userID, amountCentavos, h); err == nil {
				return f, nil
			}
		}
	}
	return s.facts.BuildFacts(ctx, userID) // general — and the degradation fallback
}

// priorContext returns the last priorWindow messages of the thread (bounded conversational context).
func (s *Service) priorContext(ctx context.Context, userID, threadID string) ([]Message, error) {
	msgs, err := s.repo.ListMessages(ctx, userID, threadID)
	if err != nil {
		return nil, err
	}
	if len(msgs) > priorWindow {
		msgs = msgs[len(msgs)-priorWindow:]
	}
	return msgs, nil
}

// turnFacts merges the deterministic grounding facts with the bounded conversation + the new
// question. The conversation is DIALOGUE context only — prior assistant text is never a source of
// figures; the numbers come solely from the grounding facts (BR-1081).
func turnFacts(grounding insight.Facts, prior []Message, question string) insight.Facts {
	out := make(insight.Facts, len(grounding)+2)
	for k, v := range grounding {
		out[k] = v
	}
	conv := make([]map[string]string, 0, len(prior))
	for _, m := range prior {
		conv = append(conv, map[string]string{"role": string(m.Role), "content": m.Content})
	}
	out["conversation"] = conv
	out["question"] = question
	return out
}

// degradedReply is a safe, non-persisted product reply (not AI text) for an outage or a gate rejection.
func degradedReply(err error) Reply {
	if errors.Is(err, insight.ErrAdviceDetected) || errors.Is(err, insight.ErrMissingExplanation) {
		return Reply{
			Message:    Message{Role: RoleAssistant, Content: "Posso trazer considerações para a sua análise, mas não ordens de compra ou venda."},
			Disclaimer: insight.Disclaimer,
			Available:  true,
		}
	}
	return Reply{
		Message:   Message{Role: RoleAssistant, Content: "O copiloto está temporariamente indisponível. Tente novamente em instantes."},
		Available: false,
	}
}

// explained reports whether the gated result carries a usable explanation (FR-1084) — a
// successful-but-empty result must not surface as an unexplained assistant turn.
func explained(insights []insight.Insight) bool {
	return len(insights) > 0 && strings.TrimSpace(insights[0].Explanation) != ""
}

func replyContent(in insight.Insight) string {
	return firstNonEmpty(strings.TrimSpace(in.Detail), strings.TrimSpace(in.Explanation))
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func title(content string) string {
	t := strings.TrimSpace(content)
	if r := []rune(t); len(r) > maxTitleLen {
		return strings.TrimSpace(string(r[:maxTitleLen]))
	}
	return t
}
