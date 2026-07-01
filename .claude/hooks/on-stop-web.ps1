# Stop hook — non-blocking frontend backstop (the web/ mirror of on-stop.ps1). exit 0 always.
# Reminder-only: it does NOT run the (slow) typecheck/lint/build gate — it just nudges. It
# surfaces, as one systemMessage when relevant:
#   1. run the web/ gate before done, when web/ code changed;
#   2. CHANGELOG reminder — web/ changed but CHANGELOG.md did not (working agreement);
#   3. regen the typed API client — api/openapi.yaml changed but web/lib/api/schema.ts did not.
$ErrorActionPreference = 'SilentlyContinue'
$null = [Console]::In.ReadToEnd()  # drain stdin (fields unused)

if ($env:CLAUDE_PROJECT_DIR) { Set-Location -LiteralPath $env:CLAUDE_PROJECT_DIR }

$notes = @()

$changed = & git status --porcelain 2>$null
$webChanged       = $changed | Where-Object { $_ -match 'web/.+\.(ts|tsx|js|jsx|mjs|cjs|css)("?)$' }
$changelogChanged = $changed | Where-Object { $_ -match 'CHANGELOG\.md' }
$openapiChanged   = $changed | Where-Object { $_ -match 'api/openapi\.yaml' }
$schemaChanged    = $changed | Where-Object { $_ -match 'web/lib/api/schema\.ts' }

if ($webChanged) {
    $notes += "web/ changed — before done, run the frontend gate: (in web/) ``npm run typecheck`` + ``npm run lint`` + ``npm run build``."
    if (-not $changelogChanged) {
        $notes += "web/ changed but CHANGELOG.md was not updated (working agreement: update the [Unreleased] section in the same change)."
    }
}

if ($openapiChanged -and -not $schemaChanged) {
    $notes += "api/openapi.yaml changed but web/lib/api/schema.ts was not regenerated — run (in web/) ``npm run gen:api`` and commit it (the check:api drift guard will fail otherwise)."
}

if ($notes.Count -gt 0) {
    @{ systemMessage = ($notes -join "`n`n") } | ConvertTo-Json -Compress | Write-Output
}

exit 0
