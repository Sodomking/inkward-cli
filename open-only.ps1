# Tiny browser entry: no Node, download, system proxy changes or admin permission.
param(
  [ValidateSet('', 'english1', 'english2', 'postgraduate', 'cet4', 'cet6', 'ielts', 'ielts-academic', 'ielts-general')][string]$Exam = '',
  [ValidateSet('', 'big', 'small')][string]$Trial = '',
  [switch]$Yes, [switch]$PrintUrl
)
$ErrorActionPreference = 'Stop'
if ($Exam -eq 'english2') { $Exam = 'postgraduate' }
if ($Exam -eq 'ielts-academic') { $Exam = 'ielts' }
$route = '/home'
if ($Exam) { $route = "/exams?exam=$Exam" }
if ($Trial) {
  if (-not $Exam -or $Exam -eq 'english1') { $topic = "english1-$Trial-2026" }
  elseif ($Exam -eq 'postgraduate') {
    $topic = if ($Trial -eq 'big') { 'postgraduate-english2-2026' } else { 'postgraduate-english2-small-2026' }
  } else { throw 'Trials are available for english1/postgraduate only.' }
  $route = "/brief?topic=$topic"
}
$url = "https://rimbaud.asia/writing/#$route"
Write-Output $url
if ($PrintUrl) { exit 0 }
if (-not $Yes) { if ((Read-Host 'Open the website? [y/N]') -notmatch '^(y|yes)$') { exit 0 } }
Start-Process -FilePath $url
Write-Output 'Browser requested. Login and payment remain in the website.'
