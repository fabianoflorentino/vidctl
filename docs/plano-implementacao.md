# Plano de implementação — novas funcionalidades

Documento de plano para a próxima rodada de funcionalidades do vidctl.
Baseado no estado atual do código (Go 1.25 + Wails v2.15 + Svelte 5, um único
componente `App.svelte`).

> Regras do projeto (AGENTS.md) que valem para **todas** as fases:
> - toda mudança de código vem com testes cobrindo o comportamento;
> - rodar sempre `go test ./...` e manter cobertura ≥ 80% (gate no CI);
> - comportamento novo deve refletir no `README.md` e, quando aplicável, em `docs/`;
> - sem comentários em código a menos que realmente necessários.

---

## Síntese das fases (ordem recomendada)

| Fase | Funcionalidade | Doc | Esforço |
|---|---|---|---|
| 1 | Configuração persistente + preset "último usado" | §1 | M |
| 2 | Fila de conversões (batch) | §2 | G |
| 3 | Estimativa de tamanho antes do encode | §3 | M |
| 4 | Codec HEVC/x265 + aceleração de hardware | §4 | G |
| 5 | Ajustes por arquivo (escala, corte, áudio, FPS, thumbnail) | §5 | G |
| 6 | Extração de áudio (mp3/opus) | §6 | P |
| 7 | Notificação ao concluir + abrir pasta | §7 | P |
| 8 | Verificação de atualização no app | §8 | M |
| 9 | Presets personalizados pelo usuário | §9 | M |
| 10 | i18n (pt/en) — opcional | §10 | M |
| 11 | Corte em segmentos por tempo (split) | §11 | G |

Redesign da UI no estilo do Constrict (fila visual, drag&drop, tema adwaita)
está detalhado em [`plano-ui-constrict.md`](./plano-ui-constrict.md); ele consome
as Fases 1, 2, 3, 4 e 10 deste plano.

Ordem por dependência: config (1) libera fila (2, `MaxParallel`), notificações
(7), presets custom (9). Estimativa (3) e codec/hw (4) são independentes entre
si, mas o job enriquecido da fase 4/5 deve ser desenhado junto com a fila (2)
para o contrato `Job` não quebrar duas vezes.

---

## Fase 1 — Configuração persistente + preset "último usado"

**Objetivo:** introduzir a primeira camada de persistência do app e restaurar o
preset/tamanho/saída no próximo boot. Hoje não existe nenhuma persistência:
`sizeMB`/`crf` e caminhos vivem só em memória (`App.svelte`).

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
  - "Adicionar vídeo(s)" multipick: Wails v2 expõe
    `OpenMultipleFilesDialog` (hoje `OpenInputDialog` é single, app.go:80-92).
    Substituir por `OpenMultipleFilesDialog` com os mesmos filtros e inserir
    cada arquivo na fila.
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

## Fase 4 — Codec HEVC/x265 + aceleração de hardware

**Objetivo:** além do libx264 atual (fixo em `buildSizePasses`/`buildCrfPass`,
compress.go:145-213), permitir x265 e encoders de GPU (NVENC/QuickSync/AMF)
com detecção de disponibilidade.

### Backend
- `presets.Preset`: novos campos `Codec string` (`"h264"|"h265"`) e
  `Hardware string` (`""|"nvenc"|"qsv"|"amf"`).
- Novo `internal/encode` (ou evolução de `compress`): construtor de argumentos
  por encoder, substituindo o if-else fixo:
  - software: `libx264` / `libx265` (2-pass igual ao atual — `-pass 1/2`
    funciona para ambos);
  - `h264_nvenc` / `hevc_nvenc`: qualidade via `-cq`; tamanho via `-rc vbr`
    `-b:v` `-maxrate` `-bufsize` (1-pass VBR aproxima o alvo — **sem** precisão
    do 2-pass; documentar o trade-off e manter default software);
  - `qsv`: 2-pass e `-global_quality`; `amf`: `-quality`/`-rc vbr`.
  - Tabela `encoder → args` cobertos por testes.
