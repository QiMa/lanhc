# Install lanhc-agent on a Windows lanhc node.
# Usage (PowerShell as Administrator):
#   powershell -ExecutionPolicy Bypass -File install-lanhc-agent.ps1
# Environment for first enrollment is optional; use -AuthKey to pass one.
param(
    [string]$AuthKey = $env:TS_AUTHKEY,
    [string]$HostName = $env:LANHC_AGENT_HOSTNAME,
    [string]$InstallDir = "$env:ProgramFiles\LanhcAgent",
    [string]$StateDir = "$env:ProgramData\LanhcAgent"
)

$ErrorActionPreference = 'Stop'
$bin = Join-Path $PSScriptRoot 'lanhc-agent.exe'
if (-not (Test-Path $bin)) { throw "lanhc-agent.exe not found next to this script: $bin" }

New-Item -ItemType Directory -Force -Path $InstallDir, $StateDir | Out-Null
Copy-Item -Force $bin (Join-Path $InstallDir 'lanhc-agent.exe')

$args = @('-hostname', $(if ($HostName) { $HostName } else { "$env:COMPUTERNAME-agent" }),
          '-dir', $StateDir,
          '-listen', ':8088')
if ($AuthKey) { $args += @('-auth-key', $AuthKey) }

# Remove any previous scheduled task, then register a logon-started task that
# keeps the agent running with the current host identity.
$taskName = 'LanhcAgent'
schtasks.exe /Delete /TN $taskName /F 2>$null | Out-Null
schtasks.exe /Create /TN $taskName /TR ("`"$(Join-Path $InstallDir 'lanhc-agent.exe')`" " + ($args -join ' ')) /SC ONLOGON /RU SYSTEM /RL HIGHEST /F | Out-Null
schtasks.exe /Run /TN $taskName | Out-Null
Write-Host "installed: $(Join-Path $InstallDir 'lanhc-agent.exe')"
Write-Host "scheduled task: $taskName (ONLOGON, SYSTEM)"
Write-Host "state dir: $StateDir"
