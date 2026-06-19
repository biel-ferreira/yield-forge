# PreToolUse(Edit|Write) hook — enforces the working agreement's immutability rules
# as a HARD rule: once committed to git, these files must be superseded, never edited.
#   - migrations/*.sql        (a migration is immutable once committed/applied)
#   - .../adr/ADR-*.md        (ADRs are immutable once accepted; supersede, never edit)
# New (untracked) files are allowed — you create the next migration / next ADR freely.
# Blocks by exiting 2 with a message on stderr (shown to Claude); exit 0 = allow.
$ErrorActionPreference = 'SilentlyContinue'

$payload = [Console]::In.ReadToEnd() | ConvertFrom-Json
$path = $payload.tool_input.file_path
if (-not $path) { exit 0 }

# Normalize separators for matching.
$norm = $path -replace '\\', '/'

$isMigration = $norm -match '/migrations/[^/]+\.sql$'
$isADR = $norm -match '/adr/ADR-[^/]+\.md$'
if (-not ($isMigration -or $isADR)) { exit 0 }

if ($env:CLAUDE_PROJECT_DIR) { Set-Location -LiteralPath $env:CLAUDE_PROJECT_DIR }

# Only block if the file is already tracked by git (i.e. committed → immutable).
& git ls-files --error-unmatch -- "$path" *> $null
if ($LASTEXITCODE -eq 0) {
    $kind = if ($isMigration) { 'migration' } else { 'ADR' }
    $rule = if ($isMigration) {
        'Migrations are append-only once committed/applied. Create a NEW migration instead.'
    } else {
        'ADRs are immutable once accepted. Supersede with a NEW ADR instead of editing.'
    }
    [Console]::Error.WriteLine("BLOCKED: '$path' is a committed $kind. $rule")
    exit 2
}

exit 0
