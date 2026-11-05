#!/usr/bin/env sh
# Downloads OSV vulnerability dumps for offline use with `traverse scan --vulndb`.
# Run it on a machine that is allowed to reach the Internet; copy the folder to
# where Traverse runs. Traverse itself never connects to the network.
# Usage: scripts/fetch-osv.sh [DEST_DIR] [ECOSYSTEM...]
#   default ecosystems: Maven npm NuGet PyPI Packagist Go
set -eu
DEST="${1:-osv/$(date +%Y-%m-%d)}"
[ $# -gt 0 ] && shift
ECOS="${*:-Maven npm NuGet PyPI Packagist Go}"
BASE="https://osv-vulnerabilities.storage.googleapis.com"
mkdir -p "$DEST"
for e in $ECOS; do
  echo "Downloading $e ..."
  curl -fSL --retry 3 -o "$DEST/$e-all.zip" "$BASE/$e/all.zip"
done
( cd "$DEST" && if command -v sha256sum >/dev/null; then sha256sum ./*.zip; else shasum -a 256 ./*.zip; fi ) > "$DEST/SHA256SUMS"
cat "$DEST/SHA256SUMS"
echo "Done. Use: traverse vulndb info $DEST && traverse scan --vulndb $DEST ..."
