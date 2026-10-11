#!/usr/bin/env bash
set -euo pipefail

# A versão não é mais exibida no CTA do site (o botão "baixar" aponta para a
# página de releases, onde a versão aparece). Mantido apenas como compat do
# workflow de release (site-sync.yml), que dispara este script por tag.
VERSION="${1:?uso: site/sync-release.sh vX.Y.Z}"
[ "$VERSION" != "${VERSION#v}" ] || VERSION="v${VERSION}"

echo "site sem versão embutida; nada a sincronizar para ${VERSION}"