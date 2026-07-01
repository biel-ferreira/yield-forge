package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/chat"
)

// maxChatContentLen bounds a chat message at the edge (cost-safety, SPEC-108 §9).
const maxChatContentLen = 2000

// ChatService is the slice of the chat engine the transport needs (consumer-defined); *chat.Service
// satisfies it (SPEC-108).
type ChatService interface {
	Send(ctx context.Context, userID, threadID, content string) (chat.Reply, error)
	ListThreads(ctx context.Context, userID string) ([]chat.Thread, error)
	Thread(ctx context.Context, userID, threadID string) (chat.Thread, []chat.Message, error)
	DeleteThread(ctx context.Context, userID, threadID string) error
	ClearThreads(ctx context.Context, userID string) error
}

type chatHandler struct {
	service ChatService
	logger  *slog.Logger
}

type chatMessageRequest struct {
	ThreadID string `json:"thread_id"` // optional — omit/empty starts a new thread
	Content  string `json:"content"`
}

type chatReplyResponse struct {
	ThreadID   string              `json:"thread_id"`
	Message    chatMessageResponse `json:"message"`
	Disclaimer string              `json:"disclaimer"`
	Available  bool                `json:"available"` // false => the copilot was temporarily unavailable
}

type chatMessageResponse struct {
	ID          string    `json:"id"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Explanation string    `json:"explanation,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type chatThreadResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type chatThreadDetailResponse struct {
	Thread   chatThreadResponse    `json:"thread"`
	Messages []chatMessageResponse `json:"messages"`
}

// postMessage sends a turn, creating or continuing a thread (SPEC-108 FR-1088). Identity from the
// context (BR-1083); the content is the only free-text input, length-bounded; an unowned thread → 404.
func (h chatHandler) postMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req chatMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" || len([]rune(content)) > maxChatContentLen {
		writeError(w, http.StatusBadRequest, "content must be non-empty and at most 2000 characters")
		return
	}

	reply, err := h.service.Send(r.Context(), userID, req.ThreadID, content)
	if errors.Is(err, chat.ErrThreadNotFound) {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "chat send failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toChatReplyResponse(reply))
}

// listThreads returns the caller's threads (metadata only).
func (h chatHandler) listThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	threads, err := h.service.ListThreads(r.Context(), userID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list threads failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := make([]chatThreadResponse, len(threads))
	for i, t := range threads {
		out[i] = toChatThreadResponse(t)
	}
	writeJSON(w, http.StatusOK, out)
}

// getThread returns an owned thread and its messages, or 404.
func (h chatHandler) getThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	thread, msgs, err := h.service.Thread(r.Context(), userID, r.PathValue("id"))
	if errors.Is(err, chat.ErrThreadNotFound) {
		writeError(w, http.StatusNotFound, "thread not found")
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get thread failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := chatThreadDetailResponse{Thread: toChatThreadResponse(thread), Messages: make([]chatMessageResponse, len(msgs))}
	for i, m := range msgs {
		out.Messages[i] = toChatMessageResponse(m)
	}
	writeJSON(w, http.StatusOK, out)
}

// deleteThread removes an owned thread (204); a non-owned id is a no-op 204 (no existence oracle).
func (h chatHandler) deleteThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteThread(r.Context(), userID, r.PathValue("id")); err != nil {
		h.logger.ErrorContext(r.Context(), "delete thread failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// clearThreads removes all the caller's conversation history (204).
func (h chatHandler) clearThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	if err := h.service.ClearThreads(r.Context(), userID); err != nil {
		h.logger.ErrorContext(r.Context(), "clear threads failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toChatReplyResponse(r chat.Reply) chatReplyResponse {
	return chatReplyResponse{
		ThreadID: r.Message.ThreadID, Message: toChatMessageResponse(r.Message),
		Disclaimer: r.Disclaimer, Available: r.Available,
	}
}

func toChatMessageResponse(m chat.Message) chatMessageResponse {
	return chatMessageResponse{
		ID: m.ID, Role: string(m.Role), Content: m.Content, Explanation: m.Explanation, CreatedAt: m.CreatedAt,
	}
}

func toChatThreadResponse(t chat.Thread) chatThreadResponse {
	return chatThreadResponse{ID: t.ID, Title: t.Title, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}
