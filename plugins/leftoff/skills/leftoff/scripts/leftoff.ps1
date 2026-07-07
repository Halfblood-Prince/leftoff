$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$PluginRoot = Resolve-Path (Join-Path $ScriptDir "..")
$BinHome = if ($env:LEFTOFF_BIN_HOME) { $env:LEFTOFF_BIN_HOME } else { Join-Path $HOME ".leftoff\bin" }

function Get-PlatformKey {
    $os = if ($IsLinux) {
        "linux"
    } elseif ($IsMacOS) {
        "darwin"
    } else {
        "windows"
    }

    $archName = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString()
    $arch = switch ($archName) {
        "X64" { "amd64" }
        "Arm64" { "arm64" }
        default { $archName.ToLowerInvariant() }
    }

    return "${os}_${arch}"
}

$platform = Get-PlatformKey
$exe = if ($platform.StartsWith("windows_")) { "leftoff.exe" } else { "leftoff" }
$setupScript = Join-Path $PluginRoot "scripts\setup-binary.ps1"
$localSetupScript = Join-Path $ScriptDir "setup-binary.ps1"
if (Test-Path -LiteralPath $localSetupScript -PathType Leaf) {
    $setupScript = $localSetupScript
}
$candidates = @(
    (Join-Path $BinHome "$platform\$exe"),
    (Join-Path $ScriptDir "$platform\$exe")
)

foreach ($candidate in $candidates) {
    if (Test-Path -LiteralPath $candidate -PathType Leaf) {
        & $candidate @args
        exit $LASTEXITCODE
    }
}

Write-Error @"
leftoff needs a local binary.

Platform: $platform

Option 1: Run setup with explicit approval:
  powershell -ExecutionPolicy Bypass -File "$setupScript"

Option 2: Install Go 1.22+ and use the Go install fallback:
  go install github.com/Halfblood-Prince/leftoff/cmd/leftoff@latest

Option 3: Download the verified release archive manually.
"@
exit 127
