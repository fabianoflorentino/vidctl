# Changelog

**Português (Brasil) · [English](docs/CHANGELOG.en.md) · [Español](docs/CHANGELOG.es.md) ·
[中文（简体）](docs/CHANGELOG.zh-CN.md)**

Todas as mudanças relevantes do vidctl, no formato
[Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), com
versionamento [SemVer](https://semver.org/lang/pt-BR/) e o release por
plataforma conforme o workflow de publicação.

As releases por plataforma (`vX.Y.Z-windows`, `vX.Y.Z-linux`, `vX.Y.Z-macos`)
compartilham este mesmo changelog: a diferença entre elas é apenas o sistema
operacional dos pacotes.

## [Unreleased]

- **Nova marca**: o ícone do app e a marca do site passaram de uma câmera de
  vídeo para uma **lente central** com íris de abertura, no azul da identidade —
  nos instaladores, no favicon, na marca do site e no estado vazio da tela
  inicial.

## [2.10.0] — 2026-10-18

- **Tela principal mais limpa**: o rodapé saiu e a barra superior ficou só com o
  botão de **opções** no canto direito (sem o nome "vidctl" no centro). O status
  do **ffmpeg** passou para dentro do menu, com indicador verde. A área de
  arrastar-e-soltar (estado vazio) foi para dentro de um painel com o mesmo
  design da barra lateral. O título da janela passou de "vidctl — video
  compressor" para **"video controller"**.

## [2.9.1] — 2026-10-18

- **Corrigido — verificação de atualização**: os pacotes do **AUR** e do
  **RPM** compilavam sem embutir a versão (o binário ficava `dev`), então a
  checagem concluía que o app estava atualizado. Agora o `PKGBUILD` e o spec do
  RPM passam `-ldflags -X ...internal/version.Version`, como o `.deb` e os
  `.zip`. E, quando a versão local não puder ser determinada, o app mostra
  **"versão local desconhecida"** em vez de afirmar que está atualizado.

## [2.9.0] — 2026-10-18

- **Nova identidade visual**: o azul de destaque do app saiu do tom lavado para
  um **azul vívido** — `#3b82f6` no tema escuro e `#2563eb` no claro — deixando
  a interface mais vibrante e profissional. O site institucional ganhou o mesmo
  tratamento (fundo navy escuro, Inter + JetBrains Mono, animações de entrada e
  a marca com o ícone do app + "vidctl").

## [2.8.0] — 2026-10-17

- **Transcrição local** (fase 12): pelo menu do vídeo ("transcrever vídeo…") o
  app gera legendas `.srt`/`.vtt` e texto `.txt` usando um motor de
  speech-to-text **offline** (whisper.cpp/`whisper-cli` ou vosk), com `auto`
  detectando o que estiver instalado. Áudio extraído para PCM 16 kHz mono,
  idioma `auto`/`pt`/`en`, formato `all`/`srt`/`vtt`/`txt`, timestamps por
  palavra e janela `mm:ss`. Tudo roda local — nada é enviado para a nuvem;
  sem motor instalado o app orienta a instalação. Entra na mesma fila
  (`Task.Kind = transcribe`), com progresso, cancelamento e "abrir pasta".

- **Corrigido — corte por tempo**: removido o mínimo de 1 min por parte. A
  sobra agora vira a última parte mesmo quando tem menos de 1 min (ex.: vídeo de
  1:49 → `1:00` + `0:49`), em vez de bloquear o corte. O limite segue em 60
  partes por vídeo.

## [2.7.0] — 2026-10-14

- **Presets personalizados** (fase 9): crie, edite e exclua seus próprios
  presets além dos fixos — nome, modo (tamanho ou CRF), bitrate de áudio, codec
  (H.264/H.265) e codificador (software/NVENC/AMF/QSV/VideoToolbox). Salvos em
  `presets.json` na pasta de configuração (escrita atômica), entram na seção
  Presets e comprimem como qualquer outro, com IDs `custom-` e validação dos
  campos.

## [2.6.0] — 2026-10-13

- **Verificação de atualização** (fase 8): ao abrir, o app consulta a release
  mais recente do GitHub (`internal/update`, timeout de 5 s e cache por sessão)
  e mostra o badge "vX.Y.Z disponível" na barra de cima quando há versão nova —
  clicar abre a release no navegador. Um botão de setas reexecuta a checagem;
  offline e erros de rede são silenciosos. A versão do binário agora é embutida
  via `-ldflags` (Makefile e workflow de release).

## [2.5.0] — 2026-10-12

- **Notificações do sistema** (fase 7): aviso do desktop quando uma tarefa
  termina ou falha (`notifyOnDone`) e a opção de revelar a pasta do arquivo de
  saída ao concluir (`openFolderOnDone`), configuráveis na nova seção
  "Notificações" das Preferências. O backend entrega via `beeep` (toast no
  Windows, central de notificações no macOS, `notify-send` no Linux) e abre a
  pasta com o gerenciador de arquivos do sistema; notificações nunca afetam o
  resultado da conversão.
- **Organização do código**: a fachada de bindings do Wails foi movida para
  `internal/app` (a raiz guarda só o `main` e um adaptador fino que mantém o
  namespace `main.App` do front) e a detecção de tema do sistema para
  `internal/systemtheme`; os `README`/`CHANGELOG` em inglês, espanhol e
  chinês foram movidos para `docs/`, com links e a árvore do projeto
  atualizados.

## [2.4.0] — 2026-10-12

- **i18n (pt-BR/en/es/zh)**: a interface ganhou dicionários em português
  (Brasil, padrão), inglês, espanhol e chinês simplificado (mandarim). O idioma
  é escolhido nas Preferências e fica salvo entre execuções (`config.Language`).
- **Site e documentação em 4 idiomas**: a landing page ganhou um seletor de
  idioma (PT/EN/ES/中文); `README`, `CHANGELOG` e o guia de instalação têm
  versões em pt-BR, en, es e zh-CN. O plano de implementação interno permaneceu
  em pt (é descartável).
- **Base do código em inglês**: comentários (padrão `go doc`), mensagens de
  usuário, logs e rótulos de teste foram anglicizados no backend (Go) e no
  frontend (Svelte/TS); os dicionários de UI são a única fonte de texto em pt.

## [2.3.0] — 2026-10-11

- **Extrair áudio de um vídeo** (fase 6): a nova seção **"Extrair áudio"** da
  barra lateral extrai a faixa sem reencodar o vídeo, escolhendo formato (mp3,
  opus, aac, flac, wav ou copiar — sinalização de stream, só remux), qualidade
  (kbps), canais (mono/estéreo), normalização EBU R128 (loudnorm) e uma janela
  de tempo (mm:ss). A extração entra na mesma fila de tarefas, com progresso,
  cancelamento e abrir pasta ao concluir. O sufixo do arquivo de saída é
  configurável nas Preferências (`-audio` por padrão).
- **Nomes da barra lateral**: "Cortar em partes" virou **"Cortar"** e "Ajustes
  por arquivo" virou **"Ajustes de vídeo"**, deixando claro o que pertence a
  áudio (extração) e o que pertence ao vídeo (escala, corte, FPS, rotação…)
- **Um vídeo por vez**: "Adicionar vídeos…" virou **"Trocar vídeo…"** — trocar
  o vídeo substitui a seleção, **reseta as opções por arquivo** ao padrão e
  não enfileira mais lote (o app processa um trabalho por vez, sequencial).
- **Fila de conversões → Fila de tarefas**: a fila passou a listar qualquer
  trabalho (comprimir ou extrair áudio), com contador de tarefas.
- **Menos redundância**: com o vídeo carregado, a área de destino (drop-target)
  no topo do conteúdo foi removida — a troca fica no botão **"Trocar vídeo…"**
  ou soltando um novo arquivo em qualquer lugar da janela; a área só permanece
  na tela vazia.
- **Tipografia maior**: toda a interface da janela subiu **1px** em cada
  tamanho de fonte (a hierarquia se mantém; os textos densos do painel e da
  fila ficam mais confortáveis de ler).
- **Botão "extrair áudio" no cartão do vídeo**: as opções de áudio continuam na
  sidebar, mas o botão de extrair agora vive num cartão próprio abaixo do
  destino do vídeo (mesma aparência do "salvar como…"), com o caminho do áudio
  mostrando a extensão do formato selecionado (`.mp3`, `.flac`, `.wav`…) e um
  **"salvar como…"** para escolher outro destino.
- **Janela maior por padrão**: a janela abre em **1360×768** (antes 1000×700).
- **"Tamanho alvo" → "Tamanho"**: a seção da sidebar ficou só **Tamanho** e o
  rótulo do controle virou **megabytes** (sem a ambiguidade de MB = mega-bit).
- **Tema "sistema" fiel**: a opção de aparência "Sistema" agora lê o tema real
  do desktop (gsettings/GTK no Linux, `AppleInterfaceStyle` no macOS, registro
  no Windows) em vez de confiar no `prefers-color-scheme` do WebKit, que no
  Linux reportava "claro" mesmo com o sistema escuro.

## [2.2.1] — 2026-10-11

- Ícone unificado nos instaladores: o mesmo desenho do favicon (vídeo/play com
  gradiente) nos pacotes de todas as plataformas (a v2.2.0 ainda carregava o
  ícone anterior por ter sido compilada antes do merge do #34).

## [2.2.0] — 2026-10-11

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
- **Sidebar colapsável**: os cartões de configuração (`Tamanho alvo`/`Qualidade`,
  `Avançado`, `Presets`, `Cortar em partes`, `Ajustes por arquivo`) colapsam e
  expandem pelo cabeçalho (chevron, clique ou teclado). `Avançado`, `Cortar em
  partes` e `Ajustes` vêm **fechados** por padrão, reduzindo a poluição visual;
  o estado vale para a sessão.
- **Tempo total de conversão**: cada trabalho da fila mostra o tempo decorrido
  em tempo real durante o processamento e o **total** ao concluir (card de
  progresso, linha da fila e card de resultado) — útil para comparar presets e
  encoders (software × GPU, NVENC `p1`–`p7`).
- **Arrastar e soltar na janela principal**: o conteúdo ganha uma área de
  destino (drop-target) com "arraste um vídeo aqui" + botão **"abrir vídeo…"**,
  visível inclusive com um vídeo já carregado — soltar um arquivo troca a
  entrada e vários enfileiram (reuso do `wails:file-drop`).
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

- A barra lateral virou um **painel arredondado** (estilo GNOME/panela flutuante):
  mesma cor do conteúdo, com contorno suave (`18px` de raio) e sombra sutil no
  lugar da coluna "quadrada" com linha cheia de divisão — a separação agora é
  por espaço e cantos arredondados, com sombra definida por tema (claro/escuro)
  e sem depender do sistema.
- A **scrollbar da barra lateral** é própria da UI: a nativa (GTK/WebKit) foi
  escondida — não vaza do painel nem cresce sobre o conteúdo ao passar o mouse.
  A barra custom (fina, `8px`, recuada) aparece ao rolar/dar hover/arrastar e
  some em ~0,8s parado, com o mesmo comportamento nas três plataformas.
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
- **Arrastar e soltar** funcionando nas três plataformas: o app não só escuta
  `wails:file-drop` como registra o `OnFileDrop` do runtime (que liga os
  listeners de drop no WebView). No **Linux** o handler nativo do Wails faz o
  WebKit navegar para o arquivo solto (o vídeo "toca" e a UI some), então lá o
  caminho é lido do próprio DOM (`text/uri-list`) sem navegação; Windows/macOS
  seguem com o drop nativo do Wails. A janela inteira é a área de drop e,
  durante o arrasto, um **pontilhado ao redor da tela** indica onde soltar.
- **Saída e thumbnail por padrão junto da fonte**: sem pasta de destino salva,
  os caminhos sugeridos iam para o diretório de trabalho do app (o arquivo da
  thumbnail "sumia" para quem procurava ao lado do vídeo). Agora o default é a
  pasta do vídeo de origem (ou a pasta configurada nas Preferências).
- A **thumbnail** gerada usa um frame **5 segundos após o início** do vídeo
  (primeiro segmento) em vez do meio do último trecho — nas divisões em partes
  caía perto do fim e parecia o "último frame"; e evita telas pretas de
  abertura. Em vídeos curtos (menos de 10s) cai para a metade.

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