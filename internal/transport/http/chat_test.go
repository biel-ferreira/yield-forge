package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/chat"
	"github.com/biel-ferreira/yield-forge/internal/insight"
)

type fakeChatService struct {
	gotUserID  string
	gotThread  string
	gotContent string
	reply      chat.Reply
	threads    []chat.Thread
	messages   []chat.Message
	err        error
	cleared    bool
	deleted    string
}

func (f *fakeChatService) Send(_ context.Context, userID, threadID, content string) (chat.Reply, error) {
	f.gotUserID, f.gotThread, f.gotContent = userID, threadID, content
	return f.reply, f.err
}
func (f *fakeChatService) ListThreads(_ context.Context, userID string) ([]chat.Thread, error) {
	f.gotUserID = userID
	return f.threads, f.err
}
func (f *fakeChatService) Thread(_ context.Context, userID, threadID string) (chat.Thread, []chat.Message, error) {
	f.gotUserID, f.gotThread = userID, threadID
	if f.err != nil {
		return chat.Thread{}, nil, f.err
	}
	return chat.Thread{ID: threadID, Title: "t"}, f.messages, nil
}
func (f *fakeChatService) DeleteThread(_ context.Context, userID, threadID string) error {
	f.deleted = threadID
	return f.err
}
func (f *fakeChatService) ClearThreads(_ context.Context, userID string) error {
	f.cleared = true
	return f.err
}

func newChatHandler(svc ChatService) chatHandler {
	return chatHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleReply() chat.Reply {
	return chat.Reply{
		Message:    chat.Message{ID: "msg_1", ThreadID: "thr_1", Role: chat.RoleAssistant, Content: "Sim...", Explanation: "porque...", CreatedAt: time.Now()},
		Disclaimer: insight.Disclaimer,
		Available:  true,
	}
}

func TestPostChatMessage_IdentityAndShape(t *testing.T) {
	svc := &fakeChatService{reply: sampleReply()}
	h := newChatHandler(svc)

	rec := httptest.NewRecorder()
	h.postMessage(rec, authed(http.MethodPost, "/chat/messages", `{"content":"estou concentrado?"}`, "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "identity from context")
	require.Equal(t, "estou concentrado?", svc.gotContent)

	var resp chatReplyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Available)
	require.Equal(t, "assistant", resp.Message.Role)
	require.NotEmpty(t, resp.Message.Explanation)
	require.NotEmpty(t, resp.Disclaimer)
}

func TestPostChatMessage_RejectsEmptyOrTooLong(t *testing.T) {
	long := make([]byte, 2001)
	for i := range long {
		long[i] = 'a'
	}
	for _, body := range []string{`{"content":"   "}`, `{"content":"` + string(long) + `"}`, `{}`} {
		svc := &fakeChatService{reply: sampleReply()}
		h := newChatHandler(svc)
		rec := httptest.NewRecorder()
		h.postMessage(rec, authed(http.MethodPost, "/chat/messages", body, "u1"))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Empty(t, svc.gotUserID, "no service call on a bad request")
	}
}

func TestPostChatMessage_UnownedThread404(t *testing.T) {
	svc := &fakeChatService{err: chat.ErrThreadNotFound}
	h := newChatHandler(svc)
	rec := httptest.NewRecorder()
	h.postMessage(rec, authed(http.MethodPost, "/chat/messages", `{"thread_id":"x","content":"oi"}`, "u1"))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetChatThread_404(t *testing.T) {
	svc := &fakeChatService{err: chat.ErrThreadNotFound}
	h := newChatHandler(svc)
	req := authed(http.MethodGet, "/chat/threads/x", "", "u1")
	req.SetPathValue("id", "x")
	rec := httptest.NewRecorder()
	h.getThread(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestClearAndDeleteThreads_NoContent(t *testing.T) {
	svc := &fakeChatService{}
	h := newChatHandler(svc)

	rec := httptest.NewRecorder()
	h.clearThreads(rec, authed(http.MethodDelete, "/chat/threads", "", "u1"))
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.True(t, svc.cleared)

	req := authed(http.MethodDelete, "/chat/threads/thr_9", "", "u1")
	req.SetPathValue("id", "thr_9")
	rec = httptest.NewRecorder()
	h.deleteThread(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "thr_9", svc.deleted)
}

func TestChatEndpoints_Unauthenticated(t *testing.T) {
	h := newChatHandler(&fakeChatService{})
	rec := httptest.NewRecorder()
	h.listThreads(rec, httptest.NewRequest(http.MethodGet, "/chat/threads", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
