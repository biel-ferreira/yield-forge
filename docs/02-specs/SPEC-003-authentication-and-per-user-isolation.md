# Feature Specification (SPEC)

## 1. Document Information

| Field        | Value                                                  |
| ------------ | ------------------------------------------------------ |
| Feature Name | Authentication & Per-User Isolation                    |
| Feature ID   | SPEC-003 (foundational)                                |
| Related PRD  | [PRD.md](../01-product/PRD.md) — FR-015, §10 NFR (Security), A1 |
| Related ADRs | [ADR-0002](../04-architecture/adr/ADR-0002-tech-stack-and-layering.md), [ADR-0003](../04-architecture/adr/ADR-0003-zero-cost-and-pluggable-llm.md) |
| Version      | 0.1.0                                                  |
| Status       | Approved                                               |
| Plan         | [PLAN-003](../03-plans/PLAN-003-authentication-and-per-user-isolation.md) |

---

## 2. Overview

### Purpose

Give the application **identity and per-user data isolation**: users register and
log in, requests carry an authenticated identity, and every protected endpoint is
gated by that identity. This is the seam that lets feature specs (SPEC-101+) scope
all data with `user_id` — turning the convention SPEC-002 reserved into an enforced
boundary.

After this spec, the app has a `users` table, sessions, `register`/`login`/`logout`/`me`
endpoints, an **auth middleware** that protects business routes, and a request-scoped
**current-user** value that feature repositories will use to scope queries — but still
**no portfolio/feature data** of its own.

### Business Value

- **Unblocks every feature.** FR-001/002 ("scoped to the authenticated user") and the
  security NFR ("all endpoints require auth; data strictly isolated per user") cannot
  be satisfied without this. It is the third foundation stone.
- **Multi-user from day one (A1).** The MVP is effectively single-user (the author),
  but the data model and auth are multi-user, so onboarding a second user later needs
  no redesign.
- **Security baseline.** Establishes password hashing, session handling, and the
  "deny by default" posture the rest of the product inherits.
- **Zero cost (ADR-0003).** Self-contained email+password auth with sessions in our
  own Postgres — no paid identity provider, no external dependency.

### Scope

**In scope:** `users` + `sessions` tables (migration `0002`); password hashing;
register/login/logout/me endpoints; opaque session tokens; auth middleware; a
request-scoped current-user (`UserID` in `context`); a per-user isolation helper +
documented convention; unit + integration tests; CHANGELOG/README/lesson.

**Out of scope (later specs or future work):**
- Social / OAuth login (Google, etc.), magic links → future (needs external setup).
- Email verification, password reset, "forgot password" → future (**needs email
  sending**, deferred to keep zero-cost simple; the schema leaves room).
- MFA / TOTP, account lockout policies, password-strength meters → future hardening.
- Roles / permissions (RBAC) → not needed for a single-role product yet.
- Any feature table or business endpoint → feature specs (SPEC-1xx). SPEC-003 only
  proves isolation *mechanically*, with no real per-user data to isolate yet.
- A login UI → the future Next.js frontend; SPEC-003 is API-only.

---

## 3. Functional Requirements

> SPEC-003 implements PRD **FR-015** directly and provides the enforcement seam for
> the per-user scoping clauses in FR-001/FR-002 and the security NFR.

### FR-301 — User Registration

A new user is created from an email and a password; the password is never stored in
plaintext.

**Acceptance Criteria**
- [ ] `POST /auth/register` with `{email, password}` creates a user and returns `201`.
- [ ] Email is normalized (trimmed + lowercased) and **unique** — a duplicate returns
      `409` with a generic message (no account enumeration beyond the conflict status).
- [ ] Password is validated against a minimum policy (e.g. length ≥ 8) and stored only
      as a **hash** (never plaintext, never logged).
- [ ] Invalid input (malformed email, short password) returns `400` with a clear,
      field-level but non-sensitive error.

### FR-302 — Login & Session Issuance

Valid credentials establish an authenticated session.

**Acceptance Criteria**
- [ ] `POST /auth/login` with correct `{email, password}` returns `200` and issues a
      session (see §14-D1/D3 for the transport).
- [ ] Wrong email **or** wrong password returns the **same** `401` generic message
      ("invalid email or password") — no hint about which was wrong (anti-enumeration).