- Detecção de disponibilidade: rodar `ffmpeg -hide_banner -encoders` uma vez e
  extrair nomes; novo método `GetEncoders() ([]string, error)` (cache em
  memória, invalida em "verificar de novo"). Encoders inexistentes ficam
  desabilitados na UI.
- Fase 5 (ajustes) será plugada no mesmo builder (`-vf`, `-ss/-to`, `-r`).

### Frontend
- Passo 02: expansor "avançado" com codec (H.264/H.265) e hardware
  (auto/nenhuma/com GPU) vindo de `GetEncoders()`; agrupado por preset mas
  alterável por job.

### Testes
- `internal/encode`: tabela de argumentos por encoder; flags `-pass` só no modo
  tamanho; `-crf`/`-cq` só no modo qualidade; `-an`/áudio conforme
  `HasAudio`.
- Detecção com fake `ffmpeg` (script sh, `skipOnWindows`).
- Integração: `Run` com fake que valida o binário invocado.

### Docs
- README: seção "Codecs e aceleração de hardware" (tabela encoders × recursos,
  limitações de precisão de tamanho em hardware).

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

## Fase 6 — Extração de áudio (mp3/opus)

**Objetivo:** extrair trilha de áudio sem reencodar o vídeo.

### Backend
- `internal/audio` (ou função no `compress`): `Extract(ctx, Job)` com
  `{InputPath, OutputPath, Codec ("mp3"|"opus"|"aac")}`; comando
  `ffmpeg -i in -vn -c:a <libmp3lame|libopus|aac> -b:a 192k out` via
  `cmdutil.Command`.
- Eventos: reutilizar `compress:done` com `Stage:"audio"` (payload já carrega
  `OutputPath`); sem barra de progresso (varredura quase instantânea).
- `OpenOutputDialog` já filtra `.mp4` (app.go:96-109) — generalizar filtro por
  extensão (`.mp3`, `.opus`, `.m4a`).

### Frontend
- Card do arquivo no passo 01: botão "extrair áudio…" com escolha de codec;
  resultado mostra como um item "concluído" com `OpenFolder`.

### Testes
- Builder do comando por codec; integração fake; defaults de `-b:a`.

### Docs
- README: seção "Extrair áudio".

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

## Fase 11 — Corte em segmentos por tempo (split em N partes)

**Objetivo:** dividir um vídeo em N partes de duração fixa ou em N partes
iguais. Ex. que o usuário descreveu: um vídeo de 30 minutos com a opção de
cortar em 6 partes de 5 min cada — com mínimo de **1 min por parte**.

### Regras de negócio

- O usuário informa o **número de partes** (N) **ou a duração por parte**
  (minutos); o campo que não foi informado é calculado a partir da duração do
  vídeo (`media.Info.DurationSec`).
- **Mínimo de 1 min por parte**: qualquer parte calculada < 60 s é rejeitada na
  validação (não arredondar silenciosamente).
- A **última parte pode ser menor** que a duração target quando a duração total
  não é divisível (30 min / 6 = 5 min exatos; 32 min / 6 → cinco de 5 min + uma
  de 2 min), desde que ≥ 1 min.
- Limite de N (sugestão: ≤ 60 partes) para evitar acidente com vídeos muito
  longos; validar no backend e no frontend.

### Backend

- `compress.Job` ganha `Split *SplitSpec` (`json:"split,omitempty"`):
  ```go
  type SplitSpec struct {
      Parts       int `json:"parts"`       // 0 = derivar da duração
      MinutesEach int `json:"minutesEach"` // 0 = derivar das partes
  }
  ```
  Só um dos campos deve ser preenchido (frontend envia um; o outro fica 0).
- Novo pacote `internal/split` (testável de forma pura):
  - `Plan(durationSec float64, spec SplitSpec) ([]Segment, error)` —
    calcula `[]Segment{StartSec, EndSec}` aplicando as regras acima; devolve
    erro claro para "nenhum campo informado", "ambos informados",
    "parte < 1 min", "N fora do limite".
  - `Filename(path string, index, total int) string` —
    `meu_video-part3.mp4` (ex. parte 3 de 6).
