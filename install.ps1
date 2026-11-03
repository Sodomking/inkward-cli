param([string]$Repository = 'Sodomking/inkward-cli', [string]$Version = 'latest')
$ErrorActionPreference = 'Stop'
if ($Repository -notmatch '^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$') { throw 'Invalid repository' }
if ($Version -ne 'latest' -and $Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') { throw 'Invalid version' }
if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw 'This installer currently supports Windows x64 only.' }
$asset = 'inkward-win32-x64.exe'
$base = if ($Version -eq 'latest') { "https://github.com/$Repository/releases/latest/download" } else { "https://github.com/$Repository/releases/download/$Version" }
$target = Join-Path $env:LOCALAPPDATA 'INKWARD'
New-Item -ItemType Directory -Force -Path $target | Out-Null
$temporary = Join-Path $target ([guid]::NewGuid().ToString() + '.download')
try {
  Invoke-WebRequest -UseBasicParsing "$base/$asset" -OutFile $temporary
  $expected = ((Invoke-WebRequest -UseBasicParsing "$base/$asset.sha256").Content.Trim() -split '\s+')[0]
  if ($expected -notmatch '^[a-fA-F0-9]{64}$') { throw 'Invalid checksum file' }
  $actual = (Get-FileHash -Path $temporary -Algorithm SHA256).Hash
  if ($actual -ne $expected) { throw 'Checksum mismatch; downloaded file will not run.' }
  $exe = Join-Path $target 'inkward.exe'
  Move-Item -Force $temporary $exe
  Write-Host "Installed: $exe"
  & $exe info
  & $exe open
} finally { if (Test-Path $temporary) { Remove-Item $temporary } }