- [ ] Password verification is constant-time (provided by the hashing algorithm).
- [ ] The raw session token is high-entropy and only its **hash** is stored server-side
      (a DB leak must not yield usable sessions).
- [ ] Sessions have an expiry; an expired session is rejected.

### FR-303 — Logout & Session Revocation

A user can end their session, and it is invalidated server-side.

**Acceptance Criteria**
- [ ] `POST /auth/logout` invalidates the current session (deletes/!expires it
      server-side) and clears the client token; returns `204`.
- [ ] A revoked/expired session token is rejected by the auth middleware (`401`).

### FR-304 — Current User (`/auth/me`)

An authenticated request can read its own identity.

**Acceptance Criteria**
- [ ] `GET /auth/me` with a valid session returns `200 {"id","email"}` (never the
      password hash).
- [ ] Without a valid session it returns `401`.

### FR-305 — Auth Middleware & Deny-by-Default

Protected routes require a valid session; public routes are explicitly listed.

**Acceptance Criteria**
- [ ] An auth middleware resolves the session → loads the user → injects an
      authenticated **`UserID`** into the request `context`.
- [ ] Public routes are an explicit allowlist: `/healthz`, `/readyz`, `/version`,
      `/auth/register`, `/auth/login`. Everything else requires auth (deny by default).
- [ ] A missing/invalid/expired session on a protected route returns `401` JSON (no
      redirect, no HTML).
- [ ] Handlers retrieve the current user via a typed helper (e.g. `auth.UserID(ctx)`),
      never by re-parsing the token.

### FR-306 — Per-User Isolation Seam

The mechanism that scopes data to the owning user exists and is proven, ready for
feature repositories to adopt.

**Acceptance Criteria**
- [ ] A documented pattern: feature repositories take the authenticated `UserID` and
      scope every query (`WHERE user_id = $1`); the ID comes from `context`, never from
      client-supplied input (no `user_id` in request bodies/paths is trusted).
- [ ] A typed context key + accessor (`auth.UserID(ctx)`) is provided; reading it when
      unauthenticated is a clear programming error (not a silent zero value).
- [ ] An integration test proves the seam end-to-end (e.g. two users; a request
      authenticated as user A carries A's id, not B's).
- [ ] No feature domain type is introduced here (isolation is proven at the auth level).

### FR-307 — CHANGELOG, README & Lesson

**Acceptance Criteria**
- [ ] `CHANGELOG.md` `[Unreleased]` records the auth work.
- [ ] `README.md` documents the auth endpoints, the new env vars, and the protected-by-
      default posture.
- [ ] A PT-BR HTML lesson `docs/lessons/SPEC-003-aula.html` is produced on close.

---

## 4. User Flows

### Flow 1 — Register → use the app
1. `POST /auth/register {email, password}` → `201`.
2. `POST /auth/login {email, password}` → `200` + session issued.
3. Client calls a protected endpoint with the session → request carries `UserID`.

### Flow 2 — Wrong credentials
1. `POST /auth/login` with a bad password → `401 {"error":"invalid email or password"}`.
2. Same generic response whether the email exists or not.

### Flow 3 — Expired/!revoked session
1. Client calls a protected route with an expired/logged-out session.
2. Auth middleware rejects it → `401`. `/healthz` etc. still public.

### Flow 4 — Logout
1. `POST /auth/logout` (authenticated) → session invalidated server-side, token cleared.
2. Reusing the old token → `401`.

---

## 5. Business Rules (Architectural & Security)

- **BR-301 — Deny by default.** Every route is protected unless explicitly public.
  Adding a new route is secure-by-default (it requires auth until allowlisted).
- **BR-302 — Passwords are only ever stored hashed.** Never plaintext, never logged,
  never returned. Hashing is a one-way, salted, slow algorithm (§14-D2).
- **BR-303 — Only the session token *hash* is persisted.** The raw token lives only in
  the client; the server stores `sha256(token)`. A DB compromise yields no live sessions.
- **BR-304 — Identity comes from the session, never the request payload.** The owning
  `user_id` is taken from the authenticated context; a `user_id` in a body/path/query is
  never trusted. This is what makes per-user isolation real.
- **BR-305 — Generic auth failures.** Login/registration errors must not enable account
  enumeration (same message/timing for "no such user" and "wrong password").
