# Instalação

O vidctl é publicado em **uma release por plataforma** — a mesma versão vira
três tags, cada uma com apenas os pacotes do seu sistema:

| Tag da release | Pacotes (sempre com `checksums.txt`) |
|---|---|
| `v1.x.y-linux` | `vidctl-linux-x64.zip`, `vidctl-linux-x64.deb`, `vidctl-linux-x64.rpm` |
| `v1.x.y-macos` | `vidctl-macos-arm64.zip` |
| `v1.x.y-windows` | `vidctl-windows-x64.zip` |

Baixe na [página de releases](https://github.com/fabianoflorentino/vidctl/releases)
a tag mais recente que termina no seu sistema (`-windows`, `-macos` ou `-linux`).

---

## Windows

1. Baixe `vidctl-windows-x64.zip` da release mais recente que termina em [`-windows`](https://github.com/fabianoflorentino/vidctl/releases).
2. Extraia o zip e rode `vidctl.exe`.
3. O app exige ffmpeg e ffprobe no `PATH` (o vidctl avisa na primeira tela se estiverem faltando):

```powershell
winget install Gyan.FFmpeg
```

> Requer o WebView2 Runtime. No Windows 10/11 ele já vem instalado.

## macOS (Apple Silicon)

1. Baixe `vidctl-macos-arm64.zip` da release mais recente que termina em [`-macos`](https://github.com/fabianoflorentino/vidctl/releases).
2. Extraia e mova `vidctl.app` para a pasta Aplicativos.
3. Por não ter assinatura, o Gatekeeper pede abertura manual: clique com o botão direito no `vidctl.app` → **Abrir**.
4. Faltou o ffmpeg? `brew install ffmpeg` (e reinicie o app).

> Builds para Intel (x64) não são distribuídos; nesse caso, compile do fonte (seção abaixo).

## Linux

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install ./vidctl-linux-x64.deb
```

O `.deb` declara dependências de GTK, WebKitWeb 4.1 e **ffmpeg** — o `apt` resolve tudo.

### Fedora / RHEL

```bash
sudo dnf install ./vidctl-linux-x64.rpm
```

O `.rpm` instala GTK e WebKitGTK 4.1 automaticamente; o **ffmpeg** fica como recomendação — se não estiver no sistema, o app não abre vídeos. Instale com:

```bash
sudo dnf install ffmpeg     # pode exigir o repositório RPM Fusion
```

### Linux genérico (zip)

Sem gerenciador de pacotes:

```bash
unzip vidctl-linux-x64.zip -d vidctl
sudo install -m755 vidctl/vidctl /usr/local/bin/
sudo apt install ffmpeg    # ou equivalente da sua distro
```

## Arch Linux (estilo AUR, manual)

Não precisa de conta no AUR — o PKGBUILD vive no próprio repositório:

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
pkg/arch/install.sh
```

O script é o mesmo fluxo do `yay`: roda `makepkg -si` sobre
`pkg/aur/PKGBUILD` (baixa o tarball da tag, resolve e instala as dependências,
compila do fonte com `wails build -clean -tags webkit2_41` e instala o pacote).
Exige `base-devel`.

### Direto no makepkg (sem o script)

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl/pkg/aur
makepkg -si
```

- `-s` resolve e instala as dependências de build e runtime (pede sudo):
  `go`, `npm`, `gtk3`, `webkit2gtk-4.1`, `ffmpeg`
- `-i` instala o `.pkg.tar.zst` gerado ao final
- Quer pular as confirmações? `makepkg -si --noconfirm`

Para limpar os artefatos da compilação depois: `makepkg --cleanbuild` (ou
apague as pastas `src/` e os `*.pkg.tar.zst` dentro de `pkg/aur/`).

Quando o pacote estiver publicado no AUR, o mesmo resultado vem de
`yay -S vidctl` (ou `paru -S vidctl`).

Quer uma versão mais nova? Atualize o PKGBUILD antes de instalar:

```bash
cd vidctl
git pull
pkg/aur/update-aur.sh 2.0.0   # use a versão da última release
pkg/arch/install.sh
```

## Verificar integridade (checksums)

Antes de instalar, confira os arquivos contra o `checksums.txt` da release
(baixado junto):

```bash
sha256sum -c checksums.txt       # Linux / macOS
```

No Windows (PowerShell):

```powershell
Get-FileHash vidctl-linux-x64.zip -Algorithm SHA256
# compare com a linha correspondente do checksums.txt
```

## ffmpeg, o único requisito de execução

O app é 100% offline e não precisa de internet; o único programa externo é o
**ffmpeg** (junto com o `ffprobe`). Se o vidctl abrir e apontar que faltam,
instale pela sua distro:

| Sistema | Comando |
|---|---|
| Debian/Ubuntu | `sudo apt install ffmpeg` |
| Fedora/RHEL | `sudo dnf install ffmpeg` |
| Arch | `sudo pacman -S ffmpeg` |
| macOS | `brew install ffmpeg` |
| Windows | `winget install Gyan.FFmpeg` |

## Compilar do fonte (opcional)

```bash
git clone https://github.com/fabianoflorentino/vidctl
cd vidctl
wails build -clean -tags webkit2_41    # requisitos See README
```

O binário sai em `build/bin/vidctl`.