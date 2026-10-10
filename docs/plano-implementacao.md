# Plano de implementação — novas funcionalidades

Documento de plano para a próxima rodada de funcionalidades do vidctl.
Baseado no estado atual do código (Go 1.26 + Wails v2.16 + Svelte 5, um único
componente `App.svelte`).

> Regras do projeto (AGENTS.md) que valem para **todas** as fases:
> - toda mudança de código vem com testes cobrindo o comportamento;
> - rodar sempre `go test ./...` e manter cobertura ≥ 80% (gate no CI);
> - comportamento novo deve refletir no `README.md` e, quando aplicável, em `docs/`;
> - sem comentários em código a menos que realmente necessários;
> - **uma branch por fase**, com PR para a `main` — ver
>   [Fluxo de execução](#fluxo-de-execução).

### Fluxo de execução

Cada fase é entregue como uma unidade revisável e isolada:

1. **Branch nova a partir da `main` atual**, nunca a partir de outra branch de
   fase — assim o PR não carrega implementações anteriores junto.
   Naming: `feat/<slug>` (`feat/fase1`, `feat/audio`, `feat/transcricao`),
   `fix/<slug>` ou `chore/<slug>`.
2. **Commits na branch da fase**, em Conventional Commits e em português, um
   por unidade lógica de mudança. Cada commit compila e passa nos testes
   sozinho, o que mantém o histórico legível no `git bisect`.
3. **Docs no mesmo PR**: esta seção do plano e o `CHANGELOG.md` (`[Unreleased]`)
   são atualizados junto com o código, para o plano nunca divergir do repo.
4. **PR para a `main`** aberto como sinal de "fase entregue". Merge direto na
   `main` e push de branch só acontecem com pedido explícito.

Branch            | Fase | PR
---|---|---
`feat/fase1`      | 1 — Configuração persistente | aberto
`feat/fila`       | 2 — Fila de conversões | aberto
`feat/estimativa` | 3 — Estimativa de tamanho | mergeado
`feat/hevc`       | 4 — HEVC/x265 + GPU (hardware) | aberto
`feat/ajustes`    | 5 — Ajustes por arquivo | pendente
`feat/audio`      | 6 — Extração de áudio | pendente
`feat/notificacao`| 7 — Notificação + abrir pasta | pendente
`feat/updates`    | 8 — Verificação de atualização | pendente
`feat/presets`    | 9 — Presets personalizados | pendente
`feat/i18n`       | 10 — i18n pt/en | pendente
`feat/split-tempo`| 11 — Corte por tempo | pendente
`feat/transcricao`| 12 — Transcrição de vídeo | pendente

---

## Síntese das fases (ordem recomendada)

| Fase | Funcionalidade | Doc | Esforço |
|---|---|---|---|
| 1 | Configuração persistente + preset "último usado" | §1 | M |
| 2 | Fila de conversões (batch) | §2 | G |
| 3 | Estimativa de tamanho antes do encode | §3 | M |
| 4 | Codec HEVC/x265 + aceleração de GPU (NVENC/AMF/QSV/VideoToolbox) | §4 | G |
| 5 | Ajustes por arquivo (escala, corte, áudio, FPS, thumbnail) | §5 | G |
| 6 | Extração de áudio com opções (mp3/opus/aac/flac/wav/copiar) | §6 | M |
| 7 | Notificação ao concluir + abrir pasta | §7 | P |
| 8 | Verificação de atualização no app | §8 | M |
| 9 | Presets personalizados pelo usuário | §9 | M |
| 10 | i18n (pt/en) — opcional | §10 | M |
| 11 | Corte em segmentos por tempo (split) | §11 | G |
| 12 | Transcrição de vídeo (legendas srt/vtt + texto) | §12 | G |

Redesign da UI no estilo do Constrict (fila visual, drag&drop, tema adwaita)
está detalhado em [`plano-ui-constrict.md`](./plano-ui-constrict.md); ele consome
as Fases 1, 2, 3, 4 e 10 deste plano.

Ordem por dependência: config (1) libera fila (2, `MaxParallel`), notificações
(7), presets custom (9). Estimativa (3) e codec/hw (4) são independentes entre
si, mas o job enriquecido da fase 4/5 deve ser desenhado junto com a fila (2)
para o contrato `Job` não quebrar duas vezes.

Áudio derivado (6 extração, 12 transcrição) compartilham a mesma base: extração
da faixa em PCM com `ffmpeg`, `Task.Kind` na fila (2) e trim `-ss/-to` da fase
5. As duas fases entram **depois** da fila (2) e da config (1).

---

## Fase 1 — Configuração persistente + preset "último usado"

**Objetivo:** introduzir a primeira camada de persistência do app e restaurar o
preset/tamanho/saída no próximo boot. Hoje não existe nenhuma persistência:
`sizeMB`/`crf` e caminhos vivem só em memória (`App.svelte`).

> **Status: implementado** em `feat/fase1` (PR aberto para a `main`). Os quatro
> campos sem efeito hoje (`Language`, `MaxParallel`, `NotifyOnDone`,
> `OpenFolderOnDone`) já são preservados em disco para as fases 2, 7 e 10.

### Backend
- Novo pacote `internal/config`:
  - `type Config struct`: `PresetID, SizeMB, CRF, OutputDir, FFmpegPath,
    FFprobePath, Language, MaxParallel, NotifyOnDone, OpenFolderOnDone`.
  - `Load() (Config, error)` com `Defaults()` aplicados; arquivo em
    `os.UserConfigDir()/vidctl/config.json` (diretório sobreescrevível por
    variável de pacote para os testes).
  - `Save(Config) error` com escrita atômica (tmp + rename) para não corromper
    o arquivo em crash.
  - Arquivo ausente/corrompido → defaults sem erro.
- `app.go`: novos métodos ligados ao frontend `GetConfig() (config.Config)` e
  `SaveConfig(cfg config.Config) error`, seguindo o padrão
  `CheckFFmpeg`/`GetPresets` (app.go:46-76).
- Override de caminho do ffmpeg: `internal/media/ffprobePath()` (media.go:56) e
  `internal/compress/ffmpegPath()` passam a consultar um override
  configurado uma vez no `startup` (verificar `os.Stat`; validar na
  `CheckFFmpeg`). Se vazio, manter `cmdutil.LookPath` atual — preservando o
  merge de PATH do registro no Windows.

### Frontend
- `App.svelte`:
  - No `onMount`, carregar `GetConfig()` e restaurar `selectedPresetId`,
    `sizeMB`, `crf` e o diretório de saída sugerido.
  - Em toda mudança de preset/tamanho/saída, salvar com debounce
    (`SaveConfig`) — vale também para a fase 10 (i18n).
  - Ícone de engrenagem no masthead (perto do pill de ffmpeg, App.svelte:191-200)
    abrindo um painel: caminhos ffmpeg/ffprobe custom, pasta de saída padrão,
    idioma (fase 10), "notificar ao concluir" (fase 7), "abrir pasta ao
    concluir" (fase 7), paralelismo (fase 2 — aparece quando existir).

### Testes
- `internal/config`: defaults, roundtrip Load/Save, arquivo corrompido →
  defaults, escrita atômica deixa o arquivo válido.
- `app_test.go`: `GetConfig`/`SaveConfig` com diretório temporário.

### Docs
- README: seção "Configuração" (local do arquivo, campos). Renomear as
  funcionalidades atuais para incluir persistência.

---

## Fase 2 — Fila de conversões (batch)

**Objetivo:** permitir arrastar/adicionar vários vídeos e comprimir em
sequência (e depois, opcionalmente, em paralelo). Hoje o modelo é de **um job
ativo** em todo o stack: `currentJobId` guarda todos os eventos
(App.svelte:46,51,59) e o `Manager` (compress.go:39-42) guarda somente jobs
ativos para cancelar.

> **Status: implementado** em `feat/fila` (PR aberto para a `main`). O
> `compress.Manager` virou uma fila (`Enqueue`/`Cancel`/`Remove`/
> `ClearFinished`/`List`/`SetMaxParallel`, estados `queued/running/done/error/
> canceled`), `maxParallel` vem do `config.MaxParallel` no boot e ao salvar as
> Preferências; eventos novos `compress:queued`/`compress:start` com
> `JobID`; `app.go` expõe `CompressMultiple`, `GetTasks`, `ClearFinished` e
> `JobAck`; o frontend roteia eventos por `JobID` (buffer para eventos que
> chegam antes do ack) e o passo 04 virou a lista de fila com cancelar por
> item e limpar fila.

### Modelo de tarefas
- Um item da fila é uma `Task` genérica para acomodar compressão (só esta fase)
  e, futuramente, extração de áudio (fase 6):
  - `type Task struct { Kind, JobID, Label string; Job compress.Job }`.
- Contrato `Job` precisa ganhar campos novos (codec/ajustes) nas fases 4/5 —
  desenhar a fila com `Job` extensível desde já.

### Backend
- Estender `compress.Manager`:
  - `queue []Task`, `active *runningJob`, `maxParallel` (default 1, vindo do
    `config.MaxParallel` na fase 1).
  - `Enqueue(task) (jobID, position)` — retorna imediatamente; dispara o
    worker quando a fila vai de 0→1.
  - `Cancel(id)`: remove da fila se pendente, senão cancela o ativo.
  - `Remove(id)`/limpeza ao concluir; `List() []TaskStatus` para o frontend
    repintar o estado após refresh.
  - Concorrência: worker executa até `maxParallel` tasks; cancelamento tratado
    com o contexto cancelável já existente. Progresso de cada job continua vindo dos
    eventos por `JobID`.
- `internal/events`: novos eventos
  - `compress:queued` → `QueuedEvent{JobID, Position}` (emitido no enqueue),
  - `compress:start` → `StartEvent{JobID}` (emitido quando o worker inicia),
  - `compress:progress/done/error` permanecem (já são por `JobID`).
- `app.go`: `Compress(req)` deixa de lançar goroutine avulsa e passa a chamar
  `manager.Enqueue`. Novo `CompressMultiple([]compress.Job) []JobAck`.

### Frontend
- `App.svelte`:
  - Novo estado `queue: $state<QueueItem[]>`; eventos passam a ser roteados por
    `JobID` (Map jobID→item) em vez de filtrados por `currentJobId`.
  - Passo 04 vira lista: item mostra nome do arquivo, preset, status
    (`aguardando`/`processando`/`concluído`/`erro`), barra de progresso do item
    ativo, botão cancelar por item, botão "limpar fila".
  - "Adicionar vídeo(s)" multipick já existe como `OpenMultipleDialog`
    (app.go:179) e devolve `[]string`; a fase 2 só precisa consumir o slice e
    criar um item de fila por arquivo.
- Garantir que a UI desabilite "adicionar" apenas se `!ffmpegOk`, nunca por
  `running` (multi-job).

### Testes
- `compress.Manager`: enfileira→executa em ordem, cancelar pendente remove da
  fila sem rodar, cancelar ativo interrompe, `List` reflete estados.
- `events`: payloads novos e roteamento.
- `app_test.go`: `CompressMultiple` e ack com posições.
- Integração: 2 arquivos → 2 `compress:done` (fakes sh, `skipOnWindows`).

### Docs
- README: seção "Fila de conversões" (limite default 1, ajuste em configuração).

---

## Fase 3 — Estimativa de tamanho antes do encode

> **Status: ✅ implementada** — ver `feat/estimativa`. Decisão: a assinatura
> final do binding é `EstimateSize(job compress.Job)`, reutilizando o mesmo
> objeto do `GetAdvice` (preset efetivo + override de tamanho + probe), em vez
> da assinatura `EstimateSize(inputPath, presetID, sizeMB float64)` descrita
> abaixo. O orçamento de bits vive agora em `internal/estimate` e é a fonte
> única do encode 2-pass e da previsão.

**Objetivo:** prever o tamanho final (e a economia) antes de rodar o 2-pass.
Hoje a única estimativa é uma fórmula duplicada no frontend
(App.svelte:296-304 — hint de bitrate) que repete `computeSizeBudget`
(compress.go:103-131).

### Backend
- Extrair o orçamento de bits em função exportável no pacote
  `internal/estimate` (nova função que recebe `media.Info` + `presets.Preset` +
  `sizeMB`/`crf`), e fazer `compress.computeSizeBudget` consumir a mesma fonte
  (sem duplicação).
- Nível 1 (determinístico, imediato): para modo `size`, o tamanho alvo é o
  orçamento já calculado — retornar `TargetSizeMB` (já com headroom de 5%),
  `CurrentSizeMB` e `SavedPct` (= 1 − target/current).
- Nível 2 (opcional/incremental): amostragem real de `N` segundos em pontos de
  chave e extrapolação por CRF/bitrate para uma previsão em modo `crf`.
  Marcar como *stretch goal* da fase — a UI não deve depender dele.
- `app.go`: método `EstimateSize(inputPath, presetID, sizeMB float64)
  (estimate.Result, error)` — reusa `media.Probe` + `presets.ByID` + override
  de tamanho (mesma lógica de compress.go:262-277).

### Frontend
- No passo 02 (linha do slider) e/ou passo 04 (perto do CTA): badge
  "esperado ≈ 15,2 MB (economia ~42%)" calculada com o resultado backend;
  atualizar ao trocar preset/tamanho/arquivo. Substituir a fórmula inline
  atual pela chamada ao backend.

### Testes
- `internal/estimate`: tabelas (com/sem áudio, floors de bitrate, headroom,
  `savedPct`), mesmo invariante do `computeSizeBudget` atual; teste de paridade
  estimativa × comportamento do `compress.Run`.
- `app_test.go`: `EstimateSize` com caminhos falsos.

### Docs
- README: explicar o que a estimativa garante (exata no modo size, heurística
  no modo crf).

---

## Fase 4 — Codec HEVC/x265 + aceleração de hardware (GPU)

> **Status: ✅ implementada (v1)** — ver `feat/hevc`. Cobre o builder por
> encoder (`internal/encode`), detecção com probe real e UI com aviso de GPU.
> Default continua software; hardware é opt-in por job. **Adiado para v2:** o
> cap de slots de GPU na fila (§ Interação com a fila) e persistir a preferência
> de hardware na config (§ Frontend).

**Objetivo:** além do libx264 atual (fixo em `buildSizePasses`/`buildCrfPass`,
compress.go:192,207 e 234), permitir x265 e encoders de GPU (NVENC/AMF/QSV/
VideoToolbox) com detecção de disponibilidade e fallback para software.

> **Por que vale:** trocar `libx264 -preset medium` pelo encoder de GPU dá
> tipicamente 2–10× de velocidade de encode e libera a CPU — o que também
> desengasga a fila (fase 2) quando `MaxParallel > 1`. O contraponto está em
> *Decisões abertas* §2 e §8: qualidade por bit e precisão do modo tamanho.

### O que muda no encoder

- `presets.Preset`: novos campos `Codec string` (`"h264"|"h265"`) e
  `Hardware string` (`""|"nvenc"|"qsv"|"amf"|"videotoolbox"|"auto"`).
- Novo `internal/encode` (ou evolução de `compress`): construtor de argumentos
  por encoder, substituindo o if-else fixo:
  - software: `libx264` / `libx265` (2-pass igual ao atual — `-pass 1/2`
    funciona para ambos);
  - `h264_nvenc` / `hevc_nvenc`: qualidade via `-cq`; tamanho via `-rc vbr`
    `-b:v` `-maxrate` `-bufsize` (1-pass VBR aproxima o alvo — **sem** precisão
    do 2-pass; documentar o trade-off e manter default software);
  - `qsv`: 2-pass e `-global_quality`; `amf`: `-quality`/`-rc vbr`;
    `videotoolbox`: `-q:v` (qualidade) / `-b:v` (tamanho), sem 2-pass;
    `vaapi` (Linux): exige cadeia `-vaapi_device` + `format=vaapi,hwupload`,
    fora do escopo inicial — listar como indisponível até estender.
  - Tabela `encoder → args` coberta por testes.
- **Trade-off de qualidade:** NVENC entrega qualidade equivalente ao x264 só
  com ~20–30% mais bitrate (AMF/QSV ainda atrás do NVENC); no modo `size` o
  arquivo sai pior no mesmo MB. A UI avisa quando o encoder é de GPU — nunca
  muda o default silencioso. O conselho de bitrate do `quality.go` assume
  x264 e passa a valer só para o modo software.
- **Codec ≠ velocidade:** o maior ganho é hardware vs. software; H.265 por si
  só é *mais lento* que H.264 com melhor qualidade. Não empacotar os dois
  como se fosse a mesma escolha.

### Detecção de disponibilidade

- Rodar `ffmpeg -hide_banner -encoders` uma vez e extrair nomes; novo método
  `GetEncoders() ([]string, error)` (cache em memória, invalida em
  "verificar de novo"). Encoders inexistentes ficam desabilitados na UI.
- A listagem só diz que o **build** do ffmpeg tem o encoder; o driver da GPU
  pode ainda faltar (ex.: NVENC compilado sem driver NVIDIA). Probe real: um
  encode de 1 frame em arquivo temporário; falha → encoder sai da lista com o
  motivo. Testável com fake `ffmpeg`.
- Fase 5 (ajustes) será plugada no mesmo builder (`-vf`, `-ss/-to`, `-r`).

### Frontend

- Passo 02: expansor "avançado" com codec (H.264/H.265) e hardware
  (auto/nenhuma/com GPU) vindo de `GetEncoders()`; agrupado por preset mas
  alterável por job.
- Aviso visível quando hardware ativo: "GPU: mais rápido, qualidade um pouco
  menor" (e, no modo tamanho, "+/− ~5–10% no tamanho final"). Preferência
  persistida na config (fase 1) se o usuário quiser manter GPU sempre —
  **adiado na v1**: a escolha de codec/hardware é por preset+job e não é
  salva entre sessões.

### Interação com a fila (fase 2)

- GPUs de consumo limitam sessões de encode simultâneas; a fila não deve
  disparar mais jobs de GPU em paralelo que a fila permite de CPU. Quando
  `Hardware != ""`, o worker usa slot próprio (cap `min(maxParallel, 1..2)`)
  — detalhar o número ao implementar, medindo na máquina real. **Adiado para
  v2:** na v1 o `MaxParallel` default (=1) já serializa os jobs, então GPU e
  CPU não concorrem entre si; o cap dedicado chega junto da configuração de
  paralelismo.

### Testes

- `internal/encode`: tabela de argumentos por encoder (software/nvenc/qsv/
  amf/videotoolbox); flags `-pass` só no modo tamanho; `-crf`/`-cq` só no modo
  qualidade; `-an`/áudio conforme `HasAudio`.
- Detecção com fake `ffmpeg` (script sh, `skipOnWindows`), inclusive o caso
  "encoder listado mas probe falha".
- Integração: `Run` com fake que valida o binário invocado.

### Docs

- README: seção "Codecs e aceleração de GPU" — tabela encoders × recursos
  (2-pass, controle de tamanho), por que software continua sendo o default,
  como ver se a GPU suporta NVENC/AMF/QSV/VideoToolbox no próprio app.
- CHANGELOG: entrada na feature.

---

## Fase 5 — Ajustes por arquivo (escala, corte, remover áudio, FPS, thumbnail)

**Objetivo:** controles de ajuste por vídeo além do preset fixo.

> Diferença entre **corte** desta fase e **split** da fase 11: aqui `TrimStartSec/
> TrimEndSec` guarda **uma única janela** de um vídeo (ex. só o trecho 1:00–3:30);
> na fase 11 o vídeo é dividido em **N partes sequenciais** (ex. 30 min → 6 × 5 min)
> reaproveitando o mesmo `-ss/-to` e a nomenclatura de segmentos.

### Backend
- `compress.Job`: campos opcionais `Scale string`, `TrimStartSec/TrimEndSec
  float64`, `RemoveAudio bool`, `FPS float64`, `Rotate int`,
  `ThumbnailPath string`.
- Método `extract`/builder de filtros que compõe a string `-vf` (ordem
  fixa: transpose → scale → fps):
  - Escala: em branco mantém o default atual
    (`scale='min(1280,iw)':...`, compress.go:150-151); valor explícito vira
    `scale=WxH:force_original_aspect_ratio=decrease`.
  - Corte: `-ss start -to end` posicionados **antes** do `-i` (seek rápido);
    ativos nas duas passadas e na extração de áudio.
  - Remover áudio: skip dos args `-c:a` no pass2/CRF; pass1 já é `-an`.
  - FPS: filtro `fps=<FPS>`.
  - Rotação: `transpose=1/2/3` (90/180/270) antes do scale.
  - Thumbnail: passo pós-encode separado
    `ffmpeg -ss <pos> -i <out> -frames:v 1 <thumb>` (não entra nas passadas).
- `app.go`: `Job` já é passado inteiro no `Compress`/fila; validar conflitos
  (corte invertido, FPS ≤ 0 etc.).

### Frontend
- Passo 02: expansor "ajustes" com escala (original/padrão 1280/custom),
  `mm:ss` de início/fim, toggle remover áudio, FPS, orientação e "gerar
  thumbnail" (com caminho destino).

### Testes
- Builder: composição de `-vf`, presença/ausência de `-an`/`-c:a`, ordem dos
  filtros, posicionamento de `-ss/-to`.
- Validações de `Job` (erros esperados).
- Integração: `Run` falso capturando args para cenário completo.

### Docs
- README: seção "Ajustes por arquivo"; docs dos formatos `mm:ss`.

---

## Fase 6 — Extração de áudio com opções (mp3/opus/aac/flac/wav/copiar)

**Objetivo:** tirar a trilha de áudio do vídeo sem reencodar o vídeo, com
escolhas explícitas de formato, qualidade, canais e normalização. Hoje o app
só comprime vídeo; extração é um botão novo no card do arquivo.

> Depende da fase 2 (`Task.Kind="audio"` na fila) e da fase 1 (preferências:
> caminho de saída, sufixo). O trim reaproveita `-ss/-to` da fase 5.

### Modelo

- Novo pacote `internal/audio`, espelhando o builder de encoders da fase 4
  (mesmo padrão `Spec` + tabela `formato → args` coberta por testes):
  ```go
  type Spec struct {
      Codec       string  `json:"codec"`         // "mp3"|"opus"|"aac"|"flac"|"wav"|"copy"
      BitrateKbps int     `json:"bitrateKbps"`   // 0 = default do formato
      SampleRate  int     `json:"sampleRate"`    // 0 = manter do fonte
      Channels    int     `json:"channels"`      // 0 = manter, 1 = mono, 2 = estéreo
      TrimStartSec float64 `json:"trimStartSec"` // janela reaproveitada da fase 5
      TrimEndSec   float64 `json:"trimEndSec"`
      Normalize   bool    `json:"normalize"`     // loudnorm EBU R128
      OutputPath  string  `json:"outputPath"`
  }
  ```

### Formatos e argumentos

| Formato | Encoder | Flag de taxa | Default | Observação |
|---|---|---|---|---|
| `mp3` | `libmp3lame` | `-b:a <k>` ou `-q:a 2` | 192 kbps | padrão da UI |
| `opus` | `libopus` | `-b:a <k>` | 128 kbps | melhor voz |
| `aac` | `aac` | `-b:a <k>` | 160 kbps | sem encoder extra |
| `flac` | `flac` | — | — | lossless, sem `-b:a` |
| `wav` | `pcm_s16le` | — | — | master/edição |
| `copy` | `copy` | — | — | só remux, sem recodificar |

- Comando base: `ffmpeg [-ss start -to end] -i in -vn -map 0:a:0 <codec args>
  out` via `cmdutil.Command` (mesmo wrapper de `internal/compress`), com o
  contexto cancelável do job.
- `-ss/-to` **antes** do `-i` (seek rápido), igual ao trim da fase 5.
- `-ac` só quando `Channels > 0`; `-ar` só quando `SampleRate > 0`.
- `Normalize`: `-af loudnorm=I=-16:LRA=11:TP=-1.5` (single-pass, aproximado).
  O 2-pass medido fica como *stretch goal* — a UI não deve depender dele.

### Validações (erro claro, nunca ajuste silencioso)

- `Codec` desconhecido → erro listando os válidos.
- `BitrateKbps > 0` com `flac`/`wav`/`copy` → erro "bitrate não se aplica a
  X" (mesma política da fase 11 com parte < 1 min).
- `BitrateKbps` fora de `[32, 320]` para lossy → erro.
- `Channels` fora de `{0,1,2}` → erro.
- `TrimEndSec <= TrimStartSec` quando ambos vêm preenchidos → erro.
- `info.HasAudio == false` → erro "o vídeo não tem faixa de áudio" (usando
  `media.Info`, que já expõe `HasAudio`).

### Backend

- `internal/audio`: `Extract(ctx, inputPath, Spec) (sizeBytes int64, err error)`
  e `SuggestOutput(path, codec, suffix)` montando `<nome>-<sufixo>.<ext>` com a
  extensão coerente com o formato.
- `internal/events`: `DoneEvent` e `ProgressEvent` ganham
  `Kind string \`json:"kind"\`` (`"compress"|"audio"`) — **definir junto com a
  fase 2**, senão o frontend não distingue quem concluiu.
- `app.go`: `ExtractAudio(req audio.Spec) (string, error)` enfileira
  (`Task.Kind="audio"`) e devolve o `jobID`; `OpenOutputDialog` passa a filtrar
  pela extensão do formato escolhido (hoje filtra só `.mp4`, app.go:138-153).
- Preferências (fase 1): sufixo de saída do áudio, default `-audio` (o
  `-compressed` existente é para vídeo; `PreferencesModal.svelte` já tem o campo
  de sufixo, ganha um por tipo).

### Frontend

- `VideoRow.svelte`: item novo no `KebabMenu` ("extrair áudio…") ao lado de
  "trocar arquivo"/"limpar" — ou card de ação no `StatusPage` quando não há
  vídeo ainda.
- Modal novo `AudioModal.svelte` reaproveitando o design system (`Modal`,
  `RadioCardGroup`, `Stepper`, `Toggle` já existentes em `frontend/src/lib/`):
  - formato: `RadioCardGroup` (MP3/Opus/AAC/FLAC/WAV/copiar);
  - qualidade: `Stepper`/select de kbps, **oculto** para `flac`/`wav`/`copy`;
  - canais: `RadioCardGroup` manter/mono/estéreo;
  - normalizar: `Toggle` com `InfoTip` explicando o EBU R128;
  - janela: dois `Stepper` `mm:ss` reaproveitando o do split (fase 11);
  - prévia do nome de saída (`SuggestOutput`) e botão "EXTRAIR ÁUDIO".
- Ao concluir, o item da fila mostra formato + tamanho + `OpenFolder` (mesmo
  card de resultado da compressão).

### Testes

- `internal/audio`: tabela `Spec → argv` por formato (6 casos), incluindo
  `-b:a` ausente em lossless, `-ac`/`-ar` condicionais, ordem de `-ss/-to`
  antes do `-i`; tabela de erros de validação.
- Integração: `Extract` com fake `ffmpeg` (script sh, `skipOnWindows` como nos
  testes existentes de `internal/compress`) capturando `argv`; `SuggestOutput`
  por extensão.
- `app_test.go`: `ExtractAudio` com diretório temporário e video sem áudio.
- Eventos: `Kind` preenchido em `done`/`progress`.

### Docs

- README: seção "Extrair áudio" com a tabela de formatos × opções, o que cada
  preset de taxa faz e o aviso de que `flac`/`wav` não aceitam bitrate.
- CHANGELOG: entrada na feature.

---

## Fase 7 — Notificação ao concluir + abrir pasta

**Objetivo:** aviso de sistema quando um job termina/erro e auto-abrir pasta.

### Backend
- `internal/notify` com interface `Sender` (padrão `beeep`/`go-toast`
  Windows/`notify-send` Linux/`osascript` macOS). A dependência `go-toast`
  (v2.0.3) já existe no `go.mod`; **decisão**: usar `beeep` (portável, um
  depêndencia) ou wrappers por SO. Ver §Decisões abertas.
- Chamada após emitir `compress:done`/`compress:error` (na fila, fase 2), só se
  `config.NotifyOnDone`. Sem evento novo — notificação é efeito colateral.
- Toggle `config.OpenFolderOnDone`: acionar `OpenFolder(OutputPath)` ao
  concluir (default false).

### Frontend
- Checkboxes no painel de configuração (fase 1); nada novo no fluxo principal.

### Testes
- `notify.GetSender`/Send com sender injetado (nil-safe quando sem notifier);
- worker emite notificação nos eventos done/error (fake sender, contador).

### Docs
- README: seção "Notificações".

---

## Fase 8 — Verificação de atualização no app

**Objetivo:** avisar se há release mais nova (sem exigir internet nem quebrar o
modo offline atual).

### Backend
- `internal/update`: consulta `GET
  https://api.github.com/repos/fabianoflorentino/vidctl/releases/latest` com
  timeout curto (~5s) e cache único por sessão. Offline/timeout/HTTP erro →
  retorno `Ok=false` silencioso (não é falha bloqueante).
- Comparar tag com a versão embutida no binário: `version.Version` (var de
  pacote preenchida via `-ldflags -X` no Makefile/CI — hoje `VERSION` já é
  calculado no Makefile:10).
- `app.go`: `CheckUpdate() (update.Info, error)` com
  `{HasUpdate, LatestTag, URL, AssetURLPerPlatform}` (nome do asset por
  plataforma, igual ao release.yml).
- **Rate-limit**: API não-autenticada (60 req/h); cache + botão manual evitam
  estourar.

### Frontend
- No masthead (junto ao gear): link discreto "nova versão disponível" (badge) e
  menu "verificar atualização…" executando `CheckUpdate` em background; nunca
  bloqueia o boot.

### Testes
- `internal/update` contra `httptest.Server` (200 com payload, 404, timeout de
  rede); parse da tag e filtro por plataforma; cache de sessão; offline →
  `Ok=false` sem erro.

### Docs
- README: seção "Verificação de atualização" (fonte oficial, política offline).

---

## Fase 9 — Presets personalizados pelo usuário

**Objetivo:** criar/editar/excluir presets além dos 6 fixos em
`internal/presets/presets.go:16-71`.

### Backend
- `internal/presets`: persistência de custom presets (`os.UserConfigDir()/
  vidctl/presets.json`, mesma escrita atômica da fase 1); `List()` = builtins +
  custom (custom por último); IDs com prefixo `custom-` (unicidade garantida).
- Validação: `Mode=size → SizeMB>0`; `Mode=crf → 0<CRF<=51`; `AudioBitrate`
  parseável por `bitrateToBits` (compress.go:81) — exportar ou replicar.
- `app.go`: `AddCustomPreset`, `UpdateCustomPreset`, `DeleteCustomPreset`
  (retornam o `[]presets.Preset` atualizado para o frontend repintar).
- Campos novos da fase 4 (`Codec`, `Hardware`) entram no preset custom.

### Frontend
- Grade de presets ganha card "+ criar" → modal (nome, modo, tamanho/CRF,
  bitrate de áudio, codec/hardware); menu do card para editar/excluir (só
  custom). Persistência transparente.

### Testes
- Store: roundtrip, validações (tabela de erros), unicidade de IDs, merge
  builtins+custom e prioridade.

### Docs
- README/docs de presets: como criar um preset custom.

---

## Fase 10 — i18n pt/en (opcional)

**Objetivo:** interface em pt (default) e en.

- `frontend/src/i18n.ts`: dicionário + `t(key)`; strings do `App.svelte`
  movidas para as chaves.
- `config.Language` controla o idioma (selector no painel de configuração),
  default `"pt"`.
- Backend sem mudança de eventos; `SystemStatus.Message`/erros continuam em pt
  e podem entrar num dicionário do backend depois (fora de escopo agora).
- Testes: `i18n.ts` (chaves completas por idioma) via `svelte-check`/teste de
  TS simples.

---

## Fase 11 — Corte em segmentos por tempo (split)

**Objetivo:** cortar o vídeo em partes, seja por número de partes ou por
minutos por parte, reaproveitando o mesmo encode.

> **Status: entregue** em `internal/split` + `App.svelte` (v2.1.0, junto do
> redesign de UI). Registrada aqui para não sumir do plano.

### Backend
- `internal/split/split.go`: `Spec` (`Parts` **ou** `MinutesEach`), `Segment`
  (`Index`, `StartSec`, `EndSec`), `Plan(durationSec, spec)` e
  `Filename(path, index, total)`.
  - `Parts`: fatias iguais de `duration/Parts`.
  - `MinutesEach`: fatias de comprimento fixo; o resto vira parte própria, a não
    ser que fique abaixo de `MinPartSec` — nesse caso é unido à última fatia,
    para nenhuma parte sair com menos de 1 minuto.
  - Limites: `MinPartSec`, `MaxParts` e `MaxMinutesEach`.
  - `Filename` insere `-partN` antes da extensão e preenche com zero quando o
    total passa de 9 dígitos, para os arquivos ordenarem corretamente.
- `compress.Job.Split *split.Spec` (`omitempty`): ausente ou inválido, o job
  roda como encode único — é o que mantém a fase 2 (`Task`) compatível.
- `encodeSegment` faz `-ss`/`-to` por segmento e reporta progresso pela
  posição do segmento dentro do total.

### Frontend
- Toggle "dividir" com dois modos (`RadioCardGroup`): **N partes** (Stepper
  inteiro) e **minutos por parte** (Stepper fracionário, ex.: 2,5 min).
- Prévia do plano antes de comprimir, com a contagem de partes e o aviso de
  bloqueio quando o plano é inválido.

### Testes
- `Plan` por partes, por minutos, resto absorvido, limites (`MinPartSec`,
  `MaxParts`), duração zero/negativa, `Spec` vazio.
- `Filename` com 1, 9 e 10+ partes (zero-padding).
- `compress`: `-ss`/`-to` por segmento e output `-partN` (90,8% no pacote).

### Docs
- README: corte em N partes / X minutos.

---

## Fase 12 — Transcrição de vídeo (legendas srt/vtt + texto)

**Objetivo:** gerar transcrição automática do áudio (legendas e texto plano)
usando o próprio ffmpeg ou um backend de speech-to-text local/privado. O foco
inicial é viável com ferramentas presentes (`ffmpeg` + whisper.cpp opcional) e
sem telemetria obrigatória.

> Reutiliza `audio.Extract`/PCM (fase 6), trim `-ss/-to` (fase 5) e fila
`Task.Kind="transcribe"` (fase 2). Gera arquivos `.srt`, `.vtt` e `.txt` com
timestamp opcional.

### Modelo

- Novo pacote `internal/transcribe`:
  ```go
  type Spec struct {
      InputPath    string  `json:"inputPath"`
      OutputBase   string  `json:"outputBase"`   // sem extensão
      Engine       string  `json:"engine"`        // "whispercpp"|"vosk"|"auto"
      Language     string  `json:"language"`      // "pt"|"en"|"auto"
      ModelPath    string  `json:"modelPath"`     // caminho para modelo local
      Format       string  `json:"format"`        // "srt"|"vtt"|"txt"|"all"
      WordTimestamps bool  `json:"wordTimestamps"`
      TrimStartSec float64 `json:"trimStartSec"`
      TrimEndSec   float64 `json:"trimEndSec"`
  }
  type Result struct {
      Files []string `json:"files"`
      Text  string   `json:"text"`
  }
  ```

### Estratégias de engine (preferência por local)

| Engine | Status | Requer | Saída | Observação |
|---|---|---|---|---|
| `whispercpp` (ggml) | recomendado | binário `whisper-cpp`/`whisper-cli` no PATH | SRT/VTT/TXT | off-line, privado, roda bem CPU. Modelo `.bin` pequeno/médio. |
| `vosk` | recomendado 2 | binário/CLI `vosk-transcriber` ou API local | SRT/VTT/TXT | leve, modelos por idioma. |
| `ffmpeg+speech` | fallback | só ffmpeg (limitado) | TXT/SRT (parcial) | alguns builds trazem; não confiável p/ pt-BR. Default `auto` tenta na ordem acima e cai para o disponível. |

- **Fallback robusto:** se nenhum engine local estiver disponível, o backend
  devolve `ErrNoTranscriber` com lista de engines suportados + link para docs
  (sem sugestão de nuvem). **Nunca** sugerir API externa por padrão.

### Backend

- `internal/transcribe/detect.go`: `AvailableEngines() []string` verifica
  binários no PATH (`which whisper-cli whisper-cpp vosk-transcriber`) + modelos
  configurados. Cache em memória por sessão.
- `internal/transcribe/extract_audio.go`: extrai WAV/PCM 16kHz mono
  temporário via `ffmpeg -ss/-to -i in -vn -ac 1 -ar 16000 -c:a pcm_s16le
  tmp.wav` (reaproveita lógica da fase 6). Arquivo temporário removido com
  `defer os.Remove`.
- `internal/transcribe/whispercpp.go`, `vosk.go`: wrappers CLI com `cmdutil.Command`
  (contexto cancelável). Parse de saída SRT/VTT quando o CLI já gera, senão
  converte JSON → SRT/VTT (`internal/transcribe/convert.go`).
- `app.go`: `TranscribeVideo(spec transcribe.Spec) (transcribe.Result, error)`
  enfileira `Task.Kind="transcribe"` (fase 2); devolve lista de arquivos gerados
  + texto completo.
- Preferências (fase 1): `TranscribeEngine`, `TranscribeModelPath`,
  `TranscribeLanguage`, `TranscribeWordTimestamps`, `TranscribeFormat`.

### Eventos e progresso

- Eventos `compress:progress/done/error` recebem `Kind="transcribe"`. Progresso
  percentual: 0–20% extração PCM, 20–100% inferência (quando o CLI emite % ou
  estimado por duração). Se o CLI não reporta %, usar `Stage="transcribing"` +
  percentual baseado em tempo (heurística não-bloqueante).

### Frontend

- `VideoRow.svelte`/menu: "transcrever vídeo…" abre `TranscribeModal.svelte`
  (design system):
  - Engine: dropdown com disponíveis (`auto` + detectados). Desabilita inválidos
    com `InfoTip` "instale X e defina modelo".
  - Idioma: `pt`, `en`, `auto`.
  - Formato: `all` / `srt` / `vtt` / `txt`.
  - Word timestamps: toggle (só SRT/VTT).
  - Janela `mm:ss` (trim) reaproveitando fase 5/11.
  - Prévia de `OutputBase` (sufixo default `-transcript`).
- Resultado da fila: mostra arquivos gerados (links para abrir pasta) + botão
  "copiar texto" (txt completo). Legendas `.srt/.vtt` abrem no player externo
  (não é escopo editar no app).

### Testes

- `internal/transcribe/detect`: PATH vazio → engines vazios; PATH fake com
  binários → detectados.
- `internal/transcribe/convert`: JSON→SRT, JSON→VTT, blocos com word timestamps.
- `internal/transcribe`: Spec → args CLI por engine (mock), remoção de tmp.wav
  (`t.TempDir`), trim `-ss/-to` antes do `-i`.
- `app_test.go`: `TranscribeVideo` com job enfileirado, `Kind="transcribe"`.
- Eventos: `Kind` correto em done/progress/error.

### Docs

- README: seção "Transcrição (legendas SRT/VTT + texto)" — tabela de engines,
  requisitos (modelos locais), privacidade ("tudo roda localmente"), formatos
  gerados e como instalar `whisper-cli`/`vosk` (links).
- CHANGELOG: entrada na feature.
- `docs/instalacao.md`: opcional, instruções de modelos leves.

### Decisões

- **Privacidade por padrão:** só engines locais. Sem fallback para nuvem nesta
  fase. Se nenhum disponível, UI mostra instruções claras (não quebra o fluxo
  de compressão).
- **Formato base:** extrair PCM 16kHz mono (universal p/ STT). Gera todos os
  formatos solicitados em uma única execução quando `Format="all"`.
- **Modelos:** não baixar automático — caminho configurável (evita peso/decisão
  de licença). Sugestões de tamanhos no modal (`tiny/base/small`).

---

## Melhorias transversais necessárias

- **Fila + Job + eventos**: o contrato `Job` muda nas fases 4/5 — definir os
  campos novos **junto** da fase 2 para o `Task` da fila já nascer estável
  (evitar refator duplo).
- **`OpenMultipleDialog`** já pronto (app.go:179) e **filtro de extensão dinâmico**
  (fase 6) tocam `app.go:80-109` — agrupar o ajuste desses dois helpers na fase 2.
- **Versão embutida** (fase 8): plugar `-ldflags -X` no Makefile (Makefile:10 já
  calcula `VERSION`) e no release.yml; sem isso `CheckUpdate` não tem com o que
  comparar.
- **Cobertura**: manter o gate ≥ 80% — cada fase adiciona testes junto do
  código (nunca testes "depois"); a fase 2 é a de maior risco de queda por
  refator de eventos/managers.
- **Regressão de versão**: as fases 2 e 4/5 mudam o payload de `Job` — atualizar
  `frontend/wailsjs/go/models.ts` via `wails generate module` e as guardas de
  eventos no frontend.
- **`Kind` nos eventos** (fases 6 e 12): `ProgressEvent`/`DoneEvent`/
  `ErrorEvent` ganham `Kind` (`"compress"|"audio"|"transcribe"`) para o
  frontend rotear sem adivinhação. Definir com a fase 2.

---

## Decisões abertas

1. **Notificações (fase 7)**: adotar `beeep` (uma dependência, 3 SOs) vs.
   wrappers por SO usando `go-toast` (já no `go.mod`, Windows-only) +
   `notify-send`/`osascript` (zero dependência nova). Recomendado: **beeep**,
   com interface `Sender` para trocar depois sem tocar a UI.
2. **Precisão em hardware (fase 4)**: NVENC/AMF 1-pass VBR não atingem o
   tamanho exato do 2-pass. Default permanece software; hardware no modo
   `size` assume tolerância (+/− ~5–10%). Validar com usuário se aceita.
   > ✅ **Resolvido na v1 (feat/hevc):** opt-in com aviso visível na UI;
   > validado na entrega do PR.
3. **Paralelismo default (fase 2)**: manter `MaxParallel=1` na primeira
   entrega da fila (previsível, CPU-bound) e expor o campo em configuração
   antes de ativar >1 por default.
4. **Download de atualização (fase 8)**: só *notificar* (link para a release)
   na v1 — baixar/instalar dentro do app fica para depois (assinatura de
   binários não existe ainda).
5. **Extração de áudio na fila (fase 6)**: integrar como `Task.Kind=audio` na
   fila (recomendado, uniformiza o modelo da fase 2) ou manter botão isolado do
   passo 01 na primeira entrega.
 6. **Split com stream copy (fase 11)**: na primeira entrega, partes sempre são
    reencodadas (padrão consistente com preset). Adicionar depois um toggle
    "copiar sem reencodar" (`-c copy`) quando a resolução/codec não muda — vale
    validar com usuário se a velocidade do split importa mais que a qualidade.
 7. **Transcrição (fase 12)**: engines locais apenas (privacidade). Se nenhum
     disponível, UI mostra instruções claras e não sugere nuvem. Modelos não
     são baixados automaticamente (caminho configurável).
 8. **GPU como default (fase 4)**: ativar hardware automaticamente quando
     disponível (mais rápido, pior qualidade por bit) ou deixar opt-in por job
     com aviso explícito? Recomendado: **opt-in**, mantendo software como
     default — preserva o comportamento atual e a confiança no tamanho final
     do modo `size`; o custo (20–30% mais bitrate no mesmo resultado) só vale
     quando a velocidade importa mais. Validar com usuário.
     > ✅ **Resolvido na v1 (feat/hevc):** opt-in por job (valor `auto`
     > resolve para o primeiro backend detectado quando o usuário escolhe);
     > validado na entrega do PR.

---

## Ordem sugerida de execução

Uma fase por versão, na ordem de dependência. A coluna da branch segue o
[fluxo de execução](#fluxo-de-execução).

```
v2.1.0  → Fase 11 (cortado em partes) [entregue fora de ordem, junto do redesign de UI]
v2.2.0  → Fase 1 (config)            [base p/ tudo]
v2.3.0  → Fase 2 (fila)              [maior; define Task+Job de vez; usa OpenMultipleDialog já pronto]
v2.4.0  → Fase 3 (estimativa)
v2.5.0  → Fase 4 (codec/hw: HEVC + GPU)  [default segue software; GPU opt-in]
v2.6.0  → Fase 5 (ajustes)           [inclui trim -ss/-to reaproveitado da fase 11]
v2.7.0  → Fase 6 (extrair áudio + opções)
v2.8.0  → Fase 7 (notificações)      [depende de config]
v2.9.0  → Fase 8 (check update)      [usa a API de releases por tag + fluxo de publish novo]
v2.10.0 → Fase 9 (presets custom)    [depende de config]
v2.11.0 → Fase 10 (i18n, opcional)
v2.12.0 → Fase 12 (transcrição srt/vtt/txt)
```

Cada versão deve passar por `make check` (vet + test + gofmt + svelte-check) e
pelo gate de cobertura do CI antes do merge/push.

> A Fase 1 já está implementada em `feat/fase1` (ver o
> [fluxo de execução](#fluxo-de-execução)); ao abrir o PR, marcar a linha da
> branch correspondente como `aberto` e, no merge, renomear a versão prevista
> para o número que o release realmente levar.