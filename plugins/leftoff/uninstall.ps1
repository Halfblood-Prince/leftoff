param(
    [string]$Agent = "generic",
    [string]$Target = "",
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

function Get-DefaultTarget {
    param([string]$AgentName)

    switch ($AgentName) {
        "generic" { return "$HOME\.leftoff\skills\leftoff" }
        { $_ -in @("claude-code", "claude") } { return "$HOME\.claude\skills\leftoff" }
        "codex" {
            if ($env:CODEX_HOME) {
                return (Join-Path $env:CODEX_HOME "skills\leftoff")
            }
            return "$HOME\.codex\skills\leftoff"
        }
        "cursor" { return "$HOME\.cursor\skills\leftoff" }
        "pi" { return "$HOME\.pi\skills\leftoff" }
        { $_ -in @("github-copilot-cli", "copilot-cli", "copilot") } { return "$HOME\.github-copilot-cli\skills\leftoff" }
        "opencode" { return "$HOME\.opencode\skills\leftoff" }
        { $_ -in @("gemini-cli-antigravity", "gemini", "google-gemini", "antigravity") } { return "$HOME\.gemini\skills\leftoff" }
        { $_ -in @("factory-ai-droid", "factory-droid", "droid") } { return "$HOME\.factory-ai-droid\skills\leftoff" }
        "openclaw" { return "$HOME\.openclaw\skills\leftoff" }
        { $_ -in @("hermes-agent", "hermes") } { return "$HOME\.hermes\skills\leftoff" }
        "astrbot" { return "$HOME\.astrbot\skills\leftoff" }
        "nanoclaw" { return "$HOME\.nanoclaw\skills\leftoff" }
        "shelley" { return "$HOME\.shelley\skills\leftoff" }
        { $_ -in @("auggie-augment", "auggie", "augment") } { return "$HOME\.augment\skills\leftoff" }
        { $_ -in @("cline-roo-code", "cline", "roo", "roo-code") } { return "$HOME\.cline\skills\leftoff" }
        "codebuddy" { return "$HOME\.codebuddy\skills\leftoff" }
        "continue" { return "$HOME\.continue\skills\leftoff" }
        "crush" { return "$HOME\.crush\skills\leftoff" }
        { $_ -in @("deep-agents", "deepagents") } { return "$HOME\.deep-agents\skills\leftoff" }
        "firebender" { return "$HOME\.firebender\skills\leftoff" }
        "forgecode" { return "$HOME\.forgecode\skills\leftoff" }
        "goose" { return "$HOME\.goose\skills\leftoff" }
        "junie" { return "$HOME\.junie\skills\leftoff" }
        { $_ -in @("kilo-code", "kilocode") } { return "$HOME\.kilo-code\skills\leftoff" }
        { $_ -in @("kimi-code-cli", "kimi") } { return "$HOME\.kimi-code-cli\skills\leftoff" }
        { $_ -in @("kiro-cli", "kiro") } { return "$HOME\.kiro\skills\leftoff" }
        "lingma" { return "$HOME\.lingma\skills\leftoff" }
        "mistral-vibe" { return "$HOME\.mistral-vibe\skills\leftoff" }
        "mux" { return "$HOME\.mux\skills\leftoff" }
        "openhands" { return "$HOME\.openhands\skills\leftoff" }
        "qoder" { return "$HOME\.qoder\skills\leftoff" }
        { $_ -in @("qwen-code", "qwen") } { return "$HOME\.qwen-code\skills\leftoff" }
        { $_ -in @("rovo-dev", "rovo") } { return "$HOME\.rovo\skills\leftoff" }
        { $_ -in @("tabnine-cli", "tabnine") } { return "$HOME\.tabnine\skills\leftoff" }
        { $_ -in @("trae-trae-cn", "trae", "trae-cn") } { return "$HOME\.trae\skills\leftoff" }
        "warp" { return "$HOME\.warp\skills\leftoff" }
        "windsurf" { return "$HOME\.windsurf\skills\leftoff" }
        "zed" { return "$HOME\.zed\skills\leftoff" }
        default { throw "unsupported agent: $AgentName; see agents/supported.md" }
    }
}

if ([string]::IsNullOrWhiteSpace($Target)) {
    $Target = Get-DefaultTarget -AgentName $Agent
}

Write-Output "Agent target: $Agent"
Write-Output "Uninstall target: $Target"
Write-Output "Data store retained: $HOME\.leftoff"

if ($DryRun) {
    Write-Output "Dry run: no files will be changed."
    exit 0
}

if (-not (Test-Path -LiteralPath $Target)) {
    Write-Output "nothing installed at $Target"
    exit 0
}

$answer = Read-Host "Type 'remove leftoff' to continue"
if ($answer -ne "remove leftoff") {
    throw "aborted"
}

Remove-Item -LiteralPath $Target -Recurse -Force
Write-Output "removed $Target"
