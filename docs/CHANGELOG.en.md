# Changelog

**English · [Português (Brasil)](../CHANGELOG.md) · [Español](CHANGELOG.es.md) ·
[中文（简体）](CHANGELOG.zh-CN.md)**

All the relevant vidctl changes, in the
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format, with
[SemVer](https://semver.org/) versioning and one release per platform following
the publishing workflow.

The per-platform releases (`vX.Y.Z-windows`, `vX.Y.Z-linux`, `vX.Y.Z-macos`)
share this same changelog: the only difference between them is the operating
system of the packages.

## [Unreleased]

- **Cleaner main screen**: the top bar and footer are gone. The **options**
  (Preferences) button lives in a panel in the bottom-right corner, with the
  same design as the sidebar, and the **ffmpeg** status appears inside the menu
  itself, with a green indicator when available. The drag-and-drop area (empty
  state) moved inside a panel with the same design as the sidebar. The window
  title changed from "vidctl — video compressor" to **"video controller"**.

## [2.9.1] — 2026-10-18

- **Fixed — update check**: the **AUR** and **RPM** packages built without
  embedding the version (the binary stayed `dev`), so the check concluded the
  app was up to date. Now the `PKGBUILD` and the RPM spec pass
  `-ldflags -X ...internal/version.Version`, like the `.deb` and the `.zip`.
  And when the local version cannot be determined, the app shows
  **"unknown local version"** instead of claiming it is up to date.

## [2.9.0] — 2026-10-18

- **New visual identity**: the app's highlight blue moved from a washed tone to
  a **vivid blue** — `#3b82f6` in the dark theme and `#2563eb` in the light one —
  making the interface more vibrant and professional. The website got the same
  treatment (dark navy background, Inter + JetBrains Mono, entrance animations
  and the brand with the app icon + "vidctl").

## [2.8.0] — 2026-10-17

- **Local transcription** (phase 12): from the video menu ("transcribe
  video…") the app generates `.srt`/`.vtt` subtitles and plain `.txt` using an
  **offline** speech-to-text engine (whisper.cpp/`whisper-cli` or vosk), with
  `auto` detecting what is installed. Audio extracted to 16 kHz mono PCM,
  language `auto`/`pt`/`en`, format `all`/`srt`/`vtt`/`txt`, word timestamps
  and an `mm:ss` window. Everything runs locally — nothing is sent to the
  cloud; without an engine the app points you to the install. It joins the same
  queue (`Task.Kind = transcribe`) with progress, cancel and "open folder".

- **Fixed — time cut**: removed the 1-min-per-part minimum. The leftover now
  becomes the last part even when shorter than 1 min (e.g., a 1:49 video →
  `1:00` + `0:49`) instead of blocking the cut. The 60-part cap stays.

## [2.7.0] — 2026-10-14

- **Custom presets** (phase 9): create, edit and delete your own presets on top
  of the built-ins — name, mode (size or CRF), audio bitrate, codec (H.264/H.265)
  and encoder (software/NVENC/AMF/QSV/VideoToolbox). Saved in `presets.json` in
  the configuration folder (atomic write), they appear in the Presets section
  and compress like any other, with `custom-` IDs and field validation.

## [2.6.0] — 2026-10-13

- **Update check** (phase 8): on launch the app queries the latest GitHub
  release (`internal/update`, 5 s timeout and per-session cache) and shows the
  badge "vX.Y.Z available" in the top bar when a newer version exists —
  clicking opens the release in the browser. A refresh button re-runs the
  check; offline and network errors are silent. The binary version is now
  baked in via `-ldflags` (Makefile and release workflow).

## [2.5.0] — 2026-10-12

- **System notifications** (phase 7): a desktop alert when a task finishes or
  fails (`notifyOnDone`) and an option to reveal the output folder on success
  (`openFolderOnDone`), both configurable in the new "Notifications" section of
  Preferences. The backend delivers via `beeep` (toast on Windows, Notification
  Center on macOS, `notify-send` on Linux) and opens the folder with the
  platform file manager; notifications never affect the conversion result.
- **Code organization**: the Wails binding facade moved to `internal/app` (the
  root now keeps only the entrypoint and a thin adapter that pins the `main.App`
  namespace for the frontend), and system theme detection moved to
  `internal/systemtheme`; the English, Spanish and Chinese `README`/`CHANGELOG`
  moved to `docs/`, with links and the project tree updated.

## [2.4.0] — 2026-10-12

- **i18n (pt-BR/en/es/zh)**: the interface gained dictionaries in Portuguese
  (Brazil, default), English, Spanish and Simplified Chinese (Mandarin). The
  language is chosen in Preferences and stays saved between runs
  (`config.Language`).
- **Site and docs in 4 languages**: the landing page gained a language
  switcher (PT/EN/ES/中文); `README`, `CHANGELOG` and the install guide have
  pt-BR, en, es and zh-CN versions. The internal implementation plan stayed in
  pt (it is disposable).
- **English codebase**: comments (go doc style), user-facing messages, logs and
  test labels were anglicized in the backend (Go) and frontend (Svelte/TS); the
  UI dictionaries are the only source of pt text.

## [2.3.0] — 2026-10-11

- **Extract audio from a video** (phase 6): the new **"Extract audio"** section
  of the sidebar pulls the track without re-encoding the video, choosing format
  (mp3, opus, aac, flac, wav or copy — stream copy, remux only), quality (kbps),
  channels (mono/stereo), EBU R128 normalization (loudnorm) and a time window
  (mm:ss). The extraction joins the same task queue, with progress, cancellation
  and open folder when done. The output file suffix is configurable in
  Preferences (`-audio` by default).
- **Sidebar names**: "Cut into parts" became **"Cut"** and "Per-file
  adjustments" became **"Video adjustments"**, making clear what belongs to
  audio (extraction) and what belongs to video (scale, cut, FPS, rotation…)
- **One video at a time**: "Add videos…" became **"Switch video…"** — switching
  replaces the selection, **resets the per-file options** to default and no
  longer enqueues a batch (the app processes one job at a time, sequential).
- **Conversion queue → Task queue**: the queue now lists any job (compress or
  extract audio), with a task counter.
- **Less redundancy**: with a video loaded, the drop-target area on top of the
  content was removed — switching stays on the **"Switch video…"** button or by
  dropping a new file anywhere in the window; the area only remains on the empty
  screen.
- **Bigger typography**: every window interface size went up **1px** (the
  hierarchy is kept; dense panel and queue text reads more comfortably).
- **"Extract audio" button on the video card**: audio options stay in the
  sidebar, but the extract button now lives in its own card below the video
  destination (same look as "Save as…"), with the audio path showing the
  selected format's extension (`.mp3`, `.flac`, `.wav`…) and a **"Save as…"**
  to choose another destination.
- **Bigger default window**: the window opens at **1360×768** (was 1000×700).
- **"Target size" → "Size"**: the sidebar section became just **Size** and the
  control label became **megabytes** (removing the MB = megabit ambiguity).
- **Faithful "system" theme**: the "System" appearance option now reads the real
  desktop theme (gsettings/GTK on Linux, `AppleInterfaceStyle` on macOS, registry
  on Windows) instead of trusting WebKit's `prefers-color-scheme`, which on
  Linux reported "light" even when the system was dark.

## [2.2.1] — 2026-10-11

- Unified icon in the installers: the same favicon design (video/play with
  gradient) in every platform's packages (v2.2.0 still shipped the previous icon
  because it was compiled before the #34 merge).

## [2.2.0] — 2026-10-11

Preferences persistence, batch conversion queue, size estimate, video codecs with
GPU acceleration and per-file adjustments: preferences survive closing, you can
compress several videos at once, the expected size appears before compressing,
the **Advanced** card switches to H.265 or hardware encoders
(NVENC/QSV/AMF/VideoToolbox) and every video accepts scale, cut, audio removal,
FPS, rotation and thumbnail.

### Added

- **Size estimate**: in size mode, the "Target size" panel shows a badge with the
  expected output size — computed with the same bit budget as the 2-pass encode
  (now single, in `internal/estimate`) — plus the saving over the original file
  when the target is smaller. New `EstimateSize` binding; CRF mode has no
  forecast yet and the badge does not appear.
- **H.265/x265 codec**: the step 02 "Advanced" card lets you choose H.264 or
  H.265 per job; the preset keeps its default codec. Together with the existing
  2-pass, size mode now uses `libx265` with the same precision as x264 (the
  per-encoder argument builder now lives in `internal/encode`).
- **GPU acceleration (NVENC/QSV/AMF/VideoToolbox)**: hardware is optional per
  job (opt-in), and backends that this computer's `ffmpeg` does not have, or
  whose driver failed the 1-frame probe, appear disabled — new `GetEncoders`
  binding with cache and re-detection ("check again"). Size-mode GPUs use 1-pass
  VBR (approximate target, ±5–10%), and the UI warns when a hardware encoder is
  active; software remains the default.
- **Per-file adjustments**: new **"Per-file adjustments"** card in step 02 with
  scale (default/original/custom), window cut in `mm:ss`, audio removal, FPS,
  rotation (transpose) and thumbnail generation at the end of the encode. Filter
  builder in `internal/compress/adjust.go` with fixed order transpose→scale→fps,
  `-ss/-to` before `-i`, size budget and progress based on the cut duration,
  conflict validation (manual cut × split, inverted cut, out-of-range
  rotation/FPS) and a new `OpenThumbnailDialog` binding.
- **NVENC preset (speed × quality)**: with the NVIDIA encoder active, the
  **Advanced** card gets a per-job `p1`–`p7` preset selector (default `p4`,
  balanced). `p1`/`p2` almost double the encode speed on the video engine (which
  is already at 100%), with a slight compression-efficiency loss in size mode;
  `p7` prioritizes quality. Hardware encoding does not use the graphics engine —
  the panel's "GPU %" reflects the maximum between graphics and encoder.
- **Collapsible sidebar**: the configuration cards (`Target size`/`Quality`,
  `Advanced`, `Presets`, `Cut into parts`, `Per-file adjustments`) collapse and
  expand by their header (chevron, mouse or keyboard). `Advanced`, `Cut into
  parts` and `Adjustments` come **closed** by default, reducing visual noise; the
  state lasts for the session.
- **Total conversion time**: every queue job shows the elapsed time in real time
  during processing and the **total** when done (progress card, queue row and
  result card) — useful for comparing presets and encoders (software × GPU,
  NVENC `p1`–`p7`).
- **Drag and drop on the main window**: the content gains a drop-target area with
  "drag a video here" + an **"open video…"** button, visible even with a video
  already loaded — dropping a file switches the input and several files enqueue
  (reusing `wails:file-drop`).
- **Local debug mode**: `VIDCTL_DEBUG=1 ./build/bin/vidctl` turns on a terminal
  log with queue transitions, emitted/received events (and patches discarded by
  state), `Compress`/`GetTasks` calls and frontend JS exceptions — to diagnose
  the progress UI without affecting normal execution (off by default).
- **Conversion queue (batch)**: "Add videos…" enqueues several files at once (or
  drop several in the window), one job per file with the current preset and
  output; the queue appears in place of step 04 with name, preset, status,
  progress bar of the running item, per-item cancel and "clear queue". New
  `compress:queued`/`compress:start` events and `JobID` routing in the frontend:
  an error in one item does not bring the others down. New bindings:
  `CompressMultiple`, `GetTasks` and `ClearFinished`.
- **Persistent configuration**: preset, size in MB, CRF and destination folder
  come back as they were on the next boot. File in
  `<config-dir>/vidctl/config.json` (Linux `~/.config`, macOS
  `~/Library/Application Support`, Windows `%AppData%`), written atomically so a
  crash does not corrupt it. A missing or invalid file opens the app with the
  defaults instead of failing.
- **Preferences → Binaries**: fields to point `ffmpeg` and `ffprobe` outside the
  PATH. The configured path takes precedence over the PATH, applies immediately
  (no restart) and an invalid path is reported as an error instead of silently
  falling back to the PATH binary.
- **Preferences → Output**: destination folder field; empty keeps saving next to
  the original video.

### Changed

- The sidebar became a **rounded panel** (GNOME/floating-plate style): same
  color as the content, with a soft outline (`18px` radius) and a subtle shadow
  instead of the "square" column with a full divider line — the separation is now
  by space and rounded corners, with a per-theme shadow (light/dark) and no
  system dependency.
- The sidebar **scrollbar** is UI-native: the native one (GTK/WebKit) was hidden
  — it no longer leaks out of the panel nor grows over the content on hover. The
  custom bar (thin, `8px`, inset) appears when scrolling/hovering/dragging and
  fades after ~0.8s idle, with the same behavior on all three platforms.
- The progress card follows the selected queue item and cancellation is now per
  item — the action no longer stops the whole job.
- `maxParallel` now takes effect: it defines how many videos are compressed at
  once (default **1** = sequential), applied at startup and when saving
  Preferences.
- `CheckFFmpeg`, `GetMediaInfo` and the thumbnail now honor the configured binary
  path, with separate messages for "not on the PATH" and "the path you configured
  does not work".
- `language`, `notifyOnDone` and `openFolderOnDone` fields are already preserved
  in the file, but only take effect once the matching features exist (future
  plan phases).
- The debug mode (`VIDCTL_DEBUG=1`) now traces **external commands and the
  pipeline decision**: the full `ffmpeg` command (2-pass, single-pass and
  thumbnail), the per-segment encoder/hardware choice (`encoder=hevc_nvenc`,
  2-pass, `-vf` chain, seek, audio removal, FPS, rotation), the `ffprobe` and the
  thumbnail (`[media]`), and the encoder listing/probe (`[encode]`, with the
  reason when a backend fails). Every command is reproducible line by line.

### Fixed

- Encoder detection no longer hangs file selection when a backend is listed by
  `ffmpeg` but fails the 1-frame probe (e.g., `h264_amf` without an AMD GPU):
  `codecs` came as `null` in the JSON and the frontend crashed while building the
  encoder list. Detection now always returns a list (empty when there are no
  codecs) and the frontend also tolerates `null`.
- The progress panel's **"GPU %"** only measured `utilization.gpu`
  (graphics/compute), which stays ~0 during an NVENC encode — video encoding
  loads the **encoder** engine (`utilization.encoder`), not the graphics bar. The
  collector now reports the **maximum of the two**, so the panel reflects real
  hardware encoding.
- The hardware encoder probe now uses a `256×256` frame: the previous `64×64`
  was **below NVENC's minimum size** ("Frame Dimension less than the minimum
  supported value"), so modern NVIDIA GPUs appeared as "detected but did not run"
  even with a working driver.
- Per-platform backends: **AMF (AMD)** is only offered on Windows and
  **VideoToolbox** only on macOS; NVENC and QSV keep being detected on any OS and
  **Software**/"Automatic" always remain available. Also, the failed-probe reason
  now appears in the UI (**Advanced** card) instead of a generic text.
- **Drag and drop** working on all three platforms: the app not only listens to
  `wails:file-drop` but registers the runtime `OnFileDrop` (which wires the WebView
  drop listeners). On **Linux** the native Wails handler makes WebKit navigate to
  the dropped file (the video "plays" and the UI disappears), so there the path is
  read from the DOM itself (`text/uri-list`) without navigation; Windows/macOS
  keep the native Wails drop. The whole window is the drop area and, while
  dragging, a **dotted outline around the screen** shows where to drop.
- **Output and thumbnail default next to the source**: without a saved
  destination folder, the suggested paths went to the app's working directory
  (the thumbnail file "vanished" for those looking next to the video). Now the
  default is the source video's folder (or the folder configured in
  Preferences).
- The generated **thumbnail** uses a frame **5 seconds after the video's start**
  (first segment) instead of the middle of the last segment — in part cuts it
  landed near the end and looked like the "last frame"; it also avoids black
  opening screens. Short videos (under 10s) fall back to the middle.

## [2.1.0] — 2026-09-27

Visual redesign in the Constrict style with drag and drop, thumbnails and fine
counter tuning.

### Added

- **Constrict-style visuals**: matte dark theme (recent tokens with slate
  accent), derived light theme and a custom icon in the Linux/Windows/macOS
  installers; two-column layout (settings sidebar + video area), cards per group,
  and the Preferences modal (system/light/dark theme and ffmpeg recheck).
- **Drag and drop**: drop a video into the window to select it; the empty area is
  highlighted on hover (`--wails-drop-target`, `wails:file-drop` event).
- **Thumbnails**: the video row shows a frame extracted via ffmpeg (looking at
  `min(1s, dur/2)`, capped at 320px), cached by path+size+mtime in
  `os.TempDir()/vidctl-thumbs`; new `GetThumbnail` binding.
- **Editable counters with fine tuning**: click the number to type (Enter/click
  outside confirms, Esc cancels, comma accepted) and hold **Shift** on the `+/−`
  arrows (or `↑/↓`) for a fine 0.1 step.
- **Fractional min/part end to end**: `split.Spec.MinutesEach` accepts fractions
  (`1.5` = 1 m 30 s) from the frontend to the cut, with a tested
  `internal/split` layer.
- **Video row menu (⋮)**: switch file / clear.
- App icon redesign: rounded dark background with a compression glyph in slate
  accent (appicon.png, icon.ico and Linux pkg/icon).

### Changed

- Constrict plan phase 4 delivered: base components (Stepper, Toggle,
  RadioCardGroup, Modal, InfoTip, KebabMenu) without scoped styles, moved into
  the global design system.
- `internal/media` now also generates thumbnails; Preferences, pipeline and
  structure documentation updated in the README.

## [2.0.1] — 2026-09-22

Polishes the v2.0.0 experience: assisted quality and interface adjustments.

### Added

- **Pre-encode quality warning**: video bitrate estimate (per part when cutting)
  compared to a per-resolution-tier minimum; when low, the sidebar shows the
  reason and a one-click applicable suggestion — raise the MB target, cut into X
  min parts or move to a CRF preset.
- **Job summary card** in the free space below the video when idle: destination +
  target, cut plan and estimated vs. minimum bitrate — the same place the
  progress card occupies during processing.
- v2-era landing page on GitHub Pages, with an animated demo faithful to the real
  UI (preset pick → cut → processing → light theme).

### Changed

- Sidebar: the preset picker group moved from "Destination" to **"Presets"**.
- Wider sidebar (300→344px) to fit the cut/quality banners.
- Release became 100% intentional: pushes on `main` no longer trigger the
  workflow (only `gh workflow run release.yml --ref vX.Y.Z` on a tag).

### Fixed

- Preset descriptions in the sidebar now wrap and show the full text (previously
  truncated with "…" on Instagram/YouTube).

- Preset descriptions in the sidebar now wrap and show the full text (previously
  truncated with "…" on Instagram/YouTube).
- Wider sidebar (300→344px) and **job summary card** in the free space below the
  video when idle: destination + target, cut plan and estimated vs. minimum
  bitrate — the same place the progress card occupies during processing.
- **Pre-encode quality warning**: video bitrate estimate (per part when cutting)
  compared to a per-resolution-tier minimum; when low, the sidebar shows the
  reason and a one-click applicable suggestion — raise the MB target, cut into X
  min parts or move to a CRF preset.
- Sidebar: the preset picker group moved from "Destination" to **"Presets"**.
- Release became 100% intentional: pushes on `main` no longer trigger the
  workflow (only `gh workflow run release.yml --ref vX.Y.Z` on a tag).

## [2.0.0] — 2026-09-22

First major interface revision + time-based video cutting.

### Added

- **Cut into parts (split)**: divides the video by time — N equal parts or X
  minutes per part (e.g., 30 min → 6 × 5 min) — with at least 1 min per part and
  a 60-part cap; each part goes through the chosen compression and comes out as
  `name-partN.mp4` (`internal/split` + `Job.Split` in the pipeline).
- **Constrict-style interface redesign** (branch `feat/ui-constrict`): full-height
  layout with a control sidebar (Destination, Quality/Target size, Cut into
  parts, Output), a StatusPage-style empty state, the video row as a card with
  thumbnail/status/progress PIE and a bottom action bar.
- **Manual light/dark theme**: sun/moon button at the top; follows the system by
  default and the choice stays persisted between runs.
- **Enriched progress panel**: a dedicated card below the video (same width and
  radius) with per-part aggregated percentage and live CPU, memory, ffmpeg % and
  GPU (NVIDIA when available) — new `internal/sysinfo` via 1s polling
  (`GetUsage`).
- `OpenMultipleDialog` in the backend for multi-file selection (base of the
  future batch queue).
- Roadmap: phase 11 (segment cut) and the UI plan `docs/plano-ui-constrict.md`.

### Fixed

- Footer/action with a giant empty strip on tall windows (layout now anchors
  `html/body/#app` at 100% with a flex column).
- Unwanted horizontal scroll in the sidebar (flex children min-width +
  overflow-x).
- Empty-state icon (switched from a "stack" to a video camera).

## [1.0.17] — 2026-09-13

First **production-ready** version of vidctl.

### Added

- Video compression with per-platform presets: WhatsApp Status (10 MB),
  WhatsApp (16 MB), Instagram (16 MB), Shorts (30 MB), YouTube (CRF) or a
  user-defined free size (2–100 MB).
- ffmpeg **2-pass** encoding to hit the maximum size without blowing the chosen
  limit.
- Real-time progress per stage (pass 1/2 and pass 2/2) with cancellation at any
  moment.
- `ffmpeg`/`ffprobe` verification at startup, with a "check again" button that
  re-reads the system PATH — lets you install ffmpeg with the app open without
  restarting.
- Native desktop for **Linux, Windows and macOS** (Wails v2 + WebKit/WebView2).
- Offline mode: fonts and interface embedded in the binary.
- Per-platform packages (.zip, .deb and .rpm) with a published `checksums.txt`
  verified on every release.
- Default scale with a maximum width of 1280 px preserving the video aspect.

### Fixed

- Windows: conversions and file probing no longer open console windows
  (`CREATE_NO_WINDOW`).
- Windows: "check again" detects a newly installed ffmpeg/ffprobe without
  restarting the app (re-reads user/machine PATH in the registry).
- Release .zip packages no longer included the whole project resource tree.

### Specifications

- Go 1.25+, Wails v2.15, Svelte 5, ffmpeg/ffprobe required on the system.
- Each release publishes only its OS's packages (Windows: `.zip`; Linux: `.zip`,
  `.deb`, `.rpm`; macOS: `.zip` with `vidctl.app`).

[1.0.17]: https://github.com/fabianoflorentino/vidctl/releases/tag/v1.0.17