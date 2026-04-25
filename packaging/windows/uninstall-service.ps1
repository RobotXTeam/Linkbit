$ErrorActionPreference = "Stop"

if (Get-Service -Name "LinkbitAgent" -ErrorAction SilentlyContinue) {
  sc.exe stop LinkbitAgent | Out-Null
  sc.exe delete LinkbitAgent | Out-Null
}

Remove-Item -Force "$env:ProgramFiles\Linkbit\linkbit-agent.exe" -ErrorAction SilentlyContinue
