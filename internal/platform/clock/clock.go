// Package clock provides an injectable time source so business logic and tests stay
// deterministic. The Clock port replaces direct time.Now() calls in domain/service
// code (CLAUDE.md: "Time is UTC and comes from the injected Clock port"). SPEC-001
// deferred this until the first consumer of time; SPEC-003 (session expiry) is it.
package clock

import "time"

// Clock is the time source port. Now returns the current instant in UTC.
//
// Consumers accept a Clock; production wires System, and tests pass a hand-written
// fake that returns a fixed time.
type Clock interface {
	Now() time.Time
}

// System is the production Clock, backed by the wall clock in UTC.
type System struct{}

// Now returns the current UTC time.
func (System) Now() time.Time { return time.Now().UTC() }
