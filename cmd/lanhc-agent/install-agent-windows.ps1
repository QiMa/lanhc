# Install lanhc-agent on a Windows lanhc node.
# Usage (PowerShell as Administrator):
#   powershell -ExecutionPolicy Bypass -File .\install-lanhc-agent.ps1
#
# First enrollment: set TS_AUTHKEY before running, or pass -AuthKey.
param(
    [string]$AuthKey = $env:TS_AUTHKEY,
    [string]$HostName = $env:LANHC_AGENT_HOSTNAME,
    [string]$InstallDir = "$env:ProgramFiles\LanhcAgent",
    [string]$StateDir = "$env:ProgramData\LanhcAgent"
)

$ErrorActionPreference = 'Stop'
$bin = Join-Path $PSScriptRoot 'lanhc-agent.exe'
if (-not (Test-Path $bin)) { throw "lanhc-agent.exe not found next to this script: $bin" }

$current = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($current)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this script from an elevated PowerShell (Run as Administrator).'
}

New-Item -ItemType Directory -Force -Path $InstallDir, $StateDir | Out-Null
Copy-Item -Force $bin (Join-Path $InstallDir 'lanhc-agent.exe')

$agentHost = if ($HostName) { $HostName } else { "$env:COMPUTERNAME-agent" }
$taskName = 'LanhcAgent'

# Remove any previous registration without relying on schtasks.exe. This also
# avoids the misleading "system cannot find the file" error on a first install.
if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
}

# Quote arguments that contain spaces so the scheduled task keeps multi-word
# paths (Program Files / ProgramData) intact.
$taskArgs = @('-hostname', $agentHost, '-dir', $StateDir, '-listen', ':8088')
if ($AuthKey) { $taskArgs += @('-auth-key', $AuthKey) }
$quotedArgs = $taskArgs | ForEach-Object {
    if ($_ -match '\s') { '"' + $_ + '"' } else { $_ }
}
$argString = $quotedArgs -join ' '

$action = New-ScheduledTaskAction -Execute (Join-Path $InstallDir 'lanhc-agent.exe') -Argument $argString
$trigger = New-ScheduledTaskTrigger -AtLogOn
$svcPrincipal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -LogonType ServiceAccount -RunLevel Highest
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $svcPrincipal -Force | Out-Null
Start-ScheduledTask -TaskName $taskName

Write-Host "installed: $(Join-Path $InstallDir 'lanhc-agent.exe')"
Write-Host "scheduled task: $taskName (ONLOGON, SYSTEM, HIGHEST)"
Write-Host "state dir: $StateDir"
Write-Host "node name: $agentHost"
