# gopls-mcp installer for Windows
# Usage: irm https://raw.githubusercontent.com/[username]/gopls-mcp/main/scripts/install.ps1 | iex
# Or: PowerShell -ExecutionPolicy Bypass -File .\install.ps1

$ErrorActionPreference = "Stop"

$Repo = "xieyuschen/gopls-mcp"
$Name = "gopls-mcp"

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error-Exit {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

# Detect architecture
function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture
    switch ($arch) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default { Write-Error-Exit "Unsupported architecture: $arch" }
    }
}

# Get latest release version
function Get-LatestVersion {
    Write-Info "Fetching latest release version..."

    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $version = $response.tag_name

        if ([string]::IsNullOrEmpty($version)) {
            Write-Error-Exit "Failed to fetch latest version"
        }

        Write-Info "Latest version: $version"
        return $version
    }
    catch {
        Write-Error-Exit "Failed to fetch version: $_"
    }
}

# Determine install directory ($HOME/.local/bin)
function Get-InstallDir {
    $installDir = Join-Path $env:USERPROFILE ".local\bin"

    # Create directory if it doesn't exist
    if (!(Test-Path $installDir)) {
        Write-Info "Creating directory: $installDir"
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    Write-Info "Install directory: $installDir"
    return $installDir
}

# Download and install
function Download-AndInstall {
    param([string]$Version, [string]$InstallDir, [string]$Arch)

    $filename = "${Name}_${Version}_windows_${Arch}"
    $url = "https://github.com/${Repo}/releases/download/${Version}/${filename}.zip"
    $tempFile = Join-Path $env:TEMP "${filename}.zip"

    Write-Info "Downloading from: $url"

    try {
        Invoke-WebRequest -Uri $url -OutFile $tempFile -UseBasicParsing
    }
    catch {
        Write-Error-Exit "Failed to download binary: $_"
    }

    Write-Info "Extracting and installing..."

    # Create temp directory for extraction
    $tempExtract = Join-Path $env:TEMP "gopls-mcp-extract"
    if (Test-Path $tempExtract) {
        Remove-Item -Recurse -Force $tempExtract
    }
    New-Item -ItemType Directory -Path $tempExtract | Out-Null

    try {
        Expand-Archive -Path $tempFile -DestinationPath $tempExtract -Force

        # Move binary to install directory
        $binaryPath = Join-Path $tempExtract "${Name}.exe"
        if (!(Test-Path $binaryPath)) {
            Write-Error-Exit "Binary not found in archive: $binaryPath"
        }

        $destPath = Join-Path $InstallDir "${Name}.exe"
        Copy-Item -Path $binaryPath -Destination $destPath -Force

        Write-Info "Installed to: $destPath"
    }
    finally {
        # Cleanup
        Remove-Item -Force $tempFile -ErrorAction SilentlyContinue
        if (Test-Path $tempExtract) {
            Remove-Item -Recurse -Force $tempExtract -ErrorAction SilentlyContinue
        }
    }
}

# Verify installation
function Verify-Installation {
    param([string]$InstallDir)

    $binaryPath = Join-Path $InstallDir "${Name}.exe"

    if (Test-Path $binaryPath) {
        try {
            $output = & $binaryPath --version 2>&1
            Write-Info "Successfully installed $Name!"
            Write-Info $output
            Write-Info "Installation location: $binaryPath"
        }
        catch {
            Write-Warn "Installation completed, but could not verify version"
        }
    }
    else {
        Write-Warn "Installation completed, but binary not found at expected location"
    }
}

# Main execution
function Main {
    Write-Host ""
    Write-Host "gopls-mcp Installer for Windows"
    Write-Host "==============================="
    Write-Host ""

    $arch = Get-Architecture
    Write-Info "Detected Architecture: $arch"

    $version = Get-LatestVersion
    $installDir = Get-InstallDir
    Download-AndInstall -Version $version -InstallDir $installDir -Arch $arch
    Verify-Installation -InstallDir $installDir

    Write-Host ""
    Write-Info "Installation complete!"

    # Check if install directory is in PATH
    $pathEnv = $env:PATH -split ';'
    if ($installDir -notin $pathEnv) {
        Write-Warn "$installDir is not in your PATH"
        Write-Warn "Add it to your PATH:"
        Write-Warn "  [Environment]::SetEnvironmentVariable('Path', `$env:PATH + ';$installDir', 'User')"
        Write-Warn "Then restart your terminal"
    }
}

Main
