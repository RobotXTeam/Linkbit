param(
  [string]$InstallDir = "$env:ProgramFiles\Linkbit"
)

$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item -Force ".\bin\linkbit-agent.exe" "$InstallDir\linkbit-agent.exe"

if (Get-Service -Name "LinkbitAgent" -ErrorAction SilentlyContinue) {
  sc.exe stop LinkbitAgent | Out-Null
  sc.exe delete LinkbitAgent | Out-Null
}

sc.exe create LinkbitAgent binPath= "`"$InstallDir\linkbit-agent.exe`"" start= demand DisplayName= "Linkbit Agent" | Out-Null
Write-Host "LinkbitAgent installed. Configure environment securely before starting the service."
