# Instalación

**Español · [Português (Brasil)](instalacao.md) · [English](instalacao.en.md) ·
[中文（简体）](instalacao.zh-CN.md)**

vidctl se publica como **una release por plataforma** — la misma versión se
vuelve tres tags, cada una con solo los paquetes de tu sistema:

| Tag de release | Paquetes (siempre con `checksums.txt`) |
|---|---|
| `v1.x.y-linux` | `vidctl-linux-x64.zip`, `vidctl-linux-x64.deb`, `vidctl-linux-x64.rpm` |
| `v1.x.y-macos` | `vidctl-macos-arm64.zip` |
| `v1.x.y-windows` | `vidctl-windows-x64.zip` |

Descarga de la [página de releases](https://github.com/fabianoflorentino/vidctl/releases)
la tag más reciente que termine en tu sistema (`-windows`, `-macos` o `-linux`).

---

## Windows

1. Descarga `vidctl-windows-x64.zip` de la release más reciente que termine en [`-windows`](https://github.com/fabianoflorentino/vidctl/releases).
2. Extrae el zip y ejecuta `vidctl.exe`.
3. La app exige ffmpeg y ffprobe en el `PATH` (vidctl avisa en la primera pantalla si faltan):

```powershell
winget install Gyan.FFmpeg
```

> Requiere el WebView2 Runtime. En Windows 10/11 ya viene instalado.

## macOS (Apple Silicon)

1. Descarga `vidctl-macos-arm64.zip` de la release más reciente que termine en [`-macos`](https://github.com/fabianoflorentino/vidctl/releases).
2. Extrae y mueve `vidctl.app` a la carpeta Aplicaciones.
3. Por no estar firmado, Gatekeeper pide apertura manual: clic derecho en `vidctl.app` → **Abrir**.
4. ¿Falta ffmpeg? `brew install ffmpeg` (y reinicia la app).

> No se distribuyen builds para Intel (x64); en ese caso, compila desde el fuente (sección abajo).

## Linux

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install ./vidctl-linux-x64.deb
```

El `.deb` declara dependencias de GTK, WebKitWeb 4.1 y **ffmpeg** — `apt` resuelve todo.

### Fedora / RHEL

```bash
sudo dnf install ./vidctl-linux-x64.rpm
```

El `.rpm` instala GTK y WebKitGTK 4.1 automáticamente; el **ffmpeg** queda como recomendación — si no está en el sistema, la app no abre vídeos. Instálalo con:

```bash
sudo dnf install ffmpeg     # puede requerir el repositorio RPM Fusion
```

### Linux genérico (zip)

Sin gestor de paquetes:

```bash
unzip vidctl-linux-x64.zip -d vidctl
sudo install -m755 vidctl/vidctl /usr/local/bin/
sudo apt install ffmpeg    # o el equivalente de tu distro
```

## Arch Linux (estilo AUR, manual)

No se necesita cuenta en el AUR — el PKGBUILD vive en este repositorio:

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
pkg/arch/install.sh
```

El script sigue el mismo flujo de `yay`: ejecuta `makepkg -si` sobre
`pkg/aur/PKGBUILD` (descarga el tarball de la tag, resuelve e instala las
dependencias, compila del fuente con `wails build -clean -tags webkit2_41` e
instala el paquete). Requiere `base-devel`.

### Directo con makepkg (sin el script)

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl/pkg/aur
makepkg -si
```

- `-s` resuelve e instala las dependencias de build y runtime (pide sudo):
  `go`, `npm`, `gtk3`, `webkit2gtk-4.1`, `ffmpeg`
- `-i` instala el `.pkg.tar.zst` generado al final
- ¿Quieres saltarte las confirmaciones? `makepkg -si --noconfirm`

Para limpiar los artefactos de compilación luego: `makepkg --cleanbuild` (o borra
las carpetas `src/` y los `*.pkg.tar.zst` dentro de `pkg/aur/`).

Cuando el paquete esté publicado en el AUR, el mismo resultado viene con
`yay -S vidctl` (o `paru -S vidctl`).

¿Quieres una versión más nueva? Actualiza el PKGBUILD antes de instalar:

```bash
cd vidctl
git pull
pkg/aur/update-aur.sh 2.0.0   # usa la versión de la última release
pkg/arch/install.sh
```

## Verificar integridad (checksums)

Antes de instalar, comprueba los archivos contra el `checksums.txt` de la release
(descargado junto):

```bash
sha256sum -c checksums.txt       # Linux / macOS
```

En Windows (PowerShell):

```powershell
Get-FileHash vidctl-linux-x64.zip -Algorithm SHA256
# compara con la línea correspondiente del checksums.txt
```

## ffmpeg, el único requisito de ejecución

La app es 100% offline y no necesita internet; el único programa externo es
**ffmpeg** (junto con `ffprobe`). Si vidctl abre y avisa que faltan, instálalos
por tu distro:

| Sistema | Comando |
|---|---|
| Debian/Ubuntu | `sudo apt install ffmpeg` |
| Fedora/RHEL | `sudo dnf install ffmpeg` |
| Arch | `sudo pacman -S ffmpeg` |
| macOS | `brew install ffmpeg` |
| Windows | `winget install Gyan.FFmpeg` |

## Transcripción local (opcional)

La transcripción (subtítulos `.srt`/`.vtt` + texto) es **opcional** y usa un
motor local — nada se envía a la nube. Instala uno de los motores e indica el
modelo en las opciones del modal (**⋮ → "transcribir vídeo…"**):

| Motor | Instalación | Modelo |
|---|---|---|
| whisper.cpp | binario `whisper-cli` en el PATH | archivo `ggml-*.bin` (sugerencia: `tiny`/`base`/`small`) |
| vosk | `vosk-transcriber` en el PATH | carpeta del modelo del idioma |

Sin motor instalado la app avisa y la compresión sigue funcionando normalmente.

## Compilar del fuente (opcional)

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
wails build -clean -tags webkit2_41    # requisitos: ver README
```

El binario sale en `build/bin/vidctl`.