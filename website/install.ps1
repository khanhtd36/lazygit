[CmdletBinding()]
param(
    [string]$InstallDir = $env:LAZYGIT_INSTALL_DIR
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repo = "khanhtd36/lazygit"
$Bin = "lazygit.exe"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message"
}

if ($env:OS -ne "Windows_NT") {
    Write-Error "install.ps1 supports Windows only. Use install.sh on Linux or macOS."
    exit 1
}

# Only x86_64 is shipped; it also runs fine under emulation on ARM64 Windows.
$architecture = [System.Runtime.InteropServices.RuntimeInformation,mscorlib]::OSArchitecture.ToString()
if ($architecture -eq "Arm64") {
    Write-Step "Windows ARM64 detected; installing the x86_64 build under Windows emulation."
} elseif ($architecture -ne "X64") {
    Write-Error "Unsupported Windows architecture: $architecture"
    exit 1
}

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\lazygit"
}

Write-Step "Fetching latest release info"
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$tag = [string]$release.tag_name
if ([string]::IsNullOrWhiteSpace($tag)) {
    throw "Couldn't determine the latest release tag."
}
$version = $tag.TrimStart("v")

$asset = "lazygit_${version}_windows_x86_64.zip"
$baseUrl = "https://github.com/$Repo/releases/download/$tag"

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("lazygit-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
try {
    $zipPath = Join-Path $tempDir $asset
    Write-Step "Downloading $tag ($asset)"
    Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $zipPath

    $checksumsPath = Join-Path $tempDir "checksums.txt"
    Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksumsPath

    $checksumLine = Get-Content -LiteralPath $checksumsPath | Where-Object { $_ -match [regex]::Escape($asset) }
    if ([string]::IsNullOrWhiteSpace($checksumLine)) {
        throw "checksums.txt does not include an entry for $asset."
    }
    $expectedSha256 = ($checksumLine -split '\s+')[0]

    $sha256 = [System.Security.Cryptography.SHA256]::Create()
    try {
        $bytes = [System.IO.File]::ReadAllBytes($zipPath)
        $actualSha256 = [System.BitConverter]::ToString($sha256.ComputeHash($bytes)).Replace("-", "").ToLowerInvariant()
    } finally {
        $sha256.Dispose()
    }
    if ($actualSha256 -ne $expectedSha256.ToLowerInvariant()) {
        throw "Downloaded lazygit checksum did not match. Expected $expectedSha256 but got $actualSha256."
    }

    $extractDir = Join-Path $tempDir "extracted"
    Expand-Archive -LiteralPath $zipPath -DestinationPath $extractDir

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -LiteralPath (Join-Path $extractDir $Bin) -Destination (Join-Path $InstallDir $Bin) -Force
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Step "Installed lazygit $version to $InstallDir\$Bin"

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = @()
if (-not [string]::IsNullOrWhiteSpace($userPath)) {
    $pathEntries = $userPath.Split(";", [System.StringSplitOptions]::RemoveEmptyEntries)
}
if (-not ($pathEntries | Where-Object { $_.TrimEnd("\") -ieq $InstallDir.TrimEnd("\") })) {
    $newUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
    Write-Step "Added $InstallDir to your User PATH. Open a new PowerShell window to use 'lazygit'."
} else {
    Write-Step "$InstallDir is already on your User PATH."
}

$env:Path = "$env:Path;$InstallDir"
Write-Host "lazygit $version installed successfully."
