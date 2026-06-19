# Implementation Plan

## 1. Document Information

| Field           | Value                                                        |
| --------------- | ------------------------------------------------------------ |
| Plan Name       | Authentication & Per-User Isolation                          |
| Related Feature | Foundational — identity + per-user data boundary             |
| Related Spec    | [SPEC-003](../02-specs/SPEC-003-authentication-and-per-user-isolation.md) |
| Version         | 0.1.0                                                        |
| Status          | Done                                                         |
| Author          | Gabigol                                                      |
| Last Updated    | 2026-06-18                                                   |

---

## 2. Objective

### Goal

Add email+password authentication with **server-side sessions** and a **deny-by-default**
auth middleware that injects an authenticated `UserID` into the request context — the
seam every feature uses to scope data per user. Built on SPEC-002's database and
migration tooling, with **no feature/business data** of its own.

### Expected Outcome

`POST /auth/register` then `POST /auth/login` issues a session cookie; protected routes
require it (`401` otherwise); `GET /auth/me` returns the caller's identity; `POST
/auth/logout` deletes the session server-side. Passwords are stored only as bcrypt
hashes; only the `sha256` of each session token is stored. Feature specs can now call
`auth.UserID(ctx)` and scope every query by it.

---

## 3. Scope

### Included

- `internal/auth` feature package: `User`/`Session` domain, ports (`UserRepository`,
  `SessionRepository`, `PasswordHasher`), the auth `Service`, session-token generation,
  and the `UserID` context helper.
- bcrypt `PasswordHasher` adapter (`internal/auth/bcrypt`).
- Postgres adapters for the repositories (`internal/auth/postgres`).
- Migration `0002_auth` (users + sessions) with a tested `down`.
- Transport: `register`/`login`/`logout`/`me` handlers, the auth middleware
  (deny-by-default + public allowlist), cookie handling.
- Config: session TTL + cookie settings (env-driven); `.env.example` updated.
- Unit tests (no DB) + env-gated integration tests; CHANGELOG/README/lesson.

### Excluded (later specs / future work — SPEC-003 §2, §15)

- OAuth/social login, magic links; email verification, password reset (need email).
- MFA/TOTP, rate limiting / account lockout, password-strength meters.
- Roles/permissions (RBAC); Postgres RLS (future defense-in-depth on top of app scoping).
- Any feature table or business endpoint, and any login UI (future Next.js).
- Periodic expired-session cleanup job (lazy ignore now; sweep later).

---

## 4. Dependencies

### Technical Dependencies

- SPEC-002 database + migration tooling (pgx pool, `golang-migrate`, `cmd/migrate`).
- `golang.org/x/crypto/bcrypt` — promoted from **indirect to direct** (already in the
  module graph via existing deps; no new download expected). Stdlib `crypto/rand`,
  `crypto/sha256`, `crypto/subtle` for tokens.

### Blocking Decisions (resolved — SPEC-003 §14)

- **D1 — email + password, server-side sessions.**
- **D2 — bcrypt** (behind the `PasswordHasher` port; argon2id swappable later).
- **D3 — `HttpOnly` + `Secure` + `SameSite` cookie** (`Secure` off only in dev).
- **D4 — app-level `user_id` scoping** (RLS deferred).
- **Logout = hard `DELETE`** of the session row (no soft-delete; expiry covers the rest).

### Conventions adopted

- **IDs are `string`** (UUID text form). The database generates them
  (`gen_random_uuid()` / `RETURNING id`), so Go needs no UUID library.
- Email normalized to **trimmed + lowercase**; uniqueness enforced by the DB.
- Session token: 32 random bytes (`crypto/rand`), base64url-encoded → raw token to the
  client; `sha256(token)` stored (BR-303).

---

## 5. Architecture Impact

### Existing Components Affected

| Component | Impact |
| --------- | ------ |
| `internal/platform/config` | Add `SessionTTL`, cookie name/secure settings. |
| `internal/transport/http` (router/middleware) | Add auth routes; wrap protected routes with auth middleware; public allowlist. |
| `cmd/api/main.go` | Build hasher + repos + auth service; inject into the router. |
| `migrations/` | New `0002_auth.up.sql` / `.down.sql`. |
| `.env.example` / `CHANGELOG.md` / `README.md` / specs+plans indexes | Updated. |

### New Components

| Component | Purpose |
| --------- | ------- |
| `internal/auth/auth.go` | Domain: `User`, `Session`, errors, validation. |
| `internal/auth/ports.go` | `UserRepository`, `SessionRepository`, `PasswordHasher` interfaces. |
| `internal/auth/service.go` | Use cases: `Register`, `Login`, `Logout`, `Authenticate`. |
| `internal/auth/token.go` | Session-token generation + `sha256` hashing. |
| `internal/auth/context.go` | Typed context key + `WithUserID` / `UserID(ctx)` accessor. |
| `internal/auth/bcrypt/` | bcrypt `PasswordHasher` adapter. |
| `internal/auth/postgres/` | Postgres `UserRepository` + `SessionRepository`. |
| `internal/transport/http/auth.go` | Auth handlers (register/login/logout/me) + DTOs. |
| `internal/transport/http/authmw.go` | Auth middleware (deny-by-default + allowlist). |

---

## 6. Implementation Strategy

### Approach

Bottom-up, each phase compiling and testable: domain/ports/config → security
primitives → persistence → application service → transport → tests → docs. The `auth`
core imports no SQL/HTTP/vendor-SDK types (BR-306); bcrypt and SQL live in adapter
subpackages; the HTTP middleware lives in `transport/http` and calls the auth service.

### Rollout Method

**Incremental**, one PR for SPEC-003, reviewed phase-by-phase (same cadence as
SPEC-001/002: a phase, you review, continue).

### Rollback Strategy

Greenfield, not deployed. Rollback = revert the PR. The only stateful artifact is the
dev database; `0002_auth` ships a tested `down`, and the dev DB is a disposable Docker
volume. No production data.

---

## 7. Implementation Phases

### Phase 1 — Domain, Ports & Config

#### Tasks
- [ ] `internal/auth`: `User`, `Session` types; sentinel errors (`ErrEmailTaken`,
      `ErrInvalidCredentials`, `ErrSessionExpired`, …); email/password validation
      (normalize email; min password length).
- [ ] Declare the ports in `ports.go` (`UserRepository`, `SessionRepository`,
      `PasswordHasher`) — interfaces only, defined by the consumer.
- [ ] `context.go`: typed key + `WithUserID(ctx, id)` and `UserID(ctx) (string, bool)`.
- [ ] Extend `Config`: `SessionTTL` (default e.g. `168h`), `AuthCookieName`
      (default `yf_session`); cookie `Secure` derived from `!IsDev()`. Update
      `.env.example`.

#### Deliverables
- `auth` package compiles with pure domain + ports; config carries session settings;
  unit tests for validation + the context accessor.

---

### Phase 2 — Password Hashing & Session Tokens

#### Tasks
- [ ] `internal/auth/bcrypt`: `Hasher` implementing `PasswordHasher` (bcrypt cost as a
      documented constant, e.g. 12); `Hash`/`Compare`.
- [ ] `internal/auth/token.go`: `newToken()` → 32 random bytes (`crypto/rand`),
      base64url; `hashToken(raw)` → `sha256` hex; never log the raw token.

#### Deliverables
- Hash→compare round-trips; wrong password fails; token generation is high-entropy and
  only its hash is persisted. Unit-tested (no DB).

---

### Phase 3 — Persistence (migration + adapters)

#### Tasks
- [ ] `migrations/0002_auth.up.sql`:
      `users(id uuid pk default gen_random_uuid(), email text not null, password_hash
      text not null, created_at/updated_at timestamptz not null default now())` with a
      unique index on `email`;
      `sessions(id uuid pk default gen_random_uuid(), user_id uuid not null references
      users(id) on delete cascade, token_hash text not null unique, expires_at
      timestamptz not null, created_at timestamptz not null default now())` with an
      index on `user_id`.
- [ ] `0002_auth.down.sql`: drop `sessions` then `users`.
- [ ] `internal/auth/postgres`: `UserRepository` (create `RETURNING id`, find by email,
      find by id) and `SessionRepository` (create, find valid by token hash
      `expires_at > now()`, delete by token hash). Parameterized SQL only.

#### Deliverables
- `task migrate:up` applies `0002` (schema version 2); up→down→up round-trip green;
  adapters compile against the ports.

---

### Phase 4 — Application Service

#### Tasks
- [ ] `Register(ctx, email, password)`: validate, normalize, hash, insert; map a unique
      violation to `ErrEmailTaken`.
- [ ] `Login(ctx, email, password)`: find user; `Compare`; on success create a session
      (token + hash + expiry) and return the **raw** token + user; on any failure return
      the same `ErrInvalidCredentials` (generic, BR-305).
- [ ] `Logout(ctx, rawToken)`: `DELETE` the session by token hash.
- [ ] `Authenticate(ctx, rawToken) (User, error)`: resolve a valid, unexpired session →
      its user (used by the middleware).

#### Deliverables
- Service orchestrates the ports; unit-tested with **fake** repos + a fake hasher
  (register new/duplicate, login good/bad, logout, authenticate valid/expired).

---

### Phase 5 — Transport (handlers + middleware + isolation seam)

#### Tasks
- [ ] `transport/http/auth.go`: `register`/`login`/`logout`/`me` handlers + DTOs;
      generic error envelope; never echo the password. `login` sets the session cookie
      (`HttpOnly`, `Secure` per env, `SameSite`); `logout` clears it.
- [ ] `transport/http/authmw.go`: middleware that reads the cookie, calls
      `auth.Service.Authenticate`, injects `UserID` into context, adds `user_id` to the
      request log; deny-by-default with a public allowlist (`/healthz`, `/readyz`,
      `/version`, `/auth/register`, `/auth/login`).
- [ ] Wire routes + middleware in `NewRouter`; build hasher/repos/service in `main.go`.

#### Deliverables
- End-to-end: register → login (cookie) → `/auth/me` 200 → logout → `/auth/me` 401;
  protected routes 401 without a session; handler/middleware unit-tested with fakes.

---

### Phase 6 — Testing

#### Unit Tests (no DB)
- [ ] bcrypt hasher round-trip; token hashing.
- [ ] Service with fakes: register (new/duplicate), login (good/bad → identical error),
      logout, authenticate (valid/expired/missing).
- [ ] Middleware: valid session injects `UserID`; missing/invalid/expired → `401`;
      public routes bypass. `auth.UserID(ctx)` present/absent.

#### Integration Tests (real Postgres, gated by `testing.Short()` + `TEST_DATABASE_URL`)
- [ ] Full flow over HTTP (`httptest`): register → login → me → logout → me `401`.
- [ ] Persistence asserts **no plaintext**: the stored `password_hash` ≠ the password and
      `token_hash` ≠ the raw token.
- [ ] Isolation seam (FR-306): two users; a request authenticated as A resolves to A's
      id, not B's.

#### Deliverables
- `go test ./...` green with and without a DB; `go vet`/`gofmt` clean; `auth` core imports
  no SQL/HTTP types (BR-306).

---

### Phase 7 — Documentation & Lesson

#### Tasks
- [ ] `CHANGELOG.md` `[Unreleased]`: auth package, `0002_auth`, endpoints, middleware,
      session/cookie handling, new env vars.
- [ ] `README.md`: auth endpoints, protected-by-default posture, new env vars, the
      register→login→me flow.
- [ ] Flip SPEC-003 + PLAN-003 to Done in both indexes.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-003-aula.html`.

