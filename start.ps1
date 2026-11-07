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
  $script:endpoints = @{}
  $script:apiLoaded = $false
  function Get-ReleaseAsset([string]$Name, [string]$Target) {
    # The direct exception is scoped to this single first-party request.
    $response = $null; $stream = $null; $file = $null
    try {
      $request = [Net.HttpWebRequest]::Create("https://static.rimbaud.asia/releases/eb87c3e97e94dbcb/cli/v2.0.0/$Name")
      $request.Proxy = $null; $request.Timeout = 15000; $request.ReadWriteTimeout = 30000; $request.AllowAutoRedirect = $false
      $response = $request.GetResponse()
      if ([int]$response.StatusCode -ne 200) { throw 'Unexpected CDN response' }
      $stream = $response.GetResponseStream(); $file = [IO.File]::Create($Target)
      $buffer = New-Object byte[] 65536; $total = 0
      while (($count = $stream.Read($buffer,0,$buffer.Length)) -gt 0) {
        $total += $count; if ($total -gt 20971520) { throw 'Unexpected download size' }
        $file.Write($buffer,0,$count)
      }
      return
    } catch { Write-Warning 'Site CDN unavailable; trying the official GitHub Release.' }
    finally { if ($file) { $file.Dispose() }; if ($stream) { $stream.Dispose() }; if ($response) { $response.Dispose() } }
    if (-not $script:apiLoaded) {
      $script:apiLoaded = $true
      try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/Sodomking/inkward-cli/releases/tags/v$version" -TimeoutSec 20
        foreach ($item in $release.assets) {
          if ($item.name -in @($asset, 'SHA256SUMS')) {
            if ($item.url -notmatch '^https://api\.github\.com/repos/Sodomking/inkward-cli/releases/assets/[0-9]+$') { throw 'Invalid official asset endpoint' }
            $script:endpoints[$item.name] = $item.url
          }
        }
      } catch { Write-Warning 'Official API unavailable; trying the standard Release download.'; $script:endpoints = @{} }
    }
    if ($script:endpoints.ContainsKey($Name)) {
      try {
        Invoke-WebRequest -UseBasicParsing -Uri $script:endpoints[$Name] -Headers @{ Accept = 'application/octet-stream' } -OutFile $Target -TimeoutSec 120
        return
      } catch { Write-Warning 'Official asset API unavailable; trying the standard download.' }
    }
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$Name" -OutFile $Target -TimeoutSec 120
  }
  $manifest = Join-Path $temp 'SHA256SUMS'
  Get-ReleaseAsset 'SHA256SUMS' $manifest
  if ((Get-FileHash $manifest -Algorithm SHA256).Hash.ToLowerInvariant() -ne 'eb87c3e97e94dbcb3e11d3efbb62e027354af024a90b69280fa6c6cb42eaa0d5') { throw 'Release manifest mismatch' }
  $matchesFound = @(Get-Content $manifest | Where-Object { $_ -match ('^[a-f0-9]{64}  ' + [regex]::Escape($asset) + '$') })
  if ($matchesFound.Count -ne 1) { throw 'Missing or invalid checksum' }
  $expected = $matchesFound[0].Substring(0,64)
  $binary = Join-Path $dest 'inkward.exe'
  $actual = ''
  if ((Test-Path $binary) -and -not ((Get-Item $binary).Attributes -band [IO.FileAttributes]::ReparsePoint)) { $actual = (Get-FileHash $binary -Algorithm SHA256).Hash.ToLowerInvariant() }
  if ($actual -ne $expected) {
    $download = Join-Path $temp 'inkward.exe'
    Get-ReleaseAsset $asset $download
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
