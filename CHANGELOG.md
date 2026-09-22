# Changelog

Todas as mudanças relevantes do vidctl, no formato
[Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), com
versionamento [SemVer](https://semver.org/lang/pt-BR/) e o release por
plataforma conforme o workflow de publicação.

As releases por plataforma (`v1.x.y-windows`, `v1.x.y-linux`, `v1.x.y-macos`)
compartilham este mesmo changelog: a diferença entre elas é apenas o sistema
operacional dos pacotes.

## [Não publicado]

- Nada ainda.

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