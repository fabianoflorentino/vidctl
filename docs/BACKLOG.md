# Backlog

Correções e melhorias pendentes ou resolvidas, da mais recente para a mais antiga.

## 2026-10-11 — Resolvido: extração de áudio + modelo de um vídeo por vez

- **Fase 6 (extração de áudio):** novo pacote `internal/audio` desenha o
  ffmpeg para mp3/opus/aac/flac/wav/copiar (validações claras, trim com seek,
  loudnorm EBU R128), enquadrado na fila por `Task.Kind = "audio"` com os
  mesmos eventos de progresso (agora com `-progress pipe:1`); bindings
  `ExtractAudio`/`SuggestAudioOutput`/`OpenAudioDialog` e seção **"Extrair
  áudio"** na barra lateral.
- **Um vídeo por vez:** "Adicionar vídeos…" virou **"Trocar vídeo…"** — trocar
  o vídeo substitui a seleção e reseta as opções por arquivo (não enfileira
  lote); "Fila de conversões" virou **"Fila de tarefas"**; seções da sidebar
  renomeadas: "Cortar" e "Ajustes de vídeo".

## 2026-09-27 — Resolvido: topbar estica o indicador de ffmpeg até o título

- Na janela principal, o fundo em forma de pílula do indicador `ffmpeg OK` ficava
  esticado até o "vidctl" no centro — era o grid item esticando na coluna `1fr`.
- Correção: removido o fundo/contorno (`background` + `border-radius` + `padding`
  do pill) e adicionado `justify-self: start`; o indicador agora é só o dot + texto.
- Arquivo: `frontend/src/style.css` (`.ffmpeg-pill`).