package chat

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// fakeRepo is an in-memory chat.Repository.
type fakeRepo struct {
	threads  map[string]Thread
	messages map[string][]Message
	seq      int
	capCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{threads: map[string]Thread{}, messages: map[string][]Message{}}
}

func (r *fakeRepo) CreateThread(_ context.Context, t Thread) (Thread, error) {
	r.seq++
	t.ID = "thr_" + string(rune('a'+r.seq))
	r.threads[t.ID] = t
	return t, nil
}
func (r *fakeRepo) GetThreadByID(_ context.Context, userID, threadID string) (Thread, error) {
	t, ok := r.threads[threadID]
	if !ok || t.UserID != userID {
		return Thread{}, ErrThreadNotFound
	}
	return t, nil
}
func (r *fakeRepo) ListThreads(_ context.Context, userID string) ([]Thread, error) {
	var out []Thread
	for _, t := range r.threads {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (r *fakeRepo) ListMessages(_ context.Context, userID, threadID string) ([]Message, error) {
	t, ok := r.threads[threadID]
	if !ok || t.UserID != userID {
		return nil, nil
	}
	return r.messages[threadID], nil
}
func (r *fakeRepo) AppendMessage(_ context.Context, m Message) (Message, error) {
	r.seq++
	m.ID = "msg_" + string(rune('a'+r.seq))
	r.messages[m.ThreadID] = append(r.messages[m.ThreadID], m)
	return m, nil
}
func (r *fakeRepo) DeleteThread(_ context.Context, userID, threadID string) error { return nil }
func (r *fakeRepo) ClearThreads(_ context.Context, userID string) error           { return nil }
func (r *fakeRepo) EnforceCap(_ context.Context, userID string, max int) error {
	r.capCalls++
	return nil
}

// fake fact sources.
type fakeFacts struct{ f insight.Facts }

func (s fakeFacts) BuildFacts(context.Context, string) (insight.Facts, error) {
	return s.f, nil
}

type fakeContribution struct {
	calls int
	err   error
}

func (s *fakeContribution) BuildContributionFacts(context.Context, string, int64) (insight.Facts, error) {
	s.calls++
	return insight.Facts{"kind": "contribution"}, s.err
}

type fakeProjection struct{ calls int }

func (s *fakeProjection) BuildProjectionFacts(context.Context, string, int64, int) (insight.Facts, error) {
	s.calls++
	return insight.Facts{"kind": "projection"}, nil
}

// stubInsighter records the facts it received and returns a configured result/error.
type stubInsighter struct {
	gotFacts insight.Facts
	result   insight.InsightResult
	err      error
	calls    int
}

func (s *stubInsighter) Generate(_ context.Context, req insight.InsightRequest) (insight.InsightResult, error) {
	s.calls++
	s.gotFacts = req.Facts
	return s.result, s.err
}

func okResult() insight.InsightResult {
	return insight.InsightResult{
		Insights:   []insight.Insight{{Detail: "Sim, você está concentrado.", Explanation: "porque 60%..."}},
		Disclaimer: insight.Disclaimer,
	}
}

func newEngine(repo Repository, contrib ContributionFactSource, proj ProjectionFactSource, ins insight.Insighter) *Service {
	return NewService(repo, fakeFacts{f: insight.Facts{"kind": "general"}}, contrib, proj, ins,
		fixedClock{t: time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)})
}

func TestSend_GeneralTurnPersistsAndGrounds(t *testing.T) {
	repo := newFakeRepo()
	ins := &stubInsighter{result: okResult()}
	svc := newEngine(repo, &fakeContribution{}, &fakeProjection{}, ins)

	reply, err := svc.Send(context.Background(), "u1", "", "estou concentrado demais em logística?")
	require.NoError(t, err)
	require.True(t, reply.Available)
	require.Equal(t, RoleAssistant, reply.Message.Role)
	require.NotEmpty(t, reply.Message.Content)
	require.NotEmpty(t, reply.Message.Explanation, "every assistant message carries an explanation")
	require.Equal(t, insight.Disclaimer, reply.Disclaimer)

	// A thread was created and both the user + assistant messages persisted, in order.
	require.Len(t, repo.threads, 1)
	var threadID string
	for id := range repo.threads {
		threadID = id
	}
	require.Len(t, repo.messages[threadID], 2)
	require.Equal(t, RoleUser, repo.messages[threadID][0].Role)
	require.Equal(t, RoleAssistant, repo.messages[threadID][1].Role)
	require.Equal(t, 1, repo.capCalls, "the cap is enforced")

	// Grounded with the GENERAL facts (not contribution/projection), plus the question + conversation.
	require.Equal(t, "general", ins.gotFacts["kind"])
	require.Equal(t, "estou concentrado demais em logística?", ins.gotFacts["question"])
	require.Contains(t, ins.gotFacts, "conversation")
}

func TestSend_ContributionTurnRoutesToRebalancer(t *testing.T) {
	repo := newFakeRepo()
	ins := &stubInsighter{result: okResult()}
	contrib := &fakeContribution{}
	svc := newEngine(repo, contrib, &fakeProjection{}, ins)

	_, err := svc.Send(context.Background(), "u1", "", "tenho R$2.000 pra aportar esse mês")
	require.NoError(t, err)
	require.Equal(t, 1, contrib.calls, "contribution intent grounds via SPEC-105")
	require.Equal(t, "contribution", ins.gotFacts["kind"])
}

func TestSend_ProjectionTurnRoutesToProjections(t *testing.T) {
	repo := newFakeRepo()
	ins := &stubInsighter{result: okResult()}
	proj := &fakeProjection{}
	svc := newEngine(repo, &fakeContribution{}, proj, ins)

	_, err := svc.Send(context.Background(), "u1", "", "como fica meu patrimônio daqui a 10 anos?")
	require.NoError(t, err)
	require.Equal(t, 1, proj.calls, "projection intent grounds via SPEC-107")
	require.Equal(t, "projection", ins.gotFacts["kind"])
}

func TestSend_ContributionSourceErrorDegradesToGeneral(t *testing.T) {
	repo := newFakeRepo()
	ins := &stubInsighter{result: okResult()}
	svc := newEngine(repo, &fakeContribution{err: context.DeadlineExceeded}, &fakeProjection{}, ins)

	_, err := svc.Send(context.Background(), "u1", "", "tenho R$2.000 pra aportar")
	require.NoError(t, err)
	require.Equal(t, "general", ins.gotFacts["kind"], "a rebalancer error falls back to general facts")
}

func TestSend_LLMOutageDegradesNotPersisted(t *testing.T) {
	repo := newFakeRepo()
	ins := &stubInsighter{err: insight.ErrInsightsUnavailable}
	svc := newEngine(repo, &fakeContribution{}, &fakeProjection{}, ins)

	reply, err := svc.Send(context.Background(), "u1", "", "olá")
	require.NoError(t, err, "an outage is degradation, not a hard error")
	require.False(t, reply.Available)
	require.NotEmpty(t, reply.Message.Content)

	// The user message persisted; no assistant message was persisted (thread stays clean, retry-able).
	var threadID string
	for id := range repo.threads {
		threadID = id
	}
	require.Len(t, repo.messages[threadID], 1)
	require.Equal(t, RoleUser, repo.messages[threadID][0].Role)
}

func TestSend_GateRejectReturnsSafeReply(t *testing.T) {
	repo := newFakeRepo()
	ins := &stubInsighter{err: insight.ErrAdviceDetected}
	svc := newEngine(repo, &fakeContribution{}, &fakeProjection{}, ins)

	reply, err := svc.Send(context.Background(), "u1", "", "devo comprar XPML11 agora?")
	require.NoError(t, err)
	require.True(t, reply.Available, "the copilot is available — it just declines to give an order")
	require.Contains(t, reply.Message.Content, "considerações")
}

func TestSend_UnownedThread404(t *testing.T) {
	repo := newFakeRepo()
	svc := newEngine(repo, &fakeContribution{}, &fakeProjection{}, &stubInsighter{result: okResult()})
	_, err := svc.Send(context.Background(), "u1", "thr_nope", "oi")
	require.ErrorIs(t, err, ErrThreadNotFound)
}
