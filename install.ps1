# Compatibility alias. Prefer START.md; no separate installer workflow.
param([switch]$Yes, [switch]$Plan, [switch]$DryRun, [switch]$NoOpen)
$ErrorActionPreference = 'Stop'
$entry = Join-Path $PSScriptRoot 'start.ps1'
if (-not (Test-Path -LiteralPath $entry)) { throw 'Download start.ps1 using https://github.com/Sodomking/inkward-cli/blob/main/START.md' }
& $entry @PSBoundParameters
exit $LASTEXITCODE
