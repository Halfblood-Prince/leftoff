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
leftoff needs a local binary.

Platform: $platform

Option 1: Run setup with explicit approval:
  powershell -ExecutionPolicy Bypass -File .\scripts\setup-binary.ps1

Option 2: Install Go 1.22+ and use the source fallback.

Option 3: Download the verified release archive manually.
"@
exit 127
