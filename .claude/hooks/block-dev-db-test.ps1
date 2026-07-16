# PreToolUse(Bash|PowerShell) hook — blocks TEST_DATABASE_URL from ever pointing at the
# docker-compose DEV database (host port 5433 / db name yieldforge_dev).
#
# Why: integration tests (internal/*/postgres/*_integration_test.go) TRUNCATE tables as part of
# their setup (e.g. portfolioDB(t) in internal/portfolio/postgres/postgres_integration_test.go
# runs "TRUNCATE users CASCADE"). Pointing TEST_DATABASE_URL at the shared dev DB wipes real,
# hand-seeded dev data — this happened once (SPEC-007 implementation session) and is not
# acceptable to repeat. TEST_DATABASE_URL must point at a disposable Postgres the caller can
# freely truncate/drop — never the persistent dev stack.
#
# Blocks by exiting 2 with a message on stderr (shown to Claude); exit 0 = allow.
$ErrorActionPreference = 'SilentlyContinue'

$payload = [Console]::In.ReadToEnd() | ConvertFrom-Json
$command = $payload.tool_input.command
if (-not $command) { exit 0 }

if ($command -notmatch 'TEST_DATABASE_URL') { exit 0 }

# The dev DB is identifiable by its published host port (5433) or its database name
# (yieldforge_dev) — either marker in a TEST_DATABASE_URL value means it's the dev stack.
if ($command -match '5433' -or $command -match 'yieldforge_dev') {
    [Console]::Error.WriteLine(
        "BLOCKED: TEST_DATABASE_URL points at the dev database (port 5433 / yieldforge_dev). " +
        "Integration tests TRUNCATE tables and will destroy real dev data. Use a disposable " +
        "Postgres instead, e.g.:`n" +
        "  docker run --rm -d --name yf-test-pg -e POSTGRES_USER=yieldforge " +
        "-e POSTGRES_PASSWORD=yieldforge -e POSTGRES_DB=yieldforge_test -p 5434:5432 postgres:16-alpine`n" +
        "  TEST_DATABASE_URL=`"postgres://yieldforge:yieldforge@localhost:5434/yieldforge_test?sslmode=disable`"`n" +
        "See CLAUDE.md's Testing conventions."
    )
    exit 2
}

exit 0
