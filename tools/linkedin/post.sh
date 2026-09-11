#!/usr/bin/env bash
# Publica posts no LinkedIn via API (Posts API v2 — POST /rest/posts),
# sem abrir a UI do LinkedIn depois do login inicial.
#
# Pré-requisitos (uma vez):
#   - App em https://developer.linkedin.com com o produto "Share on LinkedIn"
#     (escopo w_member_social).
#   - Variáveis de ambiente:
#       LINKEDIN_CLIENT_ID      (id do app)
#       LINKEDIN_CLIENT_SECRET  (segredo do app)
#       LINKEDIN_REDIRECT_URI   (URL de callback EXATAMENTE como cadastrada no app)
#   - curl e python3 instalados.
#
# Uso:
#   tools/linkedin/post.sh login                           # autoriza 1x no navegador
#   tools/linkedin/post.sh whoami                          # mostra o seu person URN
#   tools/linkedin/post.sh publish <arquivo.md> [opções]
#     --title "..."     título do card (se --source for dado)
#     --description ".." descrição do card
#     --source URL      link do card (sem passar -> post só de texto)
#
# O rascunho do post fica em tools/linkedin/posts/apresentacao.md.
set -euo pipefail

CONF="${LINKEDIN_CONFIG:-$HOME/.config/vidctl/linkedin.json}"
URI_BASE="https://www.linkedin.com"
API="https://api.linkedin.com"
LINKEDIN_VERSION="$(date +%Y%m)"

die() { echo "erro: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "$1 não encontrado (instale)."; }
need curl; need python3
need_env() { [ -n "${!1:-}" ] || die "defina $1"; }

uri_enc() { python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=""))' "$1"; }

