# Hubble Network Installer Download and Run Script for Windows
# Usage: 
#   With credentials: iex "& { $(irm https://hubble.com/install.ps1) } <base64-credentials>"
#   Without credentials: iex "& { $(irm https://hubble.com/install.ps1) }"

param(
    [string]$Credentials = ""
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Accept credentials as parameter (base64 encoded org_id:api_key)
if ($Credentials) {
    $ValidationFailed = $false
    
    try {
        # Validate base64 format and decode
        $DecodedBytes = [System.Convert]::FromBase64String($Credentials)
        $DecodedString = [System.Text.Encoding]::UTF8.GetString($DecodedBytes)
        
        # Validate format (should contain a colon)
        if (-not $DecodedString.Contains(':')) {
            $ValidationFailed = $true
        }
    } catch {
        $ValidationFailed = $true
    }
    
    if ($ValidationFailed) {
        Write-Host ""
        Write-Host "⚠️  We were unable to validate your credentials." -ForegroundColor Yellow
        Write-Host ""
        Write-Host "You can either:"
        Write-Host "  • Exit and check that you pasted the complete command correctly"
        Write-Host "  • Continue and enter your credentials manually"
        Write-Host ""
        $Response = Read-Host "Would you like to exit and try again? (Y/n)"
        if ([string]::IsNullOrEmpty($Response) -or $Response -match '^[Yy]') {
            Write-Host "Please check your command and run the installer again."
            exit 1
        }
        Write-Host "Continuing - you'll be prompted for credentials..."
        Write-Host ""
    } else {
        $env:HUBBLE_CREDENTIALS = $Credentials
        Write-Host "✓ Credentials provided" -ForegroundColor Green
    }
}

$InstallUrl = "https://hubble-install.s3.amazonaws.com"
$BinaryName = "hubble-install-windows-amd64.exe"

Write-Host "🛰️  Hubble Network Installer" -ForegroundColor Cyan
Write-Host "=============================="
Write-Host ""

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Currently only supporting amd64
if ($Arch -ne "amd64") {
    Write-Host "❌ Error: Only 64-bit Windows is supported" -ForegroundColor Red
    exit 1
}

$DownloadUrl = "$InstallUrl/$BinaryName"

Write-Host "✓ Detected platform: Windows/$Arch" -ForegroundColor Green
Write-Host "📥 Downloading installer..." -ForegroundColor Cyan
Write-Host ""

# Download the binary to temp location
$TempFile = [System.IO.Path]::GetTempFileName() + ".exe"

try {
    # Use TLS 1.2 for secure connection
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    
    # Download with progress
    $ProgressPreference = 'SilentlyContinue'
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempFile -UseBasicParsing
    $ProgressPreference = 'Continue'
    
    Write-Host "✓ Download complete!" -ForegroundColor Green
    Write-Host "🚀 Running installer..." -ForegroundColor Cyan
    Write-Host ""
    
    # Check if running as administrator
    $IsAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $IsAdmin) {
        Write-Host "⚠️  Administrator privileges required" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Attempting to restart with administrator privileges..."
        Write-Host "Please accept the UAC prompt to continue."
        Write-Host ""
        
        # Restart with admin privileges
        $Arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$TempFile`" --debug"
        Start-Process -FilePath $TempFile -ArgumentList "--debug" -Verb RunAs -Wait
    } else {
        # Run the installer directly
        & $TempFile --debug
    }
    
} catch {
    Write-Host "❌ Download or execution failed: $_" -ForegroundColor Red
    exit 1
} finally {
    # Clean up
    if (Test-Path $TempFile) {
        Remove-Item $TempFile -Force -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
