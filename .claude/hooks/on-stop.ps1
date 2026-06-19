# Stop hook — non-blocking quality backstop (never traps the session; exit 0 always).
# Surfaces two nudges as a single systemMessage when relevant:
#   1. `go vet ./...` findings — but ONLY when Go code is dirty (avoids latency/noise
#      on every turn-end, e.g. when Claude stops just to ask a question).
#   2. CHANGELOG reminder — if Go code changed but CHANGELOG.md did not, per the SDD
#      working agreement (update [Unreleased] in the same change).
$ErrorActionPreference = 'SilentlyContinue'
$null = [Console]::In.ReadToEnd()  # drain stdin (fields unused)

if ($env:CLAUDE_PROJECT_DIR) { Set-Location -LiteralPath $env:CLAUDE_PROJECT_DIR }

$notes = @()

$changed = & git status --porcelain 2>$null
$goChanged = $changed | Where-Object { $_ -match '\.go(?:"?)$' }
$changelogChanged = $changed | Where-Object { $_ -match 'CHANGELOG\.md' }

if ($goChanged) {
    $vet = (& go vet ./... 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -and $vet) {
        $notes += "go vet reported issues:`n$vet"
    }
    if (-not $changelogChanged) {
        $notes += "Go code changed but CHANGELOG.md was not updated (working agreement: update the [Unreleased] section in the same change)."
    }
}

if ($notes.Count -gt 0) {
    @{ systemMessage = ($notes -join "`n`n") } | ConvertTo-Json -Compress | Write-Output
}

exit 0
