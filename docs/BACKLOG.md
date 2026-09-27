# Backlog

Correções e melhorias pendentes ou resolvidas, da mais recente para a mais antiga.

## 2026-09-27 — Resolvido: topbar estica o indicador de ffmpeg até o título

- Na janela principal, o fundo em forma de pílula do indicador `ffmpeg OK` ficava
  esticado até o "vidctl" no centro — era o grid item esticando na coluna `1fr`.
- Correção: removido o fundo/contorno (`background` + `border-radius` + `padding`
  do pill) e adicionado `justify-self: start`; o indicador agora é só o dot + texto.
- Arquivo: `frontend/src/style.css` (`.ffmpeg-pill`).