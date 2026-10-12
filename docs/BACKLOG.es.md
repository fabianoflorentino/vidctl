# Backlog

Correcciones y mejoras, pendientes o resueltas, de la más reciente a la más antigua.

## 2026-10-11 — Resuelto: extracción de audio + modelo de un vídeo a la vez

- **Fase 6 (extracción de audio):** el nuevo paquete `internal/audio` arma el
  ffmpeg para mp3/opus/aac/flac/wav/copiar (validaciones claras, trim con seek,
  loudnorm EBU R128), encolado por `Task.Kind = "audio"` con los mismos eventos
  de progreso (ahora con `-progress pipe:1`); bindings
  `ExtractAudio`/`SuggestAudioOutput`/`OpenAudioDialog` y sección **"Extraer
  audio"** en la barra lateral.
- **Un vídeo a la vez:** "Agregar vídeos…" pasó a **"Cambiar vídeo…"** — cambiar
  reemplaza la selección y reinicia las opciones por archivo (sin encolar lote);
  "Cola de conversiones" pasó a **"Cola de tareas"**; secciones de la sidebar
  renombradas: "Cortar" y "Ajustes de vídeo".

## 2026-09-27 — Resuelto: la topbar estira el indicador de ffmpeg hasta el título

- En la ventana principal, el fondo en forma de píldora del indicador `ffmpeg OK`
  se estiraba hasta el "vidctl" del centro — era el item del grid estirándose en
  la columna `1fr`.
- Corrección: se quitó el fondo/contorno (`background` + `border-radius` +
  `padding` de la píldora) y se añadió `justify-self: start`; el indicador ahora
  es solo el punto + texto.
- Archivo: `frontend/src/style.css` (`.ffmpeg-pill`).
