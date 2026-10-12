# Backlog

Fixes and improvements, pending or resolved, from most recent to oldest.

## 2026-10-11 — Resolved: audio extraction + one-video-at-a-time model

- **Phase 6 (audio extraction):** the new `internal/audio` package builds the
  ffmpeg command for mp3/opus/aac/flac/wav/copy (clear validations, seek-based
  trim, loudnorm EBU R128), wired into the queue by `Task.Kind = "audio"` with
  the same progress events (now using `-progress pipe:1`); bindings
  `ExtractAudio`/`SuggestAudioOutput`/`OpenAudioDialog` and an **"Extract
  audio"** section in the sidebar.
- **One video at a time:** "Add videos…" became **"Switch video…"** — switching
  replaces the selection and resets the per-file options (no batch enqueue);
  "Conversion queue" became **"Task queue"**; sidebar sections renamed: "Cut"
  and "Video adjustments".

## 2026-09-27 — Resolved: topbar stretches the ffmpeg indicator up to the title

- In the main window, the pill-shaped background of the `ffmpeg OK` indicator
  stretched to the "vidctl" in the center — it was the grid item stretching in
  the `1fr` column.
- Fix: removed the background/outline (`background` + `border-radius` +
  `padding` of the pill) and added `justify-self: start`; the indicator is now
  just the dot + text.
- File: `frontend/src/style.css` (`.ffmpeg-pill`).
