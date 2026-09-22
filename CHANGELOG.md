# Changelog

Todas as mudanças relevantes do vidctl, no formato
[Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), com
versionamento [SemVer](https://semver.org/lang/pt-BR/) e o release por
plataforma conforme o workflow de publicação.

As releases por plataforma (`vX.Y.Z-windows`, `vX.Y.Z-linux`, `vX.Y.Z-macos`)
compartilham este mesmo changelog: a diferença entre elas é apenas o sistema
operacional dos pacotes.

## [2.0.1] — 2026-09-22

Retoca a experiência da v2.0.0: qualidade assistida e ajustes de interface.

### Adicionado

- **Aviso de qualidade pré-encode**: estimativa do bitrate de vídeo (por parte
  quando há corte) comparada a um mínimo por faixa de resolução; quando baixo,
  a sidebar mostra o motivo e uma sugestão aplicável com um clique — subir o
  alvo em MB, cortar em partes de X min ou migrar para preset CRF.
- Card de **resumo do trabalho** no espaço livre abaixo do vídeo quando
  ocioso: destino + alvo, plano de corte e bitrate estimado vs. mínimo — o
  mesmo lugar que o card de progresso ocupa durante o processamento.
- Landing page da era v2 no GitHub Pages, com demo animado fiel à UI real
  (escolha de preset → corte → processamento → tema claro).

### Mudado

- Sidebar: o grupo de escolha de preset passou de "Destino" para **"Presets"**.
- Sidebar mais larga (300→344px) para acomodar os banners de corte/qualidade.
- Release passou a ser 100% intencional: push em `main` não dispara mais o
  workflow (só `gh workflow run release.yml --ref vX.Y.Z` em uma tag).

### Corrigido

- Descrição dos presets na sidebar agora quebra linha e mostra o texto
  completo (antes aparecia cortada com "…" em Instagram/YouTube).

- Descrição dos presets na sidebar agora quebra linha e mostra o texto
  completo (antes aparecia cortada com "…" em Instagram/YouTube).
- Sidebar mais larga (300→344px) e card de **resumo do trabalho** no espaço
  livre abaixo do vídeo quando ocioso: destino + alvo, plano de corte e bitrate
  estimado vs. mínimo — o mesmo lugar que o card de progresso ocupa durante o
  processamento.
- **Aviso de qualidade pré-encode**: estimativa do bitrate de vídeo (por parte
  quando há corte) comparada a um mínimo por faixa de resolução; quando baixo,
  a sidebar mostra o motivo e uma sugestão aplicável com um clique — subir o
  alvo em MB, cortar em partes de X min ou migrar para preset CRF.
- Sidebar: o grupo de escolha de preset passou de "Destino" para **"Presets"**.
- Release passou a ser 100% intencional: push em `main` não dispara mais o
  workflow (só `gh workflow run release.yml --ref vX.Y.Z` em uma tag).

## [2.0.0] — 2026-09-22

Primeira grande revisão da interface + corte de vídeo por tempo.

### Adicionado

- **Corte em partes (split)**: divide o vídeo por tempo — N partes iguais ou
  X minutos por parte (ex.: 30 min → 6 × 5 min) — com mínimo de 1 min por
  parte e teto de 60 partes; cada parte passa pela compressão escolhida e sai
  como `nome-partN.mp4` (`internal/split` + `Job.Split` no pipeline).
- **Redesign da interface no estilo Constrict** (branch `feat/ui-constrict`):
  layout full-height com barra lateral de controles (Destino, Qualidade/
  Tamanho alvo, Cortar em partes, Saída), estado vazio estilo StatusPage,
  linha do vídeo como card com miniatura/statu/PIE de progresso e barra de
  ações inferior.
- **Tema claro/escuro manual**: botão sol/lua no topo; segue o sistema por
  padrão e a escolha fica persistida entre execuções.
- **Painel de progresso enriquecido**: card próprio abaixo do vídeo (mesma
  largura e raio) com percentual agregado por parte e leitura ao vivo de
  CPU, memória, % do ffmpeg e GPU (NVIDIA quando disponível) — novo
  `internal/sysinfo` via polling 1s (`GetUsage`).
- `OpenMultipleDialog` no backend para multi-seleção de arquivos (base da
  futura fila batch).
- Roadmap: Fase 11 (corte em segmentos) e plano de UI `docs/plano-ui-constrict.md`.

### Corrigido

- Rodapé/ação com faixa vazia gigante em janelas altas (layout agora ancora
  `html/body/#app` em 100% com coluna flex).
- Scroll horizontal indesejado na barra lateral (min-width de flex children
  + overflow-x).
- Ícone do estado vazio (troca de "pilha" por câmera de vídeo).

## [1.0.17] — 2026-09-13

Primeira versão **production ready** do vidctl.

### Adicionado

- Compressão de vídeo com presets por plataforma: WhatsApp Status (10 MB),
  WhatsApp (16 MB), Instagram (16 MB), Shorts (30 MB), YouTube (CRF) ou um
  tamanho livre definido pelo usuário (2–100 MB).
- Encode ffmpeg **2-pass** para entregar o tamanho máximo sem estourar o
  limite escolhido.
- Progresso em tempo real por etapa (pass 1/2 e pass 2/2) com cancelamento a
  qualquer momento.
- Verificação de `ffmpeg`/`ffprobe` na inicialização, com o botão "verificar
  de novo" que relê o PATH do sistema — permite instalar o ffmpeg com o app
  aberto sem reiniciar.
- Desktop nativo para **Linux, Windows e macOS** (Wails v2 + WebKit/WebView2).
- Modo offline: fontes e interface embutidas no binário.
- Pacotes por plataforma (.zip, .deb e .rpm) com `checksums.txt` publicado e
  conferido em todas as releases.
- Escala padrão com largura máxima de 1280 px preservando o aspecto do vídeo.

### Corrigido

- Windows: conversões e sondagens de arquivo não abrem mais janelas de
  console (`CREATE_NO_WINDOW`).
- Windows: "verificar de novo" detecta um ffmpeg/ffprobe recém-instalado sem
  reiniciar o app (releitura do PATH de usuário/máquina no registro).
- Pacotes .zip de release não incluíam mais a árvore inteira de recursos do
  projeto.

### Especificações

- Go 1.25+, Wails v2.15, Svelte 5, ffmpeg/ffprobe requeridos no sistema.
- Cada release publica apenas os pacotes do seu sistema operacional
  (Windows: `.zip`; Linux: `.zip`, `.deb`, `.rpm`; macOS: `.zip` com
  `vidctl.app`).

[1.0.17]: https://github.com/fabianoflorentino/vidctl/releases/tag/v1.0.17