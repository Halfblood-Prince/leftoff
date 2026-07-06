param(
    [string]$Version = "",
    [string]$Repo = "Halfblood-Prince/leftoff",
    [switch]$Yes
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$PluginRoot = Resolve-Path (Join-Path $ScriptDir "..")

if ([string]::IsNullOrWhiteSpace($Version)) {
    $versionPath = Join-Path $PluginRoot "VERSION"
    if (-not (Test-Path -LiteralPath $versionPath)) {
        throw "missing -Version and VERSION file"
    }
    $Version = "v$((Get-Content -Raw -LiteralPath $versionPath).Trim())"
}
if (-not $Version.StartsWith("v")) {
    $Version = "v$Version"
}

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    throw "gh is required to download and verify release provenance"
}

$goos = if ($IsLinux) {
    "linux"
} elseif ($IsMacOS) {
    "darwin"
} else {
    "windows"
}

$archName = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString()
$goarch = switch ($archName) {
    "X64" { "amd64" }
    "Arm64" { "arm64" }
    default { throw "unsupported architecture: $archName" }
}

$asset = "leftoff_${Version}_${goos}_${goarch}.zip"
$installDir = Join-Path $PluginRoot "bin\.leftoff\${goos}_${goarch}"

if (-not $Yes) {
    Write-Output "This will download from GitHub:"
    Write-Output "  repo:  $Repo"
    Write-Output "  tag:   $Version"
    Write-Output "  asset: $asset"
    Write-Output ""
    Write-Output "The script will verify GitHub artifact provenance and SHA256SUMS before"
    Write-Output "installing the binary under this plugin root:"
    Write-Output "  $installDir"
    $answer = Read-Host "Continue? [y/N]"
    if ($answer -notin @("y", "Y", "yes", "YES")) {
        Write-Output "cancelled"
        exit 1
    }
}

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) "leftoff-setup-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

try {
    & gh release download $Version --repo $Repo --pattern $asset --pattern SHA256SUMS --dir $tmp
    if ($LASTEXITCODE -ne 0) { throw "gh release download failed" }

    & gh attestation verify (Join-Path $tmp $asset) --repo $Repo
    if ($LASTEXITCODE -ne 0) { throw "gh attestation verify failed" }

    $sumPath = Join-Path $tmp "SHA256SUMS"
    $line = Get-Content -LiteralPath $sumPath | Where-Object { $_ -match "\s$([regex]::Escape($asset))$" } | Select-Object -First 1
    if (-not $line) {
        throw "missing SHA256SUMS entry for $asset"
    }
    $expected = ($line -split "\s+")[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $tmp $asset)).Hash.ToLowerInvariant()
    if ($expected -ne $actual) {
        throw "checksum mismatch for $asset"
    }

    $extractDir = Join-Path $tmp "extract"
    Expand-Archive -LiteralPath (Join-Path $tmp $asset) -DestinationPath $extractDir -Force

    $exe = if ($goos -eq "windows") { "leftoff.exe" } else { "leftoff" }
    $sourceBin = Join-Path $extractDir "leftoff_${Version}_${goos}_${goarch}\bin\$exe"
    if (-not (Test-Path -LiteralPath $sourceBin -PathType Leaf)) {
        throw "release bundle did not contain bin/$exe"
    }

    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Copy-Item -LiteralPath $sourceBin -Destination (Join-Path $installDir $exe) -Force

    Write-Output "installed verified leftoff binary: $(Join-Path $installDir $exe)"
    Write-Output "launcher: $(Join-Path $PluginRoot 'bin\leftoff.ps1')"
} finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
