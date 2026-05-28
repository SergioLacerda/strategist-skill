# Strategist curl installer — Windows PowerShell
#
# Wizard runs by default. Use -Silent to skip interactive setup.
#
# Usage:
#   irm https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.ps1 | iex
#   Invoke-WebRequest -Uri https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.ps1 -OutFile bootstrap.ps1
#   .\bootstrap.ps1              # wizard (default)
#   .\bootstrap.ps1 -Silent      # skip wizard
#   .\bootstrap.ps1 -Target C:\my\project
#   .\bootstrap.ps1 -Ref v1.0.0

[CmdletBinding()]
param(
    [switch]$Silent,
    [string]$Target = "",
    [string]$Ref = ""
)

$ErrorActionPreference = "Stop"

$REPO = "SergioLacerda/strategist-skill"
$DEFAULT_REF = "main"

# ── resolve version ───────────────────────────────────────────────────────────

function Resolve-Ref {
    if ($Ref -ne "") { return $Ref }

    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -ErrorAction Stop
        if ($release.tag_name) { return $release.tag_name }
    } catch {
        # no releases yet
    }

    Write-Host "[Strategist] No release found, using branch: $DEFAULT_REF"
    return $DEFAULT_REF
}

# ── detect bash runtime ───────────────────────────────────────────────────────

function Find-Bash {
    # 1. Explicit override via env var
    if ($env:GIT_BASH -and (Test-Path $env:GIT_BASH)) { return $env:GIT_BASH }

    # 2. Git Bash standard locations
    $gitBashPaths = @(
        "C:\Program Files\Git\bin\bash.exe",
        "C:\Program Files (x86)\Git\bin\bash.exe"
    )
    foreach ($p in $gitBashPaths) {
        if (Test-Path $p) { return $p }
    }

    # 3. WSL
    $wsl = Get-Command wsl.exe -ErrorAction SilentlyContinue
    if ($wsl) { return "wsl" }

    return $null
}

# ── main ──────────────────────────────────────────────────────────────────────

$resolvedRef = Resolve-Ref

if ($resolvedRef -like "v*") {
    $archiveUrl = "https://github.com/$REPO/archive/refs/tags/$resolvedRef.zip"
} else {
    $archiveUrl = "https://github.com/$REPO/archive/refs/heads/$resolvedRef.zip"
}

$tmpDir = Join-Path $env:TEMP "strategist-install-$PID"
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    Write-Host "[Strategist] Downloading from $archiveUrl ..."
    $zipPath = Join-Path $tmpDir "strategist.zip"
    Invoke-WebRequest -Uri $archiveUrl -OutFile $zipPath

    $extractDir = Join-Path $tmpDir "extracted"
    Expand-Archive -Path $zipPath -DestinationPath $extractDir

    # GitHub zip has a top-level directory like "strategist-skill-1.0.0\"
    $inner = Get-ChildItem $extractDir -Directory | Select-Object -First 1
    if (-not $inner) {
        Write-Error "Unexpected archive structure: no top-level directory found."
        exit 1
    }

    $installScript = Join-Path $inner.FullName "strategist\install.sh"
    if (-not (Test-Path $installScript)) {
        Write-Error "install.sh not found in downloaded archive."
        exit 1
    }

    # Build install.sh args — wizard is default, -Silent opts out
    $installArgs = @("--wizard")
    if ($Silent)  { $installArgs = @() }
    if ($Target)  { $installArgs += "--target=$Target" }

    # Find bash and execute
    $bash = Find-Bash
    if (-not $bash) {
        Write-Error @"
No bash runtime detected. Strategist requires bash to run install.sh.

Options:
  - Install Git for Windows: https://git-scm.com/download/win
  - Enable WSL: https://learn.microsoft.com/en-us/windows/wsl/install
  - Set `$env:GIT_BASH to point to your bash.exe and re-run.
"@
        exit 1
    }

    # Convert Windows path to Unix path for bash
    $installScriptUnix = $installScript -replace '\\', '/' -replace '^([A-Za-z]):', '/\1'

    Write-Host "[Strategist] Using bash: $bash"

    if ($bash -eq "wsl") {
        $wslPath = wsl wslpath -u $installScript.Replace('\', '\\')
        if ($installArgs.Count -gt 0) {
            wsl bash $wslPath @installArgs
        } else {
            wsl bash $wslPath
        }
    } else {
        if ($installArgs.Count -gt 0) {
            & $bash $installScriptUnix @installArgs
        } else {
            & $bash $installScriptUnix
        }
    }

} finally {
    Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
