package jev

import (
	"errors"
	"time"
)

const (
	// Endpoint is the System One evaluation URL.
	// https://docs.typesafe.ai/api
	Endpoint = "https://api.typesafe.ai/v1/systemone"

	// TotalSlots is the process-wide in-flight cap. A free slot is capacity,
	// not a promise that the call meets the intake latency gate.
	TotalSlots = 24

	// InteractiveReserve is the number of TotalSlots background work cannot
	// take. Eight leaves sixteen slots for background labeling and keeps a
	// short filing burst from waiting behind it. Revisit the split when live
	// p50/p90 are measured.
	InteractiveReserve = 8

	// InteractiveDeadline is the one budget for queue wait and the HTTP
	// attempt on a foreground call. Background attempts use the same per-try
	// cap so a stuck call cannot hold a slot longer than a foreground one.
	InteractiveDeadline = 1200 * time.Millisecond

	// MaxResponseBytes bounds a single response body.
	MaxResponseBytes = 1 << 20

	// MaxChoiceOptions is the TypeSafe choice limit.
	// https://docs.typesafe.ai/api
	MaxChoiceOptions = 255

	TypeNoul   = "noul"
	TypeChoice = "choice"
	TypeScore  = "score"

	breakerThreshold   = 5
	breakerCooldown    = 30 * time.Second
	backgroundAttempts = 3
	backgroundBackoff  = 200 * time.Millisecond
)

var (
	ErrCircuitOpen  = errors.New("jev circuit open")
	ErrRateLimited  = errors.New("jev rate limited")
	ErrOverloaded   = errors.New("jev overloaded")
	ErrBadResponse  = errors.New("jev response rejected")
	ErrBodyLimit    = errors.New("jev response exceeds size limit")
	ErrRequest      = errors.New("jev request rejected")
	ErrUnauthorized = errors.New("jev unauthorized")
)

// Priority chooses admission and retry policy.
type Priority string

const (
	// PriorityInteractive is one attempt. The deadline covers admission and
	// transport, and a failure is not retried on the hot path.
	PriorityInteractive Priority = "interactive"
	// PriorityBackground may retry rate limits, overload and transport
	// errors with backoff. It cannot take InteractiveReserve slots.
	PriorityBackground Priority = "background"
)

// Question is one typed System One question.
// Instructions and Criteria follow https://docs.typesafe.ai/api.
type Question struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

// Call is one evaluation. State is encoded as JSON and must already be the
// relevant evidence: Jev is not asked to count, compare dates or draft text.
// https://docs.typesafe.ai/model-jaggedness/jev-1.13
type Call struct {
	State           any
	Questions       map[string]Question
	Priority        Priority
	QuestionVersion string
	// Deadline overrides the client deadline for this call when set.
	// Zero uses InteractiveDeadline, which covers admission and transport.
	Deadline time.Duration
}

// Answer is one validated answer. Missing or malformed answers are omitted
// rather than filled in. Noul carries probability only; choice and score
// carry Confidence when the response included a finite value.
// https://docs.typesafe.ai/confidence
type Answer struct {
	Type          string
	Noul          float64
	Choice        string
	Score         float64
	Probabilities map[string]float64
	Confidence    *float64
}

// Usage is token accounting from the response.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Result is the validated subset of one response.
type Result struct {
	Model           string
	Answers         map[string]Answer
	Unresolved      []string
	Usage           Usage
	Latency         time.Duration
	QuestionVersion string
}
