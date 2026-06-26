# PostToolUse(Edit|Write) hook — promotes the #1 architecture rule from subjective
# review (hexagonal-reviewer) to a HARD, deterministic gate:
#   a feature CORE package must import NO SQL, HTTP, or vendor-SDK types.
# Ports (interfaces) live in the core; adapters implement them in subpackages at the
# edge (.../postgres, .../bcrypt, .../<provider>). See CLAUDE.md "Architecture rules".
#
# Scope: only top-level files of a feature core — internal/<feature>/<file>.go,
# where <feature> is auth|insight|marketdata|profile|portfolio|projection. Adapter
# SUBpackages (internal/<feature>/<adapter>/...), internal/platform/*, and
# internal/transport/* are intentionally allowed to import SQL/HTTP and are skipped.
#
# Reads the file from disk (true post-write state, no diff guessing) and blocks with
# exit 2 + a message on stderr (fed back to Claude to fix immediately). exit 0 = allow.
$ErrorActionPreference = 'SilentlyContinue'

$payload = [Console]::In.ReadToEnd() | ConvertFrom-Json
$path = $payload.tool_input.file_path
if (-not $path) { exit 0 }

# Normalize separators; only act on a feature-core top-level .go file.
$norm = $path -replace '\\', '/'
if ($norm -notmatch '/internal/(auth|insight|marketdata|profile|portfolio|projection)/[^/]+\.go$') {
    exit 0
}
if (-not (Test-Path -LiteralPath $path)) { exit 0 }

# Forbidden imports in a core package: stdlib SQL/HTTP and the pgx vendor SDK.
# Matches an import spec line (optional alias / blank / dot import) then the quoted path.
$forbidden = @()
foreach ($line in (Get-Content -LiteralPath $path)) {
    if ($line -match '^\s*(?:import\s+)?(?:[A-Za-z_]\w*\s+|\.\s+|_\s+)?"(database/sql|net/http|github\.com/jackc/pgx[^"]*)"') {
        $forbidden += $Matches[1]
    }
}
$forbidden = $forbidden | Sort-Object -Unique

if ($forbidden.Count -gt 0) {
    [Console]::Error.WriteLine(
        "BLOCKED: layering violation in '$path'. A feature core package must not import: " +
        ($forbidden -join ', ') +
        ". Move SQL/HTTP/vendor code into an adapter subpackage (e.g. .../postgres, .../<provider>) " +
        "and depend on a port (interface) defined in the core. (CLAUDE.md: domain core imports no SQL/HTTP/framework.)"
    )
    exit 2
}

exit 0