- `compress.go`: quando `Job.Split != nil`, `Run` (compress.go:246) executa o
  pipeline uma vez por segmento (job físico por parte, mesmo `JobID` pai):
  - comando por parte = comandos atuais (fases 4/5 buildam args via builder)
    com `-ss <Start>` `-to <End>` posicionados **antes** do `-i` (seek rápido,
    igual ao trim da fase 5) e `OutputPath` substituído por `Filename(...)`.
  - eventos `compress:progress` continuam por `JobID`; “parte X/N” entra no
    `stage` (ex. `encoding 3/6`) para o pie/row mostrar progresso agregado.
- Alternativa rápida (stream copy, sem reencodar): quando `Split` está ativo
  e o preset é `title`-only/`crf` derivado sem redimensionamento, permitir
  `-c copy` por parte. Default da fase: **reencodar** (consistente com o resto);
  `stream copy` entra como opção depois (decisão aberta #6).

### Frontend

- `App.svelte`, no grupo "Destino" (sidebar): subseção **"Cortar em partes"**
  com stepper `N partes` **ou** `minutos por parte` (toggle entre os dois),
  preview calculado "30:00 → 6 × 5:00" usando `info.durationSec`, e aviso em
  vermelho se alguma parte der < 1 min.
- Botão de comprimir ganha o rótulo dinâmico: `COMPRIMIR → 6 PARTES`.
- A fila (fase 2) lista um item por parte concluída (mesma fonte de `Filename`).

### Testes

- `internal/split`: tabela com 30 min/6 → 6×5 min; 32 min/6 → 5×5 + 2 min;
  “só partes” e “só duração”; dur total < 1 min → erro; parte < 1 min → erro;
  N > limite → erro; `Filename("a.mp4",3,6) == "a-part3.mp4"`.
- `compress`: `Run` com `Split` gera N outputs (fakes sh, `skipOnWindows`) e
  N eventos `compress:done`; `stage` carrega “x/N”.
- Validação de `SplitSpec` (campos conflitantes).

### Docs

- README: seção "Cortar em partes" com exemplos (30 min → 6 × 5 min) e regra
  do mínimo de 1 min.
- CHANGELOG: entrada na feature.

---

## Melhorias transversais necessárias

- **Fila + Job + eventos**: o contrato `Job` muda nas fases 4/5 — definir os
  campos novos **junto** da fase 2 para o `Task` da fila já nascer estável
  (evitar refator duplo).
- **`OpenMultipleFilesDialog`** (fase 2) e **filtro de extensão dinâmico**
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

---

## Decisões abertas

1. **Notificações (fase 7)**: adotar `beeep` (uma dependência, 3 SOs) vs.
   wrappers por SO usando `go-toast` (já no `go.mod`, Windows-only) +
   `notify-send`/`osascript` (zero dependência nova). Recomendado: **beeep**,
   com interface `Sender` para trocar depois sem tocar a UI.
2. **Precisão em hardware (fase 4)**: NVENC/AMF 1-pass VBR não atingem o
   tamanho exato do 2-pass. Default permanece software; hardware no modo
   `size` assume tolerância (+/− ~5–10%). Validar com usuário se aceita.
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

---

## Ordem sugerida de execução

```
v2.0.0  → Fase 11 (cortado em partes) [entregue fora de ordem, junto do redesign de UI]
v2.1.0  → Fase 1 (config)            [base p/ tudo]
v2.2.0  → Fase 2 (fila)              [maior; define Task+Job de vez; usa OpenMultipleDialog já pronto]
v2.3.0  → Fase 3 (estimativa)
v2.4.0  → Fase 4 (codec/hw)
v2.5.0  → Fase 5 (ajustes)           [inclui trim -ss/-to reaproveitado da fase 11]
v2.6.0  → Fase 6 (extrair áudio)
v2.7.0  → Fase 7 (notificações)      [depende de config]
v2.8.0  → Fase 8 (check update)      [usa a API de releases por tag + fluxo de publish novo]
v2.9.0  → Fase 9 (presets custom)    [depende de config]
v2.10.0 → Fase 10 (i18n, opcional)
```

Cada versão deve passar por `make check` (vet + test + gofmt + svelte-check) e
pelo gate de cobertura do CI antes do merge/push.