[CmdletBinding()]
param(
    [string]$InstallDir = $env:LAZYGIT_INSTALL_DIR
)

# Everything runs in its own script block: `irm | iex` executes in the
# caller's session, so strict mode and the error preference would otherwise
# stay switched on in the user's shell, and `exit` would close it.
& {
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
    throw "install.ps1 supports Windows only. Use install.sh on Linux or macOS."
}

# Only x86_64 is shipped; it also runs fine under emulation on ARM64 Windows.
$architecture = [System.Runtime.InteropServices.RuntimeInformation,mscorlib]::OSArchitecture.ToString()
if ($architecture -eq "Arm64") {
    Write-Step "Windows ARM64 detected; installing the x86_64 build under Windows emulation."
} elseif ($architecture -ne "X64") {
    throw "Unsupported Windows architecture: $architecture"
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

# Add the install folder to the User PATH without flattening it: read the
# raw registry value (so %VAR% entries stay unexpanded), keep its value
# type, and save the old value to a backup file before writing.
function Add-ToUserPath {
    param([string]$Dir)

    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment", $true)
    try {
        $raw = [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
        if ($key.GetValueNames() -contains "Path") {
            $kind = $key.GetValueKind("Path")
        }

        $entries = $raw.Split(";", [System.StringSplitOptions]::RemoveEmptyEntries)
        $already = $entries | Where-Object {
            [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\") -ieq $Dir.TrimEnd("\")
        }
        if ($already) {
            Write-Step "$Dir is already on your User PATH."
            return
        }

        $backupDir = Join-Path $env:LOCALAPPDATA "lazygit"
        New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
        $backup = Join-Path $backupDir ("path-backup-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".txt")
        Set-Content -LiteralPath $backup -Value $raw -Encoding UTF8
        Write-Step "Saved your previous User PATH to $backup"

        $new = if ([string]::IsNullOrWhiteSpace($raw)) { $Dir } else { $raw.TrimEnd(";") + ";" + $Dir }
        $key.SetValue("Path", $new, $kind)
    } finally {
        $key.Close()
    }

    # Tell running programs (Explorer, new terminals) that the environment changed.
    if (-not ("LazygitInstall.Env" -as [type])) {
        Add-Type -Namespace "LazygitInstall" -Name "Env" -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
    }
    $result = [UIntPtr]::Zero
    [void][LazygitInstall.Env]::SendMessageTimeout([IntPtr]0xffff, 0x1A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result)
    Write-Step "Added $Dir to your User PATH. Open a new PowerShell window to use 'lazygit'."
}

Add-ToUserPath -Dir $InstallDir
$env:Path = "$env:Path;$InstallDir"
Write-Host "lazygit $version installed successfully."
}