cfg_set() {
  mkdir -p "$(dirname "$CONF")"
  python3 -c 'import json,os,sys
p,k,v=sys.argv[1:4]
d={}
if os.path.exists(p) and os.path.getsize(p)>0:
    d=json.load(open(p))
d[k]=v
json.dump(d, open(p,"w"))' "$CONF" "$1" "$2"
  chmod 600 "$CONF"
}
cfg_get() {
  [ -f "$CONF" ] || return 0
  python3 -c 'import json,sys
d=json.load(open(sys.argv[1]))
print(d.get(sys.argv[2], ""))' "$CONF" "$1"
}
resp_json() { python3 -c 'import json,sys
try:
    d=json.load(sys.stdin)
except Exception:
    sys.exit(1)
sys.stdout.write(d.get(sys.argv[1], ""))' "$1"; }
has_token() { python3 -c 'import json,sys
try:
    d=json.load(sys.stdin)
except Exception:
    sys.exit(1)
sys.exit(0 if d.get("access_token") else 1)'; }

access_token() {
  local t exp
  t=$(cfg_get access_token); exp=$(cfg_get access_token_expiry)
  if [ -n "$exp" ] && [ "$(date +%s)" -ge "$exp" ]; then
    echo ">> access token expirado; renovando..." >&2
    refresh_token
    t=$(cfg_get access_token)
  fi
  [ -n "$t" ] || die "sem token. Rode: $0 login"
  echo "$t"
}

login() {
  need_env LINKEDIN_CLIENT_ID; need_env LINKEDIN_CLIENT_SECRET; need_env LINKEDIN_REDIRECT_URI
  local state verifier challenge code cb st
  state=$(openssl rand -hex 16 2>/dev/null || echo "sv$(date +%s)")
  verifier=$(openssl rand -base64 48 2>/dev/null | tr -d '=+/' | head -c 64)
  if command -v openssl >/dev/null 2>&1; then
    challenge=$(printf '%s' "$verifier" | openssl dgst -sha256 -binary 2>/dev/null | base64 | tr '+/' '-_' | tr -d '=')
  else
    challenge=""
  fi

  qs="response_type=code&client_id=$LINKEDIN_CLIENT_ID&redirect_uri=$(uri_enc "$LINKEDIN_REDIRECT_URI")&state=$state&scope=w_member_social"
  [ -n "$challenge" ] && qs="$qs&code_challenge=$challenge&code_challenge_method=S256"

  echo ">> Abra no navegador e autorize:"
  echo "$URI_BASE/oauth/v2/authorization?$qs"
  echo
  read -r -p ">> Depois de autorizar, o navegador cai no seu callback com ?code=.... Cole aqui a URL inteira que apareceu: " cb
  code=$(printf '%s' "$cb" | grep -oE 'code=[^&]+' | cut -d= -f2-)
  [ -n "$code" ] || die "não achei 'code=' na URL colada."
  st=$(printf '%s' "$cb" | grep -oE 'state=[^&]+' | cut -d= -f2-)
  [ -n "$state" ] && [ "$st" != "$state" ] && echo "aviso: state não confere com o da requisição." >&2

  local url base resp
  url="$URI_BASE/oauth/v2/accessToken"
  base="grant_type=authorization_code&code=$code&redirect_uri=$(uri_enc "$LINKEDIN_REDIRECT_URI")&client_id=$LINKEDIN_CLIENT_ID&client_secret=$LINKEDIN_CLIENT_SECRET"
  if [ -n "$verifier" ]; then
    resp=$(curl -sS -X POST "$url" --data "$base&code_verifier=$verifier") || die "falha na troca do token."
  else
    resp=$(curl -sS -X POST "$url" --data "$base") || die "falha na troca do token."
  fi
  if ! printf '%s' "$resp" | has_token; then
    # fallback sem PKCE (LinkedIn antigo não aceita code_verifier)
    resp=$(curl -sS -X POST "$url" --data "$base") || die "falha na troca do token (sem PKCE)."
  fi
  printf '%s' "$resp" | has_token || die "resposta inesperada do LinkedIn: $(printf '%s' "$resp" | head -c 300)"

  expires_in=$(printf '%s' "$resp" | resp_json expires_in); [ -n "$expires_in" ] || expires_in=3600
  cfg_set access_token         "$(printf '%s' "$resp" | resp_json access_token)"
  cfg_set refresh_token        "$(printf '%s' "$resp" | resp_json refresh_token)"
  cfg_set access_token_expiry  "$(( $(date +%s) + expires_in ))"
  echo ">> Token salvo em $CONF (modo 600)."
}

refresh_token() {
  need_env LINKEDIN_CLIENT_ID; need_env LINKEDIN_CLIENT_SECRET
  local rt resp
  rt=$(cfg_get refresh_token)
  [ -n "$rt" ] || die "sem refresh token salvo. Rode: $0 login"
  resp=$(curl -sS -X POST "$URI_BASE/oauth/v2/accessToken" \
    --data "grant_type=refresh_token&refresh_token=$rt&client_id=$LINKEDIN_CLIENT_ID&client_secret=$LINKEDIN_CLIENT_SECRET") || die "falha ao renovar token."
  printf '%s' "$resp" | has_token || die "resposta inesperada ao renovar: $(printf '%s' "$resp" | head -c 300)"
  cfg_set access_token          "$(printf '%s' "$resp" | resp_json access_token)"
  cfg_set refresh_token         "$(printf '%s' "$resp" | resp_json refresh_token)"
  e=$(printf '%s' "$resp" | resp_json expires_in); [ -n "$e" ] || e=3600
  cfg_set access_token_expiry   "$(( $(date +%s) + e ))"
  echo ">> Token renovado." >&2
}

person_urn() {
  local u sub
  u=$(cfg_get person_urn)
  [ -n "$u" ] && { echo "$u"; return; }
  sub=$(curl -sS -H "Authorization: Bearer $(access_token)" "$API/v2/userinfo" | resp_json sub)
  [ -n "$sub" ] || die "não consegui ler o seu person id (userinfo). Tokens ok?"
  u="urn:li:person:$sub"
  cfg_set person_urn "$u"
  echo "$u"
}

whoami() { person_urn; }

publish() {
  local file=$1; shift
  [ -f "$file" ] || die "arquivo '$file' não encontrado."
  local title="" desc="" source=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --title) title=$2; shift 2;;
      --description) desc=$2; shift 2;;
      --source) source=$2; shift 2;;
      *) die "opção desconhecida: $1";;
    esac
  done

  if [ -n "$source" ]; then
    [ -n "$title" ] || title="vidctl — vídeo no tamanho certo de cada plataforma"
    [ -n "$desc" ] || desc="Comprime vídeos para o tamanho exato de cada plataforma, mantendo duração e qualidade. Open source, 100% offline."
  fi

  body=$(python3 -c 'import json,sys
author,path,source,title,desc=sys.argv[1:6]
text=open(path, encoding="utf-8").read()
b={"author":author,"commentary":text,"visibility":"PUBLIC",
   "distribution":{"feedDistribution":"MAIN_FEED","targetEntities":[],"thirdPartyDistributionChannels":[]},
   "lifecycleState":"PUBLISHED","isReshareDisabledByAuthor":False}
if source:
    b["content"]={"article":{"source":source,"title":title,"description":desc}}
sys.stdout.write(json.dumps(b))' "$(person_urn)" "$file" "$source" "$title" "$desc")

  echo ">> Publicando como $(person_urn)..."
  local ret hdr_body
  hdr_body=$(mktemp)
  ret=$(curl -sS -i -o "$hdr_body" -w '%{http_code}' -X POST "$API/rest/posts" \
    -H "Authorization: Bearer $(access_token)" \
    -H "Linkedin-Version: $LINKEDIN_VERSION" \
    -H "X-Restli-Protocol-Version: 2.0.0" \
    -H "Content-Type: application/json" \
    --data "$body") || { rm -f "$hdr_body"; die "falha na requisição."; }
  if [ "$ret" = "201" ]; then
    echo ">> Post publicado. $(grep -i '^x-restli-id' "$hdr_body" | tr -d '\r')"
  else
    echo "erro HTTP $ret:" >&2; cat "$hdr_body" >&2; echo >&2
    rm -f "$hdr_body"; die "o post não foi publicado."
  fi
  rm -f "$hdr_body"
}

case "${1:-}" in
  login) login;;
  whoami) whoami;;
  publish) [ $# -ge 2 ] || die "uso: $0 publish <arquivo.md> [--title .. --description .. --source URL]"; publish "$2" "${@:3}";;
  *) sed -n '2,26p' "$0"; exit 0;;
esac