#### Deliverables
- Docs current; SPEC-003 closed; lesson published.

---

## 8. Risks

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| Storing anything in plaintext (password/token) | High | bcrypt for passwords, `sha256` for tokens; integration test asserts no plaintext in the row; redact in logs. |
| Account enumeration via error/timing differences | Medium | Identical `401` message for unknown-email vs wrong-password; bcrypt compare runs on a dummy hash when the user is absent (constant-ish timing). |
| Forgetting to protect a new route | High | Deny-by-default middleware: routes are protected unless explicitly allowlisted (BR-301). |
| Trusting a client-supplied `user_id` | High | Identity only from the session context (BR-304); reviewed; integration test for the seam. |
| Cookie misconfig (missing HttpOnly/Secure) | Medium | `Secure` derived from env; `HttpOnly`+`SameSite` always; documented in `.env.example`. |
| bcrypt cost too low/high | Low | Sensible default (12), documented; tunable later. |
| `auth` core importing SQL/HTTP types | Medium | Ports in core, adapters in subpackages; review + build. |

---

## 9. Validation Checklist

### Functional Validation
- [ ] FR-301…FR-307 acceptance criteria satisfied.
- [ ] register/login/logout/me behave per §7 of the spec; generic auth errors.
- [ ] Deny-by-default holds; public allowlist is exactly the five routes.

