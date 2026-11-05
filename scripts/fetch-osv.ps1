# Downloads OSV vulnerability dumps for offline use with `traverse scan --vulndb`.
# Run it on a machine that is allowed to reach the Internet; copy the folder to
# where Traverse runs. Traverse itself never connects to the network.
# If script execution is not allowed, download the same URLs with a browser:
#   https://osv-vulnerabilities.storage.googleapis.com/<Ecosystem>/all.zip
# Usage: .\scripts\fetch-osv.ps1 [-Dest osv\2026-11-03] [-Ecosystems Maven,npm]
param(
  [string]$Dest = ("osv\" + (Get-Date -Format "yyyy-MM-dd")),
  [string[]]$Ecosystems = @("Maven", "npm", "NuGet", "PyPI", "Packagist", "Go")
)
$ErrorActionPreference = "Stop"
$base = "https://osv-vulnerabilities.storage.googleapis.com"
New-Item -ItemType Directory -Force -Path $Dest | Out-Null
foreach ($e in $Ecosystems) {
  Write-Host "Downloading $e ..."
  Invoke-WebRequest -UseBasicParsing -Uri "$base/$e/all.zip" -OutFile (Join-Path $Dest "$e-all.zip")
}
Get-ChildItem -Path $Dest -Filter *.zip | Get-FileHash -Algorithm SHA256 |
  ForEach-Object { "{0}  {1}" -f $_.Hash.ToLower(), (Split-Path $_.Path -Leaf) } |
  Tee-Object -FilePath (Join-Path $Dest "SHA256SUMS")
Write-Host "Done. Use: traverse vulndb info $Dest ; traverse scan --vulndb $Dest ..."
