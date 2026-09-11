#!/usr/bin/env bash
# Atualiza pkg/aur/PKGBUILD + .SRCINFO para uma nova release (rode no Arch Linux).
# Uso: pkg/aur/update-aur.sh <versão>   (ex.: pkg/aur/update-aur.sh 1.0.7)
set -euo pipefail

VERSION="${1:?usage: $0 <versão>}"
PKG="pkg/aur/PKGBUILD"
[ -f "$PKG" ] || { echo "erro: $PKG não encontrado (rode do raiz do repo)" >&2; exit 1; }
[ -f LICENSE ] || { echo "erro: LICENSE não encontrado" >&2; exit 1; }

sed -i "s/^pkgver=.*/pkgver=${VERSION}/" "$PKG"

URL="https://github.com/fabianoflorentino/vidctl/archive/refs/tags/v${VERSION}.tar.gz"
SUM=$(curl -sL "$URL" | sha256sum | cut -d' ' -f1)
[ -n "$SUM" ] || { echo "erro: falha ao baixar tarball v${VERSION}" >&2; exit 1; }

sed -i "s/sha256sums=('[0-9a-f]*')/sha256sums=('${SUM}')/" "$PKG"
cp LICENSE pkg/aur/LICENSE

if command -v makepkg >/dev/null 2>&1; then
  (cd pkg/aur && makepkg --printsrcinfo > .SRCINFO)
  echo "pkg/aur/.SRCINFO regenerado"
else
  echo "aviso: makepkg não encontrado. Em Arch, rode 'cd pkg/aur && makepkg --printsrcinfo > .SRCINFO' antes de publicar." >&2
fi

echo "PKGBUILD atualizado para v${VERSION} (sha256sums ${SUM:0:12}…)"
echo "Próximo passo: pkg/aur/publish.sh"