### Technical Validation
- [ ] Passwords only as bcrypt hashes; only `sha256(token)` persisted; logout deletes.
- [ ] `UserID` flows via context; no client `user_id` trusted (BR-304).
- [ ] `auth` core imports no SQL/HTTP/vendor-SDK types (BR-306).

### Quality Validation
- [ ] Unit tests pass with no DB; integration tests pass with one (incl. no-plaintext +
      isolation).
- [ ] `go build`/`go vet`/`gofmt`/`golangci-lint` clean; `go mod tidy`.
- [ ] Code reviewed; CHANGELOG updated in the same PR.

---

## 10. Definition of Done

- [ ] All phases complete; SPEC-003 acceptance criteria met.
- [ ] register→login→me→logout works end-to-end; protected routes 401 without a session.
- [ ] `0002_auth` up/down round-trip green; no-plaintext + isolation integration tests pass.
- [ ] Tests + lint/vet/fmt clean; `.env.example` documents the new vars.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-003 marked Done in the index.
- [ ] PR reviewed and merged to `main`.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-003-aula.html` produced.

---

## 11. Deliverables

### Code Deliverables
- `internal/auth/*` (domain, ports, service, token, context), `internal/auth/bcrypt`,
  `internal/auth/postgres`; `transport/http/auth.go` + `authmw.go`; `main.go` wiring;
  config additions.

### Infrastructure Deliverables
- `migrations/0002_auth.{up,down}.sql`; `.env.example` updates.

### Documentation Deliverables
- Updated `CHANGELOG.md`, `README.md`, specs/plans indexes; PT-BR lesson HTML.

---

## 12. Post-Implementation Tasks

### Monitoring
- None yet (auth metrics are SPEC-004). Confirm failed-login logs are useful and
  password/token never appear in logs.

### Future Improvements
- Email verification + password reset (needs a free email path).
- Rate limiting / account lockout; periodic expired-session cleanup sweep.
- Postgres RLS as defense-in-depth over app-level scoping; argon2id; OAuth; Bearer-token
  support for cross-origin clients.

### Technical Debt
- Single absolute session TTL (no sliding/refresh) — revisit if UX needs "remember me".
