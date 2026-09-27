<script lang="ts">
  import { onMount } from 'svelte'
  import {
    CheckFFmpeg,
    GetPresets,
    OpenInputDialog,
    GetMediaInfo,
    GetThumbnail,
    OpenOutputDialog,
    Compress as StartCompress,
    Cancel as CancelJob,
    OpenFolder,
    GetUsage,
    GetAdvice,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js'
  import type { main, presets, media } from '../wailsjs/go/models.js'
  import { compress as bindings } from '../wailsjs/go/models.js'
  import StatusPage from './lib/StatusPage.svelte'
  import VideoRow from './lib/VideoRow.svelte'
  import Stepper from './lib/Stepper.svelte'
  import RadioCardGroup from './lib/RadioCardGroup.svelte'
  import InfoTip from './lib/InfoTip.svelte'
  import PreferencesModal from './lib/PreferencesModal.svelte'
  import { stageLabel } from './lib/stages'

  type ProgressEv = { jobId: string; stage: string; percent: number }
  type DoneEv = { jobId: string; outputPath: string; sizeMB: number; sizeBytes: number }
  type ErrorEv = { jobId: string; error: string }
  type ThemeChoice = 'system' | 'light' | 'dark'

  let presetList = $state<presets.Preset[]>([])
  let ffmpegOk = $state(true)
  let ffmpegMsg = $state('')

  let theme = $state<ThemeChoice>('system')

  try {
    const saved = localStorage.getItem('vidctl-theme')
    if (saved === 'light' || saved === 'dark') {
      theme = saved
      document.documentElement.dataset.theme = saved
    }
  } catch {
    /* localStorage indisponível: segue o tema do sistema */
  }

  function setTheme(next: ThemeChoice) {
    theme = next
    if (next === 'system') delete document.documentElement.dataset.theme
    else document.documentElement.dataset.theme = next
    try {
      localStorage.setItem('vidctl-theme', next)
    } catch {
      /* sem persistência disponível; tema vale para a sessão */
    }
  }

  let inputPath = $state('')
  let info = $state<media.Info | null>(null)
  let thumb = $state('')
  let outputPath = $state('')
  let selectedPresetId = $state('whatsapp-status')
  let sizeMB = $state(10)
  let crf = $state(23)

  let running = $state(false)
  let stage = $state('')
  let percent = $state(0)
  let error = $state('')
  let currentJobId = $state('')
  let done = $state<DoneEv | null>(null)
  let partsDone = $state<DoneEv[]>([])
  let jobTotalParts = $state(1)
  let usage = $state<Awaited<ReturnType<typeof GetUsage>> | null>(null)
  let advice = $state<Awaited<ReturnType<typeof GetAdvice>> | null>(null)
  let adviceSeq = 0

  let prefsOpen = $state(false)
  let dragging = $state(false)

  const VIDEO_EXT = ['.mp4', '.mov', '.mkv', '.webm', '.m4v', '.avi', '.wmv', '.flv']

  function isVideoPath(p: string): boolean {
    const low = p.toLowerCase()
    return VIDEO_EXT.some((ext) => low.endsWith(ext))
  }

  async function adoptFile(path: string) {
    if (running) {
      error = 'aguarde o job atual terminar antes de trocar o vídeo'
      return
    }
    if (!isVideoPath(path)) {
      error = `extensão não suportada: ${path.split('.').pop() || path}`
      return
    }
    inputPath = path
    outputPath = ''
    done = null
    partsDone = []
    jobTotalParts = 1
    error = ''
    thumb = ''
    try {
      info = await GetMediaInfo(path)
      const base = path.replace(/\.[^.]+$/, '')
      outputPath = base + '-compressed.mp4'
      GetThumbnail(path, info.durationSec)
        .then((t) => {
          if (inputPath === path) thumb = t
        })
        .catch(() => {
          thumb = ''
        })
    } catch (err) {
      info = null
      error = String(err)
    }
  }

  function handleDrop(paths: string[]) {
    const first = paths.find(isVideoPath) ?? paths[0]
    if (first) adoptFile(first)
  }

  function onDragOver(e: DragEvent) {
    if (e.dataTransfer?.types.includes('Files')) dragging = true
  }

  function onDragLeave(e: DragEvent) {
    if (e.relatedTarget === null) dragging = false
  }

  function onDropEnd() {
    dragging = false
  }

  $effect(() => {
    const path = inputPath
    const presetId = selectedPresetId
    const targetMB = sizeMB
    const qualityCRF = crf
    const split = splitPayload()
    const busy = running
    if (!path || !info || busy) {
      advice = null
      return
    }
    const seq = ++adviceSeq
    const timer = setTimeout(async () => {
      try {
        const ad = await GetAdvice(new bindings.Job({
          inputPath: path,
          outputPath: 'advice://placeholder',
          presetId,
          sizeMB: targetMB > 0 ? targetMB : 10,
          crf: qualityCRF,
          split: split ?? undefined,
        }))
        if (seq === adviceSeq) advice = ad
      } catch {
        if (seq === adviceSeq) advice = null
      }
    }, 250)
    return () => clearTimeout(timer)
  })

  function applyAdvice() {
    if (!advice) return
    const sug = advice.suggestion
    const mb = sug.sizeMB ?? 0
    const mins = sug.minutesEach ?? 0
    if (mb > 0) sizeMB = Math.min(100, Math.max(2, Math.round(mb)))
    if (mins > 0) {
      splitOn = true
      splitAxis = 'minutes'
      splitMinutes = mins
    }
  }

  $effect(() => {
    if (!running) {
      usage = null
      return
    }
    let stopped = false
    const poll = async () => {
      try {
        const s = await GetUsage()
        if (!stopped) usage = s
      } catch {
        /* collector indisponível: mantém último valor */
      }
    }
    poll()
    const id = setInterval(poll, 1000)
    return () => {
      stopped = true
      clearInterval(id)
    }
  })

  type SplitAxis = 'parts' | 'minutes'
  let splitOn = $state(false)
  let splitAxis = $state<SplitAxis>('parts')
  let splitParts = $state(6)
  let splitMinutes = $state(5)

  const MIN_PART_SEC = 60
  const MAX_PARTS = 60

  type SplitPlan = { count: number; sliceSec: number; error: string }

  function planSplit(durationSec: number): SplitPlan | null {
    if (!splitOn || !durationSec || durationSec <= 0) return null
    if (durationSec < MIN_PART_SEC) {
      return { count: 0, sliceSec: 0, error: 'vídeo tem menos de 1 min — impossível cortar' }
    }
    if (splitAxis === 'parts') {
      const parts = Math.floor(splitParts)
      if (parts < 2) return { count: 0, sliceSec: 0, error: 'mínimo de 2 partes' }
      if (parts > MAX_PARTS) return { count: 0, sliceSec: 0, error: `máximo de ${MAX_PARTS} partes` }
      const slice = durationSec / parts
      if (slice < MIN_PART_SEC) {
        return { count: 0, sliceSec: 0, error: 'cada parte teria menos de 1 min — use menos partes' }
      }
      return { count: parts, sliceSec: slice, error: '' }
    }
    const slice = Math.max(1, splitMinutes) * 60
    const full = Math.floor(durationSec / slice)
    const tail = durationSec - full * slice
    let count = full
    if (tail >= MIN_PART_SEC) count = full + 1
    if (count < 2) return { count: 0, sliceSec: slice, error: 'o vídeo cabe em 1 parte; não há corte' }
    if (count > MAX_PARTS) return { count: 0, sliceSec: slice, error: `corte geraria ${count} partes (máximo ${MAX_PARTS})` }
    return { count, sliceSec: slice, error: '' }
  }

  const activeSplit = $derived(planSplit(info?.durationSec ?? 0))
  const splitBlocked = $derived(!!activeSplit?.error)

  function useAxis(axis: SplitAxis) {
    splitOn = true
    splitAxis = axis
  }

  function setParts(v: number) {
    useAxis('parts')
    splitParts = v
  }

  function setMinutes(v: number) {
    useAxis('minutes')
    splitMinutes = v
  }

  function fmtClock(sec: number): string {
    if (!sec || sec < 0) return '—'
    const m = Math.floor(sec / 60)
    const s = Math.floor(sec % 60)
    return `${m}:${String(s).padStart(2, '0')}`
  }

  const selectedPreset = $derived(presetList.find((p) => p.id === selectedPresetId) ?? null)

  const presetOptions = $derived(
    presetList.map((p) => ({
      id: p.id,
      title: p.name,
      description: `${p.description} · ${p.mode === 'size' ? `${Math.round(p.sizeMB)} MB` : `CRF ${p.crf}`}`,
    })),
  )

  const appView = $derived(info ? 'queue' : 'empty')

  const originalMB = $derived(info ? info.sizeMB : 0)
  const savedPct = $derived(done && info && info.sizeMB > 0 ? (1 - done.sizeMB / info.sizeMB) * 100 : 0)

  onMount(() => {
    load()
    EventsOn('compress:progress', (e: ProgressEv) => {
      if (e.jobId !== currentJobId) return
      percent = e.percent
      stage = e.stage
    })
    EventsOn('compress:done', (e: DoneEv) => {
      if (e.jobId !== currentJobId) return
      partsDone = [...partsDone, e]
      if (partsDone.length < jobTotalParts) {
        percent = (partsDone.length / jobTotalParts) * 100
        return
      }
      running = false
      percent = 100
      stage = 'done'
      done = { jobId: e.jobId, outputPath: partsDone[0].outputPath, sizeMB: 0, sizeBytes: 0 }
      const total = partsDone.reduce((acc, p) => acc + p.sizeBytes, 0)
      done.sizeBytes = total
      done.sizeMB = total / (1024 * 1024)
      currentJobId = ''
    })
    EventsOn('compress:error', (e: ErrorEv) => {
      if (e.jobId !== currentJobId) return
      running = false
      error = e.error
      currentJobId = ''
    })
    EventsOn('wails:file-drop', (_x: number, _y: number, paths: string[]) => {
      handleDrop(paths)
    })
    return () => {
      EventsOff('compress:progress')
      EventsOff('compress:done')
      EventsOff('compress:error')
      EventsOff('wails:file-drop')
    }
  })

  async function load() {
    try {
      const pres = await GetPresets()
      presetList = pres
      const p = pres.find((x) => x.id === selectedPresetId)
      if (p?.mode === 'size') sizeMB = p.sizeMB
    } catch (err) {
      error = 'falha ao carregar presets: ' + err
    }

    try {
      const st = await CheckFFmpeg()
      ffmpegOk = st.ffmpegOK
      ffmpegMsg = st.message
    } catch {
      ffmpegOk = false
      ffmpegMsg = 'não foi possível verificar o ffmpeg'
    }
  }

  async function pickInput() {
    const path = await OpenInputDialog()
    if (!path) return
    await adoptFile(path)
  }

  async function pickOutput() {
    const suggested = outputPath.split('/').pop() || 'comprimido.mp4'
    const path = await OpenOutputDialog(suggested)
    if (!path) return
    outputPath = path
  }

  function selectPresetById(id: string) {
    const p = presetList.find((x) => x.id === id)
    if (!p) return
    selectedPresetId = p.id
    if (p.mode === 'size') sizeMB = p.sizeMB
    else crf = p.crf
  }

  function canRun(): boolean {
    return (
      !running && !!inputPath && !!outputPath && !!selectedPreset && ffmpegOk && !splitBlocked
    )
  }

  function splitPayload(): { parts: number; minutesEach: number } | null {
    if (!splitOn || splitBlocked || !activeSplit) return null
    return splitAxis === 'parts'
      ? { parts: Math.floor(splitParts), minutesEach: 0 }
      : { parts: 0, minutesEach: splitMinutes }
  }

  async function compress() {
    if (!canRun()) return
    error = ''
    done = null
    partsDone = []
    jobTotalParts = activeSplit && !splitBlocked ? activeSplit.count : 1
    percent = 0
    stage = 'queue'
    running = true
    try {
      const jobId = await StartCompress(new bindings.Job({
        inputPath,
        outputPath,
        presetId: selectedPresetId,
        sizeMB: sizeMB > 0 ? sizeMB : 10,
        crf,
        split: splitPayload() ?? undefined,
      }))
      currentJobId = jobId
    } catch (err) {
      running = false
      error = String(err)
    }
  }

  async function cancel() {
    if (currentJobId) await CancelJob(currentJobId)
    currentJobId = ''
    running = false
    error = 'cancelado pelo usuário'
  }

  function reset() {
    inputPath = ''
    info = null
    thumb = ''
    outputPath = ''
    done = null
    partsDone = []
    jobTotalParts = 1
    error = ''
    percent = 0
    stage = ''
  }
</script>

<svelte:window ondragover={onDragOver} ondragleave={onDragLeave} ondrop={onDropEnd} />

<header class="topbar">
  <span class="ffmpeg-pill" class:bad={!ffmpegOk}>
    <span class="dot"></span>
    {ffmpegOk ? 'ffmpeg OK' : 'ffmpeg ausente'}
  </span>
  <span class="topbar-title">vidctl</span>
  <div class="topbar-end">
    <button class="icon-btn" onclick={() => (prefsOpen = true)} aria-label="Preferências" title="Preferências">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <path d="M4 7h16M4 12h16M4 17h16" />
      </svg>
    </button>
  </div>
</header>

{#if !ffmpegOk}
  <div class="banner">
    <p>{ffmpegMsg || 'ffmpeg não encontrado. Instale o ffmpeg e clique em verificar de novo.'}</p>
    <button class="btn subtle small" onclick={() => load()}>verificar de novo</button>
  </div>
{/if}

{#if appView === 'empty'}
  <StatusPage
    title="Comprimir Vídeos"
    subtitle="Arraste e solte vídeos aqui"
    actionLabel="Abrir…"
    onAction={pickInput}
    hot={dragging}
  />
{:else}
  <div class="split">
    <aside class="sidebar">
      {#if selectedPreset?.mode === 'size'}
        <section class="group-card">
          <div class="group-card-head">
            <span class="group-card-title">Tamanho alvo</span>
            <InfoTip text="Presets de tamanho fazem o ffmpeg calcular o bitrate pela duração para caber no alvo (encode 2-pass)." />
          </div>
          <div class="group-card-body">
            <div class="tune-row">
              <span class="meta-k">tamanho (MB)</span>
              <Stepper value={sizeMB} min={2} max={100} digits={1} ariaLabel="tamanho alvo" onchange={(v) => (sizeMB = v)} />
            </div>
            <div class="hint mono">
              {#if info}
                {#if advice && advice.kbps > 0}
                  ≈ video {advice.kbps} kbps · áudio 96k{#if splitOn} · por parte{/if}
                {:else}
                  ≈ video {Math.max(1, Math.round(((sizeMB * 8 * 1024 * 1024 * 0.95) / info.durationSec - 96000) / 1000))} kbps · áudio 96k
                {/if}
              {/if}
            </div>
            {#if advice && !advice.ok}
              <div class="advice">
                <p class="advice-msg">{advice.message}</p>
                {#if (advice.suggestion.sizeMB ?? 0) > 0 || (advice.suggestion.minutesEach ?? 0) > 0}
                  <button class="btn small" onclick={applyAdvice}>aplicar sugestão</button>
                {/if}
              </div>
            {/if}
          </div>
        </section>
      {:else if selectedPreset}
        <section class="group-card">
          <div class="group-card-head">
            <span class="group-card-title">Qualidade</span>
            <InfoTip text="Encode CRF: qualidade constante sem limite de tamanho. Quanto menor o CRF, melhor a imagem e maior o arquivo." />
          </div>
          <div class="group-card-body">
            <div class="tune-row">
              <span class="meta-k">CRF — quanto menor, melhor</span>
              <Stepper value={crf} min={16} max={34} ariaLabel="CRF" onchange={(v) => (crf = v)} />
            </div>
          </div>
        </section>
      {/if}

      <section class="group-card">
        <div class="group-card-head">
          <span class="group-card-title">Presets</span>
          <InfoTip text="Escolha um preset de saída. Presets de tamanho calculam o bitrate pela duração para caber no alvo (2-pass); presets CRF priorizam qualidade." />
        </div>
        <div class="group-card-body">
          <RadioCardGroup options={presetOptions} selected={selectedPresetId} name="presets" onchange={selectPresetById} />
        </div>
      </section>

      <section class="group-card">
        <div class="group-card-head">
          <span class="group-card-title">Cortar em partes</span>
          <InfoTip text="Divide o vídeo em partes sequenciais de duração fixa. Cada parte passa pela compressão escolhida. Mínimo de 1 minuto por parte; a última pode ficar menor ou levar a sobra. Ajuste um dos contadores para ativar o corte." />
        </div>
        <div class="group-card-body">
          <div class="split-head">
            <div class="split-toggle">
              <button class="btn small" class:solid={!splitOn} onclick={() => (splitOn = false)}>
                sem corte
              </button>
              <button class="btn small" class:solid={splitOn} onclick={() => (splitOn = true)}>
                cortar
              </button>
            </div>
            {#if splitOn && activeSplit && !splitBlocked}
              <span class="split-tag mono">{activeSplit.count} × ~{fmtClock(activeSplit.sliceSec)}</span>
            {/if}
          </div>

          <div class="tune-row" class:active={splitOn && splitAxis === 'parts'} class:dim={!splitOn}>
            <span class="meta-k">partes</span>
            <Stepper value={splitParts} min={2} max={60} ariaLabel="partes" onchange={setParts} />
          </div>

          <div class="tune-row" class:active={splitOn && splitAxis === 'minutes'} class:dim={!splitOn}>
            <span class="meta-k">min/parte</span>
            <Stepper value={splitMinutes} min={1} max={60} digits={1} ariaLabel="minutos por parte" onchange={setMinutes} />
          </div>

          {#if splitOn && !info}
            <div class="hint">escolha o vídeo para calcular o corte</div>
          {:else if splitOn && activeSplit?.error}
            <div class="split-error mono">{activeSplit.error}</div>
          {/if}
        </div>
      </section>

      <section class="group-card">
        <div class="group-card-head">
          <span class="group-card-title">Saída</span>
        </div>
        <div class="group-card-body">
          <div class="out-row">
            <span class="out-path mono" class:empty={!outputPath}>
              {outputPath || 'escolha o vídeo para gerar o caminho de saída'}
            </span>
            <button class="btn subtle small" onclick={pickOutput} disabled={!inputPath}>salvar como…</button>
          </div>
        </div>
      </section>
    </aside>

    <main class="content dropzone" class:hot={dragging}>
      <div class="sources-head">
        <span class="group-card-title">Fontes de vídeo</span>
        {#if info}
          <button class="btn subtle small" onclick={reset}>limpar</button>
        {/if}
      </div>

      {#if info}
        <VideoRow
          {info}
          status={running ? 'progress' : done ? 'done' : error ? 'error' : 'idle'}
          {stage}
          {percent}
          {savedPct}
          {thumb}
          onSwitch={pickInput}
          onClear={reset}
        />
      {/if}

      {#if running}
        <div class="progress-card">
          <div class="progress-head">
            <span class="mono">{stageLabel(stage) || 'PROCESSANDO'}</span>
            <span class="progress-pct mono">{percent.toFixed(1)}%</span>
          </div>
          <div class="track"><div class="fill" style="width:{percent}%"></div></div>
          <div class="progress-foot">
            <button class="btn small" onclick={cancel}>cancelar</button>
            <div class="usage mono" aria-live="polite">
              {#if usage}
                CPU {Math.round(usage.cpu)}% · RAM {(usage.memUsedMB / 1024).toFixed(1)}/{(usage.memTotalMB / 1024).toFixed(1)} GB
                {#if usage.ffmpegCpu > 0.5}· ffmpeg {Math.round(usage.ffmpegCpu)}%{/if}
                {#if usage.gpu >= 0}· GPU {Math.round(usage.gpu)}%{/if}
              {:else}
                medindo uso…
              {/if}
            </div>
          </div>
        </div>
      {:else if done}
        <div class="result">
          <div class="result-big">
            <span class="result-pct mono">−{savedPct.toFixed(0)}%</span>
            <span class="result-size mono">
              {done.sizeMB.toFixed(1)} MB{#if jobTotalParts > 1} · {jobTotalParts} partes{/if}
            </span>
            <span class="result-from mono">de {originalMB.toFixed(1)} MB</span>
          </div>
          <div class="result-actions">
            <button class="btn solid" onclick={() => OpenFolder(done!.outputPath)}>abrir pasta</button>
            <button class="btn" onclick={reset}>novo vídeo</button>
          </div>
        </div>
      {/if}

      {#if info && !running && !done}
        <div class="summary-card">
          <div class="sum-row">
            <span class="meta-k">destino</span>
            <span class="sum-v">
              {#if selectedPreset}
                {selectedPreset.name} · {selectedPreset.mode === 'size' ? `${sizeMB} MB${splitOn ? '/parte' : ''}` : `CRF ${crf}`}
              {:else}
                —
              {/if}
            </span>
          </div>
          <div class="sum-row">
            <span class="meta-k">corte</span>
            <span class="sum-v">
              {#if splitOn && activeSplit && !splitBlocked}
                {activeSplit.count} × ~{fmtClock(activeSplit.sliceSec)}
              {:else if splitOn}
                ajuste o corte
              {:else}
                sem corte
              {/if}
            </span>
          </div>
          {#if advice && advice.kbps > 0}
            <div class="sum-row">
              <span class="meta-k">bitrate</span>
              <span class="sum-v">≈{advice.kbps} kbps{#if splitOn} / parte{/if} · mín {advice.minKbps}</span>
            </div>
          {/if}
        </div>
      {/if}

      {#if error}
        <div class="alert">
          <p class="mono">{error}</p>
        </div>
      {/if}

      {#if !running && !done}
        <div class="actionbar">
          <button class="pill-btn" onclick={compress} disabled={!canRun()}>Comprimir…</button>
        </div>
      {/if}
    </main>
  </div>
{/if}

<footer class="foot mono">
  offline · h264 + aac · 2-pass quando há limite de tamanho
</footer>

<PreferencesModal
  open={prefsOpen}
  {theme}
  {ffmpegOk}
  {ffmpegMsg}
  onclose={() => (prefsOpen = false)}
  ontheme={setTheme}
  onRecheck={() => load()}
/>
