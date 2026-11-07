# INKWARD native bootstrap. No Node, Git, Go, elevation or policy changes.
param(
  [switch]$Yes, [switch]$Plan, [switch]$DryRun, [switch]$NoOpen,
  [ValidateSet('', 'english1', 'english2', 'postgraduate', 'cet4', 'cet6', 'ielts', 'ielts-academic', 'ielts-general')][string]$Exam = '',
  [ValidateSet('', 'big', 'small')][string]$Trial = ''
)
$ErrorActionPreference = 'Stop'
$version = '2.0.0'
if ($Trial -and $Exam -notin @('', 'english1', 'english2', 'postgraduate')) { throw 'Trial requires English I or II' }
$machine = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
# Read native architecture even when the agent uses an emulated x64 process.
$native = (Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment' -ErrorAction SilentlyContinue).PROCESSOR_ARCHITECTURE
if ($native) { $machine = $native }
$arch = switch ($machine.ToUpperInvariant()) { 'AMD64' { 'amd64' }; 'ARM64' { 'arm64' }; default { throw 'Windows x64 or ARM64 is required' } }
$asset = "inkward-windows-$arch.exe"
$base = "https://github.com/Sodomking/inkward-cli/releases/download/v$version"
if ($Plan) { @{ version = $version; os = 'windows'; arch = $arch; asset = $asset; url = "$base/$asset" } | ConvertTo-Json -Compress; exit 0 }
if (-not $Yes -and (Read-Host 'Configure INKWARD direct access and open the website? [y/N]') -notmatch '^(y|yes)$') { exit 1 }
$dest = Join-Path $env:LOCALAPPDATA "INKWARD\bin\$version"
if ((Test-Path $dest) -and ((Get-Item $dest).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Install directory must not be a link' }
New-Item -ItemType Directory -Path $dest -Force | Out-Null
$temp = Join-Path ([IO.Path]::GetTempPath()) ('inkward-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $temp | Out-Null
$exitCode = 1
try {
  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
  $manifest = Join-Path $temp 'SHA256SUMS'
  Invoke-WebRequest -UseBasicParsing -Uri "$base/SHA256SUMS" -OutFile $manifest -TimeoutSec 120
  $matchesFound = @(Get-Content $manifest | Where-Object { $_ -match ('^[a-f0-9]{64}  ' + [regex]::Escape($asset) + '$') })
  if ($matchesFound.Count -ne 1) { throw 'Missing or invalid checksum' }
  $expected = $matchesFound[0].Substring(0,64)
  $binary = Join-Path $dest 'inkward.exe'
  $actual = ''
  if ((Test-Path $binary) -and -not ((Get-Item $binary).Attributes -band [IO.FileAttributes]::ReparsePoint)) { $actual = (Get-FileHash $binary -Algorithm SHA256).Hash.ToLowerInvariant() }
  if ($actual -ne $expected) {
    $download = Join-Path $temp 'inkward.exe'
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $download -TimeoutSec 120
    if ((Get-FileHash $download -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) { throw 'Checksum mismatch; not executed' }
    Move-Item -LiteralPath $download -Destination $binary -Force
  }
  $arguments = @('setup','--yes','--json')
  if ($DryRun) { $arguments += '--dry-run' }
  if ($NoOpen) { $arguments += '--no-open' }
  if ($Exam) { $arguments += @('--exam', $Exam) }
  if ($Trial) { $arguments += @('--trial', $Trial) }
  & $binary @arguments
  $exitCode = $LASTEXITCODE
} finally { Remove-Item -LiteralPath $temp -Recurse -Force }
exit $exitCode
