#!/usr/bin/env bash
# Atualiza o PKGBUILD para uma nova versão (usa no AUR).
# Uso: pkg/aur/update-aur.sh 1.0.2
set -euo pipefail

VERSION="${1:?usage: $0 <version>}"
PKG="pkg/aur/PKGBUILD"
[ -f "$PKG" ] || { echo "erro: $PKG não encontrado (rode do raiz do repo)" >&2; exit 1; }

sed -i "s/^pkgver=.*/pkgver=${VERSION}/" "$PKG"

URL="https://github.com/fabianoflorentino/vidctl/archive/refs/tags/v${VERSION}.tar.gz"
SUM=$(curl -sL "$URL" | sha256sum | cut -d' ' -f1)
[ -n "$SUM" ] || { echo "erro: falha ao baixar tarball v${VERSION}" >&2; exit 1; }

python3 - "$SUM" <<'EOF'
import sys, re
path = "pkg/aur/PKGBUILD"
s = open(path).read()
s = re.sub(r"sha256sums=\('[0-9a-f]*'\)", "sha256sums=('" + sys.argv[1] + "')", s)
open(path, "w").write(s)
EOF

echo "PKGBUILD atualizado para v${VERSION} (sha256sums ok)"
echo "Para publicar no AUR: cd <repo-aur> && cp $PKG PKGBUILD && makepkg --printsrcinfo > .SRCINFO && git add -A && git commit -m \"bump ${VERSION}\" && git push"