- **BR-306 — `auth` is a feature package; SQL stays in its adapter.** `internal/auth`
  owns the domain + service + ports; the Postgres adapter implements them (SPEC-001
  BR-001/002, SPEC-002 BR-202). The HTTP middleware lives in `transport/http` and calls
  the auth service.
- **BR-307 — Sessions expire.** Every session has an absolute expiry; expired sessions
  are invalid regardless of server state.

---

## 6. Domain Model

### Entity: User
| Field         | Type      | Notes                                        |
| ------------- | --------- | -------------------------------------------- |
| id            | uuid      | PK, `gen_random_uuid()` (SPEC-002 convention) |
| email         | text      | unique, normalized lowercase                 |
| password_hash | text      | hash only — never exposed                    |
| created_at    | timestamptz | UTC                                        |
| updated_at    | timestamptz | UTC                                        |

### Entity: Session
| Field      | Type      | Notes                                            |
| ---------- | --------- | ------------------------------------------------ |
| id         | uuid      | PK                                               |
| user_id    | uuid      | FK → users(id) `ON DELETE CASCADE`               |
| token_hash | text      | unique; `sha256` of the raw token (BR-303)       |
| expires_at | timestamptz | absolute expiry (BR-307)                       |
| created_at | timestamptz | UTC                                            |

Ports (defined in `internal/auth`, per BR-306):
- `UserRepository` — create, find by email, find by id.
- `SessionRepository` — create, find by token hash (valid/unexpired), delete.
- `PasswordHasher` — `Hash(plain) (string, error)` / `Compare(hash, plain) error`
  (so the algorithm in §14-D2 is swappable and testable).

---

## 7. API Specification

```
POST /auth/register      {"email","password"}      201 {"id","email"}        | 400 | 409
POST /auth/login         {"email","password"}      200 {"id","email"} + session | 401
POST /auth/logout        (authenticated)           204                        | 401
GET  /auth/me            (authenticated)           200 {"id","email"}         | 401
```

- All bodies/responses are `application/json`. The password is **never** echoed back.
- Generic error envelope: `{"error":"<message>"}`. Auth failures use the same
  message/status regardless of cause (BR-305).
- Session transport (cookie vs bearer header) is decided in §14-D3.

---

## 8. Data Storage

Migration **`0002_auth`** (up/down), following SPEC-002's tooling and conventions:
- `users` and `sessions` tables as in §6.
- Indexes: unique on `users.email`; unique on `sessions.token_hash`; index on
  `sessions.user_id`; (optional) index on `sessions.expires_at` for cleanup.
- A documented strategy for expired-session cleanup (lazy delete on access now; a
  periodic sweep can come with SPEC-004). No feature tables.

---

## 9. Edge Cases

| Scenario | Expected behaviour |
| -------- | ------------------ |
| Register with an existing email | `409` generic conflict; no second account. |
| Register with weak/short password | `400` with a non-sensitive validation message. |
| Login, wrong password | `401` "invalid email or password" (same as unknown email). |
| Login, unknown email | `401`, identical message + comparable timing (BR-305). |
| Valid session, then logout, then reuse token | `401` (revoked server-side). |
| Expired session | `401`; treated as no session. |
| Protected route, no session | `401` JSON (no redirect/HTML). |
| `user_id` supplied in a request body | Ignored; identity always from the session (BR-304). |
| Concurrent logins (same user) | Multiple valid sessions allowed (each its own token). |

---

## 10. Security Considerations

- **Hashing:** salted, slow algorithm (§14-D2); cost/params tuned and documented.
  Password never logged or returned.
- **Sessions:** high-entropy token from `crypto/rand`; only `sha256(token)` stored
  (BR-303); absolute expiry (BR-307); logout revokes server-side.
- **Cookie hardening (if D3 = cookie):** `HttpOnly`, `Secure` (non-dev), `SameSite`
  (Lax/Strict) to blunt XSS token theft and CSRF; `Secure` relaxed only in local dev.
- **Anti-enumeration:** generic auth errors (BR-305).
- **Transport:** credentials only over TLS in hosted environments (PRD §10); local dev
  over plain HTTP is acceptable and documented.
- **Secrets:** any auth secret (e.g. a cookie/signing key, session TTL) comes from the
  environment (SPEC-001 BR-004); never committed.
- **Brute force:** basic mitigation now (constant-time compare, generic errors); rate
  limiting / lockout is noted as future hardening (§15), not built here.

