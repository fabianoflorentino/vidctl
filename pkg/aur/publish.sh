#!/usr/bin/env bash
# Publica/atualiza o pacote vidctl no AUR, seguindo as AUR submission guidelines:
#   https://wiki.archlinux.org/title/AUR_submission_guidelines
# Pré-requisitos:
#   - Conta no AUR (https://aur.archlinux.org) com chave SSH pública no perfil.
#   - SSH configurado para aur.archlinux.org (ex.: ~/.ssh/config:
#       Host aur.archlinux.org
#         IdentityFile ~/.ssh/aur
#         User aur)
#   - makepkg (Arch Linux / base-devel).
# Uso: pkg/aur/publish.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DEST="${AUR_DIR:-/tmp/aur-vidctl}"

if ! command -v makepkg >/dev/null 2>&1; then
  echo "erro: makepkg não encontrado. Rode este script numa máquina Arch Linux (base-devel)." >&2
  exit 1
fi
if command -v ssh >/dev/null 2>&1 && ! ssh -q -o BatchMode=yes aur@aur.archlinux.org help >/dev/null 2>&1; then
  echo "aviso: não consegui autenticar no AUR via SSH (chave configurada?)." >&2
fi

# 1. Clona o repositório vazio do pkgbase (criação do pacote quando não existe).
if [ ! -d "$DEST/.git" ]; then
  echo ">> Clonando repositório do AUR (repo vazio, esperado o warning 'empty repository')"
  mkdir -p "$(dirname "$DEST")"
  git -c init.defaultBranch=master clone ssh://aur@aur.archlinux.org/vidctl.git "$DEST"
fi

# 2. Copia PKGBUILD + LICENSE e regenera .SRCINFO.
cp "$ROOT/pkg/aur/PKGBUILD" "$DEST/PKGBUILD"
cp "$ROOT/pkg/aur/LICENSE" "$DEST/LICENSE"
(cd "$DEST" && makepkg --printsrcinfo > .SRCINFO)

# 3. Commit temático + push (o AUR só aceita a branch master).
MSG="bump $(sed -n 's/^pkgver=\(.*\)/\1/p' "$DEST/PKGBUILD")"
(cd "$DEST" && {
  git add PKGBUILD .SRCINFO LICENSE
  if git diff --cached --quiet; then
    echo ">> Nada a publicar (repo AUR já atualizado)"
  else
    git commit -m "$MSG"
  fi
  git push
})
echo ">> Publicado em https://aur.archlinux.org/packages/vidctl"