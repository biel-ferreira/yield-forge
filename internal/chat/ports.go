package chat

import (
	"context"
	"errors"
)

// ErrThreadNotFound is returned when a thread is absent or not owned by the caller (SPEC-108) — the
// same sentinel for both, so a cross-user probe is never an existence oracle (double-scoped reads).
var ErrThreadNotFound = errors.New("thread not found")

// Repository persists threads + messages, per-user scoped, bounded and clearable (SPEC-108 §7).
// Implemented by the chat/postgres adapter; the core depends only on this port, not on SQL.
type Repository interface {
	CreateThread(ctx context.Context, t Thread) (Thread, error)
	// GetThreadByID returns the thread, or ErrThreadNotFound when absent/not owned (double-scoped).
	GetThreadByID(ctx context.Context, userID, threadID string) (Thread, error)
	ListThreads(ctx context.Context, userID string) ([]Thread, error)
	ListMessages(ctx context.Context, userID, threadID string) ([]Message, error)
	AppendMessage(ctx context.Context, m Message) (Message, error)
	DeleteThread(ctx context.Context, userID, threadID string) error
	ClearThreads(ctx context.Context, userID string) error
	// EnforceCap keeps the maxThreads most-recently-updated threads for the user, evicting older
	// ones (rolling window, FR-1086).
	EnforceCap(ctx context.Context, userID string, maxThreads int) error
}
