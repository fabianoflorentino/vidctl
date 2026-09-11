#!/usr/bin/env bash
# Instala o vidctl no Arch de forma manual, estilo AUR — o mesmo fluxo que o
# yay usa por baixo dos panos: makepkg -s resolve/instala dependências e
# makepkg -i instala o pacote. Não requer conta no AUR.
#
# Uso (no Arch Linux, como usuário normal):
#   pkg/arch/install.sh
#
# Versão mais nova no GitHub? Atualize o PKGBUILD antes (na raiz do repo):
#   pkg/aur/update-aur.sh <versão>
set -euo pipefail

if ! command -v makepkg >/dev/null 2>&1; then
  echo "erro: makepkg não encontrado. Instale com:  sudo pacman -S --needed base-devel" >&2
  exit 1
fi
if [ "$(id -u)" -eq 0 ]; then
  echo "erro: makepkg não roda como root. Execute como usuário normal." >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKGDIR="$ROOT/pkg/aur"
[ -f "$PKGDIR/PKGBUILD" ] || { echo "erro: PKGBUILD não encontrado em $PKGDIR" >&2; exit 1; }

PKGVER=$(sed -n 's/^pkgver=\(.*\)/\1/p' "$PKGDIR/PKGBUILD")

if git -C "$ROOT" rev-parse --git-dir >/dev/null 2>&1; then
  LATEST=$(git ls-remote --tags origin 'v*' 2>/dev/null | sed 's#.*refs/tags/##' | grep -v -- '-' | sort -V | tail -1)
  if [ -n "$LATEST" ] && [ "${LATEST#v}" != "$PKGVER" ]; then
    echo "aviso: release mais nova no GitHub é ${LATEST}. Atualize o PKGBUILD na raiz do repo:" >&2
    echo "  pkg/aur/update-aur.sh ${LATEST#v}" >&2
  fi
fi

echo ">> Compilando vidctl ${PKGVER} (makepkg -si; pede sudo para instalar as dependências)"
cd "$PKGDIR"
makepkg -si
echo ">> vidctl ${PKGVER} instalado."