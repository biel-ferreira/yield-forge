// Package postgres implements the chat repository (chat.Repository) over PostgreSQL.
//
// It is an adapter: it depends on the chat core (port + sentinels) and on database/sql, never the
// reverse — the core imports no SQL (SPEC-108, SPEC-002 BR-202). All SQL is parameterized and
// per-user scoped; reads/mutations of a specific thread are double-scoped by (id, user_id) so a
// cross-user access is ErrThreadNotFound, never an existence oracle (SPEC-108 BR-1083). Storage is
// bounded by EnforceCap (rolling eviction, FR-1086).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/chat"
)

// invalidTextRepresentation is the SQLSTATE for a malformed cast (e.g. a non-UUID threadID) — a
// read treats it as not-found, never a 500.
const invalidTextRepresentation = "22P02"

// Compile-time check that the adapter satisfies the port.
var _ chat.Repository = Repository{}

// Repository is the Postgres-backed chat.Repository.
type Repository struct {
	db *sql.DB
}

// New returns a Repository over db.
func New(db *sql.DB) Repository { return Repository{db: db} }

// CreateThread inserts a new thread and returns it with the DB-assigned id.
func (r Repository) CreateThread(ctx context.Context, t chat.Thread) (chat.Thread, error) {
	const stmt = `INSERT INTO chat_threads (user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4) RETURNING id`
	if err := r.db.QueryRowContext(ctx, stmt, t.UserID, t.Title, t.CreatedAt, t.UpdatedAt).Scan(&t.ID); err != nil {
		return chat.Thread{}, fmt.Errorf("create thread: %w", err)
	}
	return t, nil
}

// GetThreadByID returns the thread for (threadID, userID), or chat.ErrThreadNotFound.
func (r Repository) GetThreadByID(ctx context.Context, userID, threadID string) (chat.Thread, error) {
	const q = `SELECT id, user_id, title, created_at, updated_at
		FROM chat_threads WHERE id = $1 AND user_id = $2`
	var t chat.Thread
	err := r.db.QueryRowContext(ctx, q, threadID, userID).Scan(&t.ID, &t.UserID, &t.Title, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) || isInvalidText(err) {
		return chat.Thread{}, chat.ErrThreadNotFound
	}
	if err != nil {
		return chat.Thread{}, fmt.Errorf("get thread: %w", err)
	}
	return t, nil
}

// ListThreads returns the user's threads, most-recently-updated first.
func (r Repository) ListThreads(ctx context.Context, userID string) ([]chat.Thread, error) {
	const q = `SELECT id, user_id, title, created_at, updated_at
		FROM chat_threads WHERE user_id = $1 ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()

	var out []chat.Thread
	for rows.Next() {
		var t chat.Thread
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list threads: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListMessages returns a thread's messages in order — double-scoped via a join to the owning thread,
// so another user's thread yields no rows (never leaks content).
func (r Repository) ListMessages(ctx context.Context, userID, threadID string) ([]chat.Message, error) {
	const q = `SELECT m.id, m.thread_id, m.role, m.content, m.explanation, m.created_at
		FROM chat_messages m JOIN chat_threads t ON m.thread_id = t.id
		WHERE t.id = $1 AND t.user_id = $2 ORDER BY m.created_at`
	rows, err := r.db.QueryContext(ctx, q, threadID, userID)
	if err != nil {
		if isInvalidText(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var out []chat.Message
	for rows.Next() {
		var (
			m    chat.Message
			role string
			expl sql.NullString
		)
		if err := rows.Scan(&m.ID, &m.ThreadID, &role, &m.Content, &expl, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("list messages: %w", err)
		}
		m.Role = chat.Role(role)
		m.Explanation = expl.String
		out = append(out, m)
	}
	return out, rows.Err()
}

// AppendMessage inserts a message and advances the parent thread's updated_at (rolling-eviction
// order). The caller has already verified thread ownership (GetThreadByID, double-scoped).
func (r Repository) AppendMessage(ctx context.Context, m chat.Message) (chat.Message, error) {
	const stmt = `INSERT INTO chat_messages (thread_id, role, content, explanation, created_at)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var expl sql.NullString
	if m.Explanation != "" {
		expl = sql.NullString{String: m.Explanation, Valid: true}
	}
	if err := r.db.QueryRowContext(ctx, stmt, m.ThreadID, string(m.Role), m.Content, expl, m.CreatedAt).Scan(&m.ID); err != nil {
		return chat.Message{}, fmt.Errorf("append message: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE chat_threads SET updated_at = $1 WHERE id = $2`, m.CreatedAt, m.ThreadID); err != nil {
		return chat.Message{}, fmt.Errorf("append message: touch thread: %w", err)
	}
	return m, nil
}

// DeleteThread removes an owned thread (cascade deletes its messages); a non-owned id is a no-op.
func (r Repository) DeleteThread(ctx context.Context, userID, threadID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chat_threads WHERE id = $1 AND user_id = $2`, threadID, userID)
	if err != nil && !isInvalidText(err) {
		return fmt.Errorf("delete thread: %w", err)
	}
	return nil
}

// ClearThreads removes all the user's threads (cascade deletes their messages).
func (r Repository) ClearThreads(ctx context.Context, userID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM chat_threads WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear threads: %w", err)
	}
	return nil
}

// EnforceCap keeps the maxThreads most-recently-updated threads for the user, evicting older ones.
func (r Repository) EnforceCap(ctx context.Context, userID string, maxThreads int) error {
	const stmt = `DELETE FROM chat_threads
		WHERE user_id = $1 AND id NOT IN (
			SELECT id FROM chat_threads WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2
		)`
	if _, err := r.db.ExecContext(ctx, stmt, userID, maxThreads); err != nil {
		return fmt.Errorf("enforce cap: %w", err)
	}
	return nil
}

func isInvalidText(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == invalidTextRepresentation
}
