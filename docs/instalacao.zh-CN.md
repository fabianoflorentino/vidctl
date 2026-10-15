# 安装

**中文（简体）· [Português (Brasil)](instalacao.md) · [English](instalacao.en.md) ·
[Español](instalacao.es.md)**

vidctl 按**每个平台一个发布版本**发布 — 同一版本对应三个标签，每个只包含你所在系统的软件包：

| 发布标签 | 软件包（始终带 `checksums.txt`） |
|---|---|
| `v1.x.y-linux` | `vidctl-linux-x64.zip`、`vidctl-linux-x64.deb`、`vidctl-linux-x64.rpm` |
| `v1.x.y-macos` | `vidctl-macos-arm64.zip` |
| `v1.x.y-windows` | `vidctl-windows-x64.zip` |

请从[发布页面](https://github.com/fabianoflorentino/vidctl/releases)下载以你所在系统为后缀的最新标签（`-windows`、`-macos` 或 `-linux`）。

---

## Windows

1. 从最新的 [`-windows`](https://github.com/fabianoflorentino/vidctl/releases) 发布版下载 `vidctl-windows-x64.zip`。
2. 解压 zip 并运行 `vidctl.exe`。
3. 应用要求在 `PATH` 中提供 ffmpeg 和 ffprobe（缺失时 vidctl 会在首屏提示）：

```powershell
winget install Gyan.FFmpeg
```

> 需要 WebView2 Runtime。Windows 10/11 已预装。

## macOS（Apple Silicon）

1. 从最新的 [`-macos`](https://github.com/fabianoflorentino/vidctl/releases) 发布版下载 `vidctl-macos-arm64.zip`。
2. 解压并把 `vidctl.app` 移到“应用程序”文件夹。
3. 由于未签名，Gatekeeper 要求手动打开：右键 `vidctl.app` → **打开**。
4. 缺少 ffmpeg？运行 `brew install ffmpeg`（然后重启应用）。

> 不提供 Intel（x64）构建；这种情况下请从源码编译（见下文）。

## Linux

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install ./vidctl-linux-x64.deb
```

`.deb` 声明了 GTK、WebKitWeb 4.1 和 **ffmpeg** 依赖 — `apt` 会全部解决。

### Fedora / RHEL

```bash
sudo dnf install ./vidctl-linux-x64.rpm
```

`.rpm` 会自动安装 GTK 和 WebKitGTK 4.1；**ffmpeg** 属于推荐项 — 若系统没有，应用将无法打开视频。安装它：

```bash
sudo dnf install ffmpeg     # 可能需要 RPM Fusion 仓库
```

### 通用 Linux（zip）

不使用包管理器时：

```bash
unzip vidctl-linux-x64.zip -d vidctl
sudo install -m755 vidctl/vidctl /usr/local/bin/
sudo apt install ffmpeg    # 或对应 distro 的等效命令
```

## Arch Linux（类 AUR，手动）

无需 AUR 账号 — PKGBUILD 就在本仓库中：

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
pkg/arch/install.sh
```

脚本与 `yay` 的流程相同：对 `pkg/aur/PKGBUILD` 运行 `makepkg -si`（下载标签 tarball、解析并安装依赖、使用 `wails build -clean -tags webkit2_41` 从源码编译并安装软件包）。需要 `base-devel`。

### 直接用 makepkg（不使用脚本）

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl/pkg/aur
makepkg -si
```

- `-s` 解析并安装构建与运行依赖（需要 sudo）：`go`、`npm`、`gtk3`、`webkit2gtk-4.1`、`ffmpeg`
- `-i` 安装最终生成的 `.pkg.tar.zst`
- 想跳过确认？`makepkg -si --noconfirm`

之后清理构建产物：`makepkg --cleanbuild`（或删除 `pkg/aur/` 内的 `src/` 文件夹和 `*.pkg.tar.zst`）。

当软件包发布到 AUR 后，可用 `yay -S vidctl`（或 `paru -S vidctl`）获得相同结果。

想要更新版本？安装前先更新 PKGBUILD：

```bash
cd vidctl
git pull
pkg/aur/update-aur.sh 2.0.0   # 使用最新发布版的版本号
pkg/arch/install.sh
```

## 校验完整性（checksums）

安装前，请将文件与发布版的 `checksums.txt`（随附下载）核对：

```bash
sha256sum -c checksums.txt       # Linux / macOS
```

Windows 上（PowerShell）：

```powershell
Get-FileHash vidctl-linux-x64.zip -Algorithm SHA256
# 与 checksums.txt 中对应的一行比较
```

## ffmpeg，唯一的运行时要求

应用 100% 离线，无需联网；唯一的外部程序是 **ffmpeg**（连同 `ffprobe`）。如果 vidctl 打开并提示缺失，请通过你的发行版安装：

| 系统 | 命令 |
|---|---|
| Debian/Ubuntu | `sudo apt install ffmpeg` |
| Fedora/RHEL | `sudo dnf install ffmpeg` |
| Arch | `sudo pacman -S ffmpeg` |
| macOS | `brew install ffmpeg` |
| Windows | `winget install Gyan.FFmpeg` |

## 本地转写（可选）

转写（`.srt`/`.vtt` 字幕 + 文本）是**可选**功能，使用本地引擎 — 不上传云端。
安装以下任一引擎，并在弹窗选项中指定模型（**⋮ → “转写视频…”**）：

| 引擎 | 安装 | 模型 |
|---|---|---|
| whisper.cpp | PATH 中的 `whisper-cli` 二进制 | `ggml-*.bin` 文件（建议 `tiny`/`base`/`small`） |
| vosk | PATH 中的 `vosk-transcriber` | 对应语言的模型目录 |

未安装引擎时应用会提示，压缩功能照常可用。

## 从源码编译（可选）

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
wails build -clean -tags webkit2_41    # 要求：见 README
```

二进制输出到 `build/bin/vidctl`。