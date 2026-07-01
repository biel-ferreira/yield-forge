package chat

import (
	"context"
	"fmt"
)

// ListThreads returns the caller's conversation threads, most-recently-updated first (FR-1086).
func (s *Service) ListThreads(ctx context.Context, userID string) ([]Thread, error) {
	return s.repo.ListThreads(ctx, userID)
}

// Thread returns an owned thread and its ordered messages, or ErrThreadNotFound (double-scoped).
func (s *Service) Thread(ctx context.Context, userID, threadID string) (Thread, []Message, error) {
	t, err := s.repo.GetThreadByID(ctx, userID, threadID)
	if err != nil {
		return Thread{}, nil, err // ErrThreadNotFound
	}
	msgs, err := s.repo.ListMessages(ctx, userID, threadID)
	if err != nil {
		return Thread{}, nil, fmt.Errorf("thread messages: %w", err)
	}
	return t, msgs, nil
}

// DeleteThread removes an owned thread and its messages (FR-1086); a non-owned id is a no-op.
func (s *Service) DeleteThread(ctx context.Context, userID, threadID string) error {
	return s.repo.DeleteThread(ctx, userID, threadID)
}

// ClearThreads removes all the caller's conversation history (FR-1086 / FR-025).
func (s *Service) ClearThreads(ctx context.Context, userID string) error {
	return s.repo.ClearThreads(ctx, userID)
}
