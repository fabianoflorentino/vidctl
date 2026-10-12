# Implementation plan — Constrict-style redesign + drag and drop

UX/visual reference: [Wartybix/Constrict](https://github.com/Wartybix/Constrict)
(a video compression app with a card sidebar, source queue, Preferences modal and
an empty state with a drop zone). This plan **reproduces the look and flow in the
Svelte 5 + Wails frontend** — there is no GTK port. Future backend dependencies
keep being orchestrated in
[`plano-implementacao.md`](./plano-implementacao.md) (`↗ Phase N` references).

Working branch: `feat/ui-constrict`. Every phase ends with the app compiling and
the current flow working; merges only after visual acceptance by the user.

---

## Technical decisions (verified in the Wails v2.16.0 source)

- **Native Wails drag and drop**, not HTML5: in `main.go`,
  `options.App.DragAndDrop = &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true}`.
  The drop target is marked in CSS with the custom property
  `--wails-drop-target: drop` (defaults `CSSDropProperty`/`CSSDropValue`); the
  frontend receives the absolute paths through the `wails:file-drop` event via
  `EventsOn('wails:file-drop', (x, y, paths) => …)` — the same `EventsOn` already
  imported in `App.svelte`.
- **Thumbnails**: a new Go binding `GetThumbnail(path)` extracts a frame via
  ffmpeg (`-ss <min(1s,dur/2)> -frames:v 1`) and returns a **base64 data-URL**
  (avoids serving a local file in the webview); cache in
  `os.TempDir()/vidctl-thumbs` keyed by path+mtime+size.
- **Single-video scope kept**: the Constrict queue look is reproduced with 1
  active video (drop replaces the selection). A real N-job queue is the
  ↗ Phase 2 of `plano-implementacao.md` (future evolution, outside this plan).
- **Theme**: dark is the visual target of the redesign; the light theme keeps
  existing, re-derived from the same tokens (the app already persists
  light/dark in `localStorage['vidctl-theme']`).
- **Preferences**: while ↗ Phase 1 (persistent Go config) does not arrive, the
  modal persists in `localStorage` (new suffix key); migrate later.

## What makes the look "Constrict" (reference checklist)

1. Empty state: camera icon, "Compress Videos" title, "Drag and drop videos
   here" subtitle, "Open…" pill button — everything is a drop zone.
2. Split view: sidebar (~300–360 px) with stacked cards, each with a title and a
   ⓘ help icon; controls inside the cards.
3. Signature controls: round −/+ stepper (Target Size, Tolerance), radio-cards
   with title+description (Framerate Limit), toggle switch (Extra Quality),
   select (Video Codec).
4. Source list: row with thumbnail, name, transformation (`720p@24 →
   144p@24`) and a ⋮ menu; list header with "Clear All".
5. Single primary action as a pill in the footer ("Export To…" / here
   "Compress…").
6. Centered "Preferences" modal with inner cards and toggles.
7. Matte dark palette (window `#141414`, sidebar `#1e1e1e`, cards `#262626`),
   12–16 px corners, sans typography (mono only for numbers/data).

---

## Phases

### Phase 0 — Baseline (0.5 h)

- Create branch `feat/ui-constrict`; run `go test ./...` and `npm run check` in
  `frontend/`; screenshot the current UI for comparison.
- Acceptance: green starting point, no code change.

### Phase 1 — Design system in `frontend/src/style.css` (0.5–1 d)

- New tokens: Constrict palette (see checklist §7), 12–16 px radius, pill
  buttons, round icons; sans as the main font (IBM Plex Mono restricted to
  numeric data); light theme re-derived from the same tokens.
- New base classes: `.card`, `.group-card`, `.stepper`, `.toggle`,
  `.radio-card`, `.select`, `.modal`, `.pill-btn`, `.icon-round`, `.kebab`,
  `.dropzone` (with `--wails-drop-target: drop`).
- Restyles the current flow **without touching logic**.
- Acceptance: `npm run check` green; current flow working with the new skin.

### Phase 2 — Components in `frontend/src/lib/` (1 d)

- `Stepper.svelte` (value + round −/+ buttons, min/max) → replaces the target
  size, parts and min/part sliders.
- `Toggle.svelte` (switch with title+description).
- `RadioCardGroup.svelte` (radio cards with title+description) → presets and cut
  mode.
- `Modal.svelte` (overlay, title, close button) → Preferences.
- `InfoTip.svelte` (ⓘ with tooltip) → replaces the `ControlGroup` `help`.
- `KebabMenu.svelte` (⋮: switch video, open folder, clear).
- Acceptance: `svelte-check` green; components consumed in Phases 3–5 (the
  project has no UI test runner; see §Tests).

### Phase 3 — Layout + drag and drop (1–1.5 d)

- `App.svelte`: minimal topbar (title + hamburger → Preferences); sidebar as
  cards (Size, Presets, Cut, Output) using the Phase 2 components; main with a
  "Video Sources"-style list (active video row + clear header); "Compress…" pill
  in the footer replacing the actionbar.
- `StatusPage.svelte` becomes the checklist §1 empty state (icon, texts, "Open…"
  pill), all of it as a drop zone.
- Drag and drop:
  - `main.go`: `DragAndDrop` option (see §Decisions).
  - `--wails-drop-target: drop` on the empty state and the list area.
  - `App.svelte`: `EventsOn('wails:file-drop', …)` → filters the video
    extension, runs the existing `GetMediaInfo` flow; drop replaces the current
    video; non-video → alert with the reason.
  - Highlight during dragover via `window.dragover/dragleave` listeners
    (acceptable fallback: no highlight, drop still works).
- Acceptance: dragging `.mp4/.mov/.mkv/.webm` selects and shows the video;
  dragging `.txt` shows an error; "Open…" keeps working.

### Phase 4 — Thumbnails via ffmpeg (0.5–1 d)

- `internal/media/thumbnail.go`: frame extraction + cache (see §Decisions);
  `GetThumbnail(path)` binding in `app.go` returning a base64 data-URL.
- `VideoRow.svelte`: thumbnail + `WxHp@fps → target` + `KebabMenu`.
- Tests in `internal/media` with an ffmpeg-lavfi-generated fixture (the pattern
  of the existing integration tests); `make cover` ≥ 80%.
- Acceptance: the video row shows the thumb in < 1 s after selection; a second
  select of the same file uses the cache.

### Phase 5 — Preferences modal (0.5 d)

- `PreferencesModal.svelte` (opened by the hamburger): theme
  (system/light/dark), output suffix (default `-compressed`), "check ffmpeg
  again" button.
- `localStorage` persistence (theme already exists; suffix is new); suffix
  applied to the suggested path in `pickInput`.
- Acceptance: preferences survive a restart; the suffix shows up in the
  suggested outputPath.

### Phase 6 — Docs, quality and release (0.5 d)

- README: drag and drop, thumbnails, new UI; CHANGELOG; screenshots in `site/`;
  update this doc with anything that diverged in execution.
- `go test -race -shuffle=on -count=1 ./...`, `make cover` (≥ 80%),
  `npm run check`, `wails build -clean -tags webkit2_41`, manual smoke
  (`VIDCTL_SMOKE=… go test -run TestSmokeReal ./internal/compress/`).
- Acceptance: green CI + the §"What makes the look Constrict" visual checklist
  verified in the built binary.

---

## Execution order and packaging

```
Phase 0 → Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6
```

Suggested PRs: **PR-A** Phases 0–2 (skin + components, low risk);
**PR-B** Phase 3 (layout + drag and drop); **PR-C** Phases 4–5 (thumbs +
Preferences); **PR-D** Phase 6 (docs/release).

## Tests

- **Backend** (AGENTS.md rule): `GetThumbnail` and extension filters tested in
  `internal/media`/`app_test.go`; ≥ 80% coverage kept
  (`go test -race ./...` + CI gate).
- **Frontend**: no UI test runner today; guaranteed via `svelte-check`
  (`npm run check`) + a manual acceptance checklist per phase with
  `make build && build/bin/vidctl`. Introducing `vitest` +
  `@testing-library/svelte` stays an optional evolution (does not block).
- **Manual per phase**: run the acceptance listed in the phase before
  considering it done.

## Risks

- `wails:file-drop` on Windows with paths containing accents/spaces — validate
  early in Phase 3 (PR-B).
- Dragover highlight may not fire in some WebKit/WebView2 (DOM events suppressed
  with `DisableWebViewDrop`) — degrading without highlight is acceptable.
- Base64 thumbs of 4K videos can weigh on memory — cap the longer side to
  ~320 px in ffmpeg (`-vf scale`) already in Phase 4.
- Rewriting all of `style.css` touches both themes — Phase 1 isolated in PR-A
  allows a quick revert.
- `compress:*`/`Job` events may change in ↗ Phase 2 (real queue); this plan only
  depends on them beyond the current contract.

## Total effort

| Block | Days |
|---|---|
| Phases 0–2 (baseline, tokens, components) | 1.5–2.5 |
| Phase 3 (layout + drag and drop) | 1–1.5 |
| Phase 4 (thumbnails) | 0.5–1 |
| Phases 5–6 (Preferences, docs/release) | 1 |
| **Total** | **~4–6 working days** (~1.5–2 d with an agent running) |

## Future evolutions (outside this plan)

- Real N-video queue with per-row progress: ↗ Phase 2 of
  [`plano-implementacao.md`](./plano-implementacao.md).
- Codec/GPU encoding and tolerance: ↗ Phases 3–4 of the same plan.
- Persistent Go config replacing `localStorage`: ↗ Phase 1.
- Responsive (collapsed sidebar ≤ 750 px) and i18n pt/en: optional, post-Phase 6.