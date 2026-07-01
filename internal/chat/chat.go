package chat

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// ErrUnknownRole marks an unrecognised message role (SPEC-108). Check with errors.Is.
var ErrUnknownRole = errors.New("unknown role")

// TaskChat is the insight.Task the gated Insighter reasons under for a chat turn (SPEC-005 is
// task-agnostic; this owns the chat task). The gates are unchanged.
const TaskChat insight.Task = "chat"

// Role is who authored a message (SPEC-108 §6). Closed enum.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// ParseRole normalises s (trim + lower-case) into a Role, or ErrUnknownRole via %w.
func ParseRole(s string) (Role, error) {
	switch r := Role(strings.ToLower(strings.TrimSpace(s))); r {
	case RoleUser, RoleAssistant:
		return r, nil
	default:
		return "", fmt.Errorf("parse role %q: %w", s, ErrUnknownRole)
	}
}

// Intent classifies a turn for grounding (SPEC-108 §6). Internal; parsed deterministically from the
// message, never trusted from the client.
type Intent string

const (
	IntentGeneral      Intent = "general"
	IntentContribution Intent = "contribution" // "tenho R$X pra aportar" → SPEC-105 facts
	IntentProjection   Intent = "projection"   // "daqui a N anos" / passive income → SPEC-107 facts
)

// Thread is a per-user conversation (SPEC-108 §6). Title is derived from the first user message
// (truncated) — never AI-generated text outside the Insighter.
type Thread struct {
	ID        string
	UserID    string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message is one turn in a thread (SPEC-108 §6). Assistant messages always carry an Explanation
// (the Insighter's explainability-gate guarantee, FR-013).
type Message struct {
	ID          string
	ThreadID    string
	Role        Role
	Content     string
	Explanation string // assistant only
	CreatedAt   time.Time
}

// Reply is the engine's result for a turn (SPEC-108 §6): the gated assistant message + the non-advice
// disclaimer. Available is false when the LLM was unavailable (a degraded turn).
type Reply struct {
	Message    Message
	Disclaimer string
	Available  bool
}
