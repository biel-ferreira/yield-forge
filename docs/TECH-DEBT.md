# Tech Debt

Known, accepted shortcuts and idiom deviations to be paid down later. Each item: what,
why it's debt, and the fix. Keep newest on top. Close an item by removing it (and noting
the fix in `CHANGELOG.md`).

## Open

### TD-001 — Package-name stutter in core types

**What:** some core types repeat their package name —
`profile.ProfileRepository`, `profile.ProfileReader`, `insight.InsightRequest`,
`insight.InsightResult`.

**Why it's debt:** non-idiomatic Go (Effective Go: avoid stutter). The convention is now
codified in `CLAUDE.md` ("Avoid package-name stutter") for new code, but the existing
identifiers predate it.

**Fix:** rename to `profile.Repository` / `profile.Reader` / `insight.Request` /
`insight.Result` and update call sites. Mechanical but cross-cutting (touches ports,
adapters, transport, tests) — schedule as a standalone `refactor:` change so it doesn't
muddy a feature diff. Adapter/transport names that disambiguate
(`postgres.ProfileRepository`, `http.ProfileService`) stay as-is.

**Added:** 2026-06-25.