---

## 11. Observability

- **Logs:** auth events at appropriate levels — registration, successful login,
  logout, and **failed** login (at `warn`, with email **redacted/partial** and no
  password). A request's authenticated `user_id` is added to the request log once
  resolved (helps trace per-user activity).
- **No secrets in logs:** never the password, raw token, or full hash.
- **Metrics/traces:** out of scope (SPEC-004); the middleware is structured so SPEC-004
  can add auth metrics without rework.

---

## 12. Testing Strategy

### Unit Tests
- `PasswordHasher`: hash→compare round-trips; wrong password fails; hash ≠ plaintext.
- Auth `Service` with **fake** repositories: register (new + duplicate), login (good +
  bad), logout; email normalization; generic errors (BR-305).
- Auth **middleware** with a fake session lookup: valid → injects `UserID`; missing/
  invalid/expired → `401`; public routes bypass.
- `auth.UserID(ctx)` accessor: present vs absent behaviour.

### Integration Tests (real Postgres, gated like SPEC-002 — `testing.Short()` +
`TEST_DATABASE_URL`)
- Full flow: register → login → `/auth/me` → logout → `/auth/me` is `401`.
- Persistence: only the password **hash** and token **hash** are stored (assert no
  plaintext in the row).
- Isolation seam (FR-306): two users; a request authenticated as A resolves to A's id,
  not B's.

### Quality gate
- `go build`/`go vet`/`gofmt` clean; unit tests pass without a DB; integration with one;
  dependency-direction rules hold (`auth` core imports no SQL/HTTP types).

---

## 13. Definition of Done

- [ ] `internal/auth` (domain + service + ports) and its Postgres adapter implemented.
- [ ] Migration `0002_auth` (users + sessions) with a tested `down`.
- [ ] Password hashing (§14-D2) behind a `PasswordHasher` port.
- [ ] `register`/`login`/`logout`/`me` endpoints with generic auth errors.
- [ ] Auth middleware (deny-by-default + public allowlist) injecting `UserID`; typed
      `auth.UserID(ctx)` accessor.
- [ ] Session tokens: `crypto/rand` token, `sha256` hash stored, expiry, logout revokes.
- [ ] Per-user isolation seam documented + proven by an integration test (FR-306).
- [ ] New config (session TTL, cookie/secret settings) env-driven; `.env.example` updated.
- [ ] Unit tests (no DB) + integration tests (real Postgres) pass; build/vet/fmt clean.
- [ ] `CHANGELOG.md` + `README.md` updated; SPEC-003/PLAN-003 flipped to Done.
- [ ] PT-BR HTML lesson `docs/lessons/SPEC-003-aula.html` produced.

---

## 14. Decisions (resolved)

> Confirmed with the project owner before PLAN-003. These are now binding.

- **D1 — Email + password, server-side sessions.** ✅ Self-contained, zero-cost,
  revocable, high learning value; fits the effectively-single-user MVP (A1). JWT and
  OAuth rejected for now (both addable later behind the same `auth` seam).
- **D2 — bcrypt** behind the `PasswordHasher` port. ✅ In `golang.org/x/crypto`,
  battle-tested, constant-time compare built in. argon2id can replace it later via the
  port with no ripple.
- **D3 — `HttpOnly` + `Secure` + `SameSite` cookie.** ✅ Most secure default (resists XSS
  token theft); `Secure` relaxed only in local dev. Bearer-token support can be added
  additively for cross-origin clients later.
- **D4 — App-level `user_id` scoping** in repositories. ✅ Transparent and easy to reason
  about; Postgres RLS deferred as future defense-in-depth on top of it.
- **Logout = hard `DELETE`** of the session row (not a soft-delete/update) — expiry
  (`expires_at`) covers natural expiry, so a delete keeps the table lean.

---

## 15. Open Questions (deferred, not blocking)

- Email verification & password reset — deferred until a free email path is chosen
  (Resend/SMTP free tier); the schema reserves room.
- Session lifetime policy (absolute vs sliding/refresh) — start with a single absolute
  TTL (env-configurable); revisit if UX needs "remember me".
- Rate limiting / account lockout on repeated failed logins — future hardening (maybe
  alongside SPEC-004 observability).
- Expired-session cleanup job — lazy delete now; periodic sweep later.
