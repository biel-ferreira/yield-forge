# PostToolUse(Edit|Write) hook — the frontend mirror of gofmt-edited.ps1.
# Promotes the web/CLAUDE.md "format the frontend" convention to a HARD, harness-enforced
# rule: whenever Claude edits a web/ code/style file, it is Prettier-formatted immediately.
# NON-BLOCKING by design: if Node or the repo-local Prettier isn't available, it no-ops
# (exit 0) and NEVER blocks the edit. Respects web/.prettierignore (so the generated
# lib/api/schema.ts is left untouched — keeping it in lockstep with `npm run gen:api`).
$ErrorActionPreference = 'SilentlyContinue'

$payload = [Console]::In.ReadToEnd() | ConvertFrom-Json
$path = $payload.tool_input.file_path
if (-not $path) { exit 0 }

# Only web/ files Prettier owns.
if ($path -notmatch '[\\/]web[\\/]') { exit 0 }
if ($path -notmatch '\.(ts|tsx|js|jsx|mjs|cjs|css|json)$') { exit 0 }
if (-not (Test-Path -LiteralPath $path)) { exit 0 }

# Need Node on PATH (restart the session after `nvm use 20` so the hook inherits it).
$node = Get-Command node -ErrorAction SilentlyContinue
if (-not $node) { exit 0 }

# Use the repo-local Prettier (zero global deps).
$prettierCli = Join-Path $env:CLAUDE_PROJECT_DIR 'web/node_modules/prettier/bin/prettier.cjs'
if (-not (Test-Path -LiteralPath $prettierCli)) { exit 0 }

& $node.Source $prettierCli --write --log-level silent --ignore-unknown -- $path | Out-Null
exit 0
