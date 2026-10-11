#!/usr/bin/env bash
set -euo pipefail

# Sincroniza a versão da última release nos links/scripts do site.
# Uso: site/sync-release.sh vX.Y.Z  (a partir de main)

VERSION="${1:?uso: site/sync-release.sh vX.Y.Z}"
[ "$VERSION" != "${VERSION#v}" ] || VERSION="v${VERSION}"
FILE="site/index.html"

# "baixar vX.Y.Z" no CTA único (aponta para a página de releases)
sed -i -E "s/baixar v[0-9]+\.[0-9]+\.[0-9]+/baixar ${VERSION}/g" "$FILE"

grep -q "baixar ${VERSION}" "$FILE"
echo "site sincronizado para ${VERSION}"