# Instalação

O vidctl é distribuído pelo repositório de [releases](https://github.com/fabianoflorentino/vidctl/releases).
Cada release traz **todos** os pacotes de uma vez:

| Arquivo | Plataforma |
|---|---|
| `vidctl-linux-x64.deb` | Debian / Ubuntu / derivados (64-bit) |
| `vidctl-linux-x64.rpm` | Fedora / RHEL / derivados (x86_64) |
| `vidctl-linux-x64.zip` | Linux genérico |
| `vidctl-macos-arm64.zip` | macOS Apple Silicon (M1+) |
| `vidctl-windows-x64.zip` | Windows 64-bit |
| `checksums.txt` | Verificação de integridade |

Sempre use a versão mais recente: <https://github.com/fabianoflorentino/vidctl/releases/latest>

---

## Windows

1. Baixe `vidctl-windows-x64.zip` da [última release](https://github.com/fabianoflorentino/vidctl/releases/latest).
2. Extraia o zip e rode `vidctl.exe`.
3. O app exige ffmpeg e ffprobe no `PATH` (o vidctl avisa na primeira tela se estiverem faltando):

```powershell
winget install Gyan.FFmpeg
```

> Requer o WebView2 Runtime. No Windows 10/11 ele já vem instalado.

## macOS (Apple Silicon)

1. Baixe `vidctl-macos-arm64.zip` da [última release](https://github.com/fabianoflorentino/vidctl/releases/latest).
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

Quer uma versão mais nova? Atualize o PKGBUILD antes de instalar:

```bash
cd vidctl
git pull
pkg/aur/update-aur.sh 1.0.8   # use a versão da última release
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