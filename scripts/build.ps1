<#
.SYNOPSIS
  Cross-compiles Traverse for all supported platforms into dist\ with SHA-256 checksums.
  Requires only Go (https://go.dev/dl/). Works with Windows PowerShell 5.1 and PowerShell 7.
.EXAMPLE
  .\scripts\build.ps1 -Version 0.1.0
#>
param([string]$Version = 'dev')
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
if (Test-Path dist) { Remove-Item -Recurse -Force dist }
New-Item -ItemType Directory dist | Out-Null
$targets = 'windows/amd64','windows/arm64','linux/amd64','linux/arm64','darwin/amd64','darwin/arm64'
foreach ($t in $targets) {
  $os, $arch = $t.Split('/')
  $ext = if ($os -eq 'windows') { '.exe' } else { '' }
  $out = "dist/traverse-$Version-$os-$arch$ext"
  Write-Host "building $out"
  $env:CGO_ENABLED = '0'; $env:GOOS = $os; $env:GOARCH = $arch
  go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=$Version" -o $out ./cmd/traverse
  if ($LASTEXITCODE -ne 0) { throw "build failed for $t" }
}
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
Get-ChildItem dist -Filter 'traverse-*' | ForEach-Object {
  '{0}  {1}' -f (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower(), $_.Name
} | Set-Content -Encoding ascii dist/SHA256SUMS.txt
Get-Content dist/SHA256SUMS.txt
