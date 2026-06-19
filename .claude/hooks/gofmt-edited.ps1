# PostToolUse(Edit|Write) hook.
# Promotes the CLAUDE.md "always gofmt" soft rule to a HARD, harness-enforced rule:
# whenever Claude edits a Go file, it is gofmt'd immediately — independent of memory.
# Reads the tool payload from stdin (no jq needed; uses built-in ConvertFrom-Json).
$ErrorActionPreference = 'SilentlyContinue'

$payload = [Console]::In.ReadToEnd() | ConvertFrom-Json
$path = $payload.tool_input.file_path

if ($path -and ($path -like '*.go') -and (Test-Path -LiteralPath $path)) {
    gofmt -w -- $path
}

exit 0
