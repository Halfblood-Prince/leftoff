$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$PluginRoot = Resolve-Path (Join-Path $ScriptDir "..")

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
$candidates = @(
    (Join-Path $ScriptDir ".leftoff\$platform\$exe"),
    (Join-Path $ScriptDir "$platform\$exe")
)

foreach ($candidate in $candidates) {
    if (Test-Path -LiteralPath $candidate -PathType Leaf) {
        & $candidate @args
        exit $LASTEXITCODE
    }
}

if (Get-Command go -ErrorAction SilentlyContinue) {
    Push-Location $PluginRoot
    try {
        & go run ./cmd/leftoff @args
        exit $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

Write-Error @"
leftoff binary is not installed for $platform.

Ask the user for explicit approval before network access, then run:
  powershell -ExecutionPolicy Bypass -File .\scripts\setup-binary.ps1

If Go 1.22+ is installed, the launcher can also run the source fallback.
"@
exit 127
