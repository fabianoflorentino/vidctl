#!/usr/bin/env bash
set -euo pipefail

# Sincroniza a versão da última release nos links/scripts do site.
# Uso: site/sync-release.sh vX.Y.Z  (a partir de main)

VERSION="${1:?uso: site/sync-release.sh vX.Y.Z}"
[ "$VERSION" != "${VERSION#v}" ] || VERSION="v${VERSION}"
FILE="site/index.html"

for os in linux windows macos; do
  sed -i -E "s#(releases/tag/)v[0-9]+\.[0-9]+\.[0-9]+-${os}#\1${VERSION}-${os}#g" "$FILE"
done

# pill de versão "última release" (vX.Y.Z)
sed -i -E "s/>v[0-9]+\.[0-9]+\.[0-9]+</>${VERSION}</g" "$FILE"

grep -q "releases/tag/${VERSION}-linux" "$FILE"
echo "site sincronizado para ${VERSION}"