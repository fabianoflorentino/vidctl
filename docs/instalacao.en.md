# Installation

**English · [Português (Brasil)](instalacao.md) · [Español](instalacao.es.md) ·
[中文（简体）](instalacao.zh-CN.md)**

vidctl is published as **one release per platform** — the same version becomes
three tags, each with only your system's packages:

| Release tag | Packages (always with `checksums.txt`) |
|---|---|
| `v1.x.y-linux` | `vidctl-linux-x64.zip`, `vidctl-linux-x64.deb`, `vidctl-linux-x64.rpm` |
| `v1.x.y-macos` | `vidctl-macos-arm64.zip` |
| `v1.x.y-windows` | `vidctl-windows-x64.zip` |

Download from the [releases page](https://github.com/fabianoflorentino/vidctl/releases)
the latest tag ending in your system (`-windows`, `-macos` or `-linux`).

---

## Windows

1. Download `vidctl-windows-x64.zip` from the latest release ending in [`-windows`](https://github.com/fabianoflorentino/vidctl/releases).
2. Extract the zip and run `vidctl.exe`.
3. The app requires ffmpeg and ffprobe on the `PATH` (vidctl warns on the first
   screen if they are missing):

```powershell
winget install Gyan.FFmpeg
```

> Requires the WebView2 Runtime. On Windows 10/11 it ships preinstalled.

## macOS (Apple Silicon)

1. Download `vidctl-macos-arm64.zip` from the latest release ending in [`-macos`](https://github.com/fabianoflorentino/vidctl/releases).
2. Extract and move `vidctl.app` to the Applications folder.
3. Because it is not signed, Gatekeeper asks for manual opening: right-click
   `vidctl.app` → **Open**.
4. Missing ffmpeg? `brew install ffmpeg` (and restart the app).

> Intel (x64) builds are not distributed; in that case, build from source
> (section below).

## Linux

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install ./vidctl-linux-x64.deb
```

The `.deb` declares GTK, WebKitWeb 4.1 and **ffmpeg** dependencies — `apt`
resolves everything.

### Fedora / RHEL

```bash
sudo dnf install ./vidctl-linux-x64.rpm
```

The `.rpm` installs GTK and WebKitGTK 4.1 automatically; **ffmpeg** stays as a
recommendation — if it is not on the system, the app cannot open videos. Install
it with:

```bash
sudo dnf install ffmpeg     # may require the RPM Fusion repository
```

### Generic Linux (zip)

Without a package manager:

```bash
unzip vidctl-linux-x64.zip -d vidctl
sudo install -m755 vidctl/vidctl /usr/local/bin/
sudo apt install ffmpeg    # or your distro's equivalent
```

## Arch Linux (AUR-style, manual)

No AUR account needed — the PKGBUILD lives in this repository:

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
pkg/arch/install.sh
```

The script follows the same flow as `yay`: it runs `makepkg -si` over
`pkg/aur/PKGBUILD` (downloads the tag tarball, resolves and installs
dependencies, builds from source with `wails build -clean -tags webkit2_41` and
installs the package). It requires `base-devel`.

### Directly with makepkg (without the script)

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl/pkg/aur
makepkg -si
```

- `-s` resolves and installs build and runtime dependencies (asks for sudo):
  `go`, `npm`, `gtk3`, `webkit2gtk-4.1`, `ffmpeg`
- `-i` installs the generated `.pkg.tar.zst` at the end
- Want to skip the prompts? `makepkg -si --noconfirm`

To clean the build artifacts afterwards: `makepkg --cleanbuild` (or delete the
`src/` folders and the `*.pkg.tar.zst` inside `pkg/aur/`).

Once the package is published on the AUR, the same result comes from
`yay -S vidctl` (or `paru -S vidctl`).

Want a newer version? Update the PKGBUILD before installing:

```bash
cd vidctl
git pull
pkg/aur/update-aur.sh 2.0.0   # use the version of the latest release
pkg/arch/install.sh
```

## Verify integrity (checksums)

Before installing, check the files against the release `checksums.txt`
(downloaded alongside):

```bash
sha256sum -c checksums.txt       # Linux / macOS
```

On Windows (PowerShell):

```powershell
Get-FileHash vidctl-linux-x64.zip -Algorithm SHA256
# compare with the corresponding line of checksums.txt
```

## ffmpeg, the only runtime requirement

The app is 100% offline and needs no internet; the only external program is
**ffmpeg** (together with `ffprobe`). If vidctl opens and warns they are missing,
install them through your distro:

| System | Command |
|---|---|
| Debian/Ubuntu | `sudo apt install ffmpeg` |
| Fedora/RHEL | `sudo dnf install ffmpeg` |
| Arch | `sudo pacman -S ffmpeg` |
| macOS | `brew install ffmpeg` |
| Windows | `winget install Gyan.FFmpeg` |

## Local transcription (optional)

Transcription (`.srt`/`.vtt` subtitles + text) is **optional** and uses a local
engine — nothing is sent to the cloud. Install one of the engines and point the
model in the modal options (**⋮ → "transcribe video…"**):

| Engine | Install | Model |
|---|---|---|
| whisper.cpp | `whisper-cli` binary on the PATH | `ggml-*.bin` file (suggested: `tiny`/`base`/`small`) |
| vosk | `vosk-transcriber` on the PATH | language model folder |

Without an engine the app warns and compression keeps working normally.

## Build from source (optional)

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
wails build -clean -tags webkit2_41    # requirements: see README
```

The binary comes out at `build/bin/vidctl`.