# Changelog

Todas as mudanças relevantes do vidctl, no formato
[Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), com
versionamento [SemVer](https://semver.org/lang/pt-BR/) e o release por
plataforma conforme o workflow de publicação.

As releases por plataforma (`vX.Y.Z-windows`, `vX.Y.Z-linux`, `vX.Y.Z-macos`)
compartilham este mesmo changelog: a diferença entre elas é apenas o sistema
operacional dos pacotes.

## [Unreleased]

Persistência das preferências, fila de conversões em lote, estimativa de
tamanho, codecs de vídeo com aceleração de GPU e ajustes por arquivo: as
preferências sobrevivem ao fechamento, dá para comprimir vários vídeos de uma
vez, o tamanho esperado aparece antes de comprimir, o cartão **Avançado** troca
para H.265 ou para encoders de hardware (NVENC/QSV/AMF/VideoToolbox) e cada
vídeo aceita escala, corte, remoção de áudio, FPS, rotação e thumbnail.

### Adicionado

- **Estimativa de tamanho**: no modo tamanho, o painel "Tamanho alvo" mostra
  um badge com o tamanho esperado da saída — calculado com o mesmo orçamento
  de bits do encode 2-pass (agora único, em `internal/estimate`) — e a
  economia em relação ao arquivo original quando o alvo é menor. Novo binding
  `EstimateSize`; no modo CRF ainda não há previsão e o badge não aparece.
- **Codec H.265/x265**: o cartão "Avançado" do passo 02 permite escolher
  H.264 ou H.265 por trabalho; o preset continua com o codec padrão. Juntando
  a isso o 2-pass existente, o modo tamanho usa `libx265` com a mesma precisão
  do x264 (agora o builder de argumentos por encoder vive em
  `internal/encode`).
- **Aceleração de GPU (NVENC/QSV/AMF/VideoToolbox)**: hardware é opcional por
  trabalho, opt-in, e aparecem desabilitados os backends que o `ffmpeg` deste
  computador não tem ou cujo driver falhou no probe de 1 frame — novo binding
  `GetEncoders` com cache e re-detecção ("verificar de novo"). GPUs do modo
  tamanho usam VBR 1-pass (destino aproximado, ±5–10%), e a UI avisa quando o
  encoder de hardware está ativo; software continua o default.
- **Ajustes por arquivo**: novo cartão **"Ajustes por arquivo"** no passo 02
  com escala (padrão/original/personalizada), corte por janela em `mm:ss`,
  remoção de áudio, FPS, rotação (transpose) e geração de thumbnail ao final do
  encode. Builder de filtros em `internal/compress/adjust.go` com ordem fixa
  transpose→scale→fps, `-ss/-to` antes do `-i`, orçamento de tamanho e
  progresso baseados na duração cortada, validação de conflitos (corte manual
  × split, corte invertido, rotação/FPS fora da faixa) e novo binding
  `OpenThumbnailDialog`.
- **Preset NVENC (velocidade × qualidade)**: com o codificador NVIDIA ativo, o
  cartão **Avançado** ganha um seletor de preset `p1`–`p7` por trabalho
  (default `p4`, equilíbrio). `p1`/`p2` quase dobram a velocidade de encode na
  engine de vídeo (que já fica em 100%), com leve perda de eficiência de
  compressão no modo tamanho; `p7` prioriza a qualidade. Encode de hardware não
  usa a engine gráfica — o "GPU %" do painel reflete o máximo entre gráficos e
  encoder.
- **Modo de depuração local**: `VIDCTL_DEBUG=1 ./build/bin/vidctl` liga um log
  no terminal com as transições da fila, os eventos emitidos/recebidos (e os
  patches descartados por estado), as chamadas `Compress`/`GetTasks` e as
  exceções de JS do frontend — para diagnosticar a UI de progresso sem
  afetar a execução normal (desligado por padrão).
- **Fila de conversões (batch)**: "Adicionar vídeos…" enfileira vários
  arquivos de uma vez (ou solte vários na janela), um trabalho por arquivo com
  o preset e a saída atuais; a fila aparece no lugar do passo 04 com nome,
  preset, status, barra de progresso do item em execução, cancelamento por
  item e "limpar fila". Eventos novos `compress:queued`/`compress:start` e
  roteamento por `JobID` no frontend: um erro em um item não derruba os
  demais. Bindings novos: `CompressMultiple`, `GetTasks` e `ClearFinished`.
- **Configuração persistente**: preset, tamanho em MB, CRF e pasta de destino
  voltam como estavam no próximo boot. Arquivo em
  `<config-dir>/vidctl/config.json` (Linux `~/.config`, macOS
  `~/Library/Application Support`, Windows `%AppData%`), escrito de forma
  atômica para não corromper em caso de queda. Arquivo ausente ou inválido abre
  o app com os padrões em vez de falhar.
- **Preferências → Binários**: campos para apontar o `ffmpeg` e o `ffprobe` para
  fora do PATH. O caminho configurado tem prioridade sobre o PATH, vale na hora
  (sem reiniciar) e um caminho inválido é reportado como erro em vez de cair
  silenciosamente no binário do PATH.
- **Preferências → Saída**: campo de pasta de destino; vazio mantém o
  comportamento de salvar ao lado do vídeo original.

### Mudado

- O card de progresso acompanha o item selecionado na fila e o cancelamento
  passa a ser por item — a ação não para mais o trabalho inteiro.
- `maxParallel` passa a valer: define quantos vídeos são comprimidos ao mesmo
  tempo (padrão **1** = sequencial), aplicado na inicialização e ao salvar as
  Preferências.
- `CheckFFmpeg`, `GetMediaInfo` e a miniatura agora respeitam o caminho de
  binário configurado, com mensagem separada para "não está no PATH" e "o
  caminho que você configurou não funciona".
- Campos `language`, `notifyOnDone` e `openFolderOnDone` já são preservados no
  arquivo, mas só passam a ter efeito quando os recursos correspondentes
  existirem (fases futuras do plano).
- O modo de depuração (`VIDCTL_DEBUG=1`) agora rastreia os **comandos externos
  e a decisão do pipeline**: o comando `ffmpeg` completo (2-pass, single-pass e
  thumbnail), a escolha de encoder/hardware por segmento (`encoder=hevc_nvenc`,
  2-pass, cadeia `-vf`, seek, remoção de áudio, FPS, rotação), o `ffprobe` e a
  miniatura (`[media]`), e a listagem/probe de encoders (`[encode]`, com o
  motivo quando um backend falha). Cada comando é reproduzível linha a linha.

### Corrigido

- A detecção de encoders não trava mais a seleção de arquivo quando um backend
  é listado pelo `ffmpeg` mas falha no probe de 1 frame (ex.: `h264_amf` sem
  GPU AMD): `codecs` vinha `null` no JSON e o frontend quebrava ao montar a
  lista de codificadores. Agora a detecção devolve sempre uma lista (vazia
  quando não há codecs) e o frontend também tolera `null`.
- O **"GPU %"** do painel de progresso media apenas `utilization.gpu`
  (gráficos/compute), que fica ~0 durante um encode NVENC — o encode de vídeo
  carrega a engine de **encoder** (`utilization.encoder`), não a barra de
  gráficos. O coletor agora reporta o **máximo entre as duas**, então o painel
  reflete o encode de hardware de verdade.
- O probe de encoders de hardware passou a usar um frame `256×256`: o `64×64`
  anterior estava **abaixo do tamanho mínimo** do NVENC ("Frame Dimension less
  than the minimum supported value"), então GPUs NVIDIA modernas apareciam como
  "detectado mas não rodou" mesmo com driver funcionando.
- Backends por plataforma: **AMF (AMD)** só é oferecido no Windows e
  **VideoToolbox** só no macOS; NVENC e QSV continuam sendo detectados em
  qualquer SO e **Software**/"Automático" seguem sempre disponíveis. Além disso,
  o motivo do probe falho agora aparece na UI (cartão **Avançado**) em vez de um
  texto genérico.

## [2.1.0] — 2026-09-27

Redesign visual no padrão Constrict com drag and drop, miniaturas e ajuste
fino dos contadores.

### Adicionado

- **Visual no padrão Constrict**: tema escuro fosco (tokens recentes com
  accent slate), tema claro derivado e ícone próprio nos instaladores de
  Linux/Windows/macOS; layout de duas colunas (sidebar de ajustes + área do
  vídeo), cards por grupo, e o modal de Preferências (tema sistema/claro/escuro
  e recheck do ffmpeg).
- **Drag and drop**: solte um vídeo na janela para selecioná-lo; a área vazia
  fica destacada em hover (`--wails-drop-target`, evento `wails:file-drop`).
- **Miniaturas**: a linha do vídeo mostra um frame extraído via ffmpeg
  (procurando em `min(1s, dur/2)`, limitado a 320px), cacheado por
  path+size+mtime em `os.TempDir()/vidctl-thumbs`; novo binding
  `GetThumbnail`.
- **Contadores com valor editável e ajuste fino**: clique no número para
  digitar (Enter/clique fora confirma, Esc cancela, vírgula aceita) e segure
  **Shift** nas setas `+/−` (ou `↑/↓`) para passo fino de 0,1.
- **min/parte fracionário de ponta a ponta**: `split.Spec.MinutesEach` aceita
  fração (`1.5` = 1 m 30 s) do frontend ao corte, com camada de `internal/split`
  testada.
- **Menu da linha do vídeo (⋮)**: trocar arquivo / limpar.
- Redesign do ícone do app: fundo escuro arredondado com glyph de compressão
  no accent slate (appicon.png, icon.ico e pkg/icon do Linux).

### Mudado

- Fase 4 do plano Constrict entregue: componentes base (Stepper, Toggle,
  RadioCardGroup, Modal, InfoTip, KebabMenu) sem estilos scoped, movidos
  para o design system global.
- `internal/media` agora também gera miniaturas; documentação das
  Preferências, pipeline e estrutura atualizadas no README.

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