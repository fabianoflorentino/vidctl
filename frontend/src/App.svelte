<script lang="ts">
  import { onMount } from 'svelte'
  import {
    CheckFFmpeg,
    GetPresets,
    OpenInputDialog,
    OpenMultipleDialog,
    GetMediaInfo,
    GetThumbnail,
    OpenOutputDialog,
    Compress as StartCompress,
    CompressMultiple,
    DebugEnabled,
    DebugLog,
    GetTasks,
    ClearFinished,
    Cancel as CancelJob,
    OpenFolder,
    GetUsage,
    GetAdvice,
    GetConfig,
    SaveConfig,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js'
  import type { main, presets, media, config } from '../wailsjs/go/models.js'
  import { compress as bindings, config as configBindings } from '../wailsjs/go/models.js'
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
  type QueuedEv = { jobId: string; position: number }
  type StartEv = { jobId: string }
  type QueueState = 'queued' | 'running' | 'done' | 'error' | 'canceled'
  type QueueItem = {
    jobId: string
    label: string
    inputPath: string
    outputPath: string
    presetId: string
    state: QueueState
    position: number
    stage: string
    percent: number
    sizeBytes: number
    partsTotal: number
    partsDone: number
    error: string
  }
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

  let outputDir = $state('')
  let ffmpegPath = $state('')
  let ffprobePath = $state('')
  let configReady = $state(false)
  let saveTimer: ReturnType<typeof setTimeout> | undefined

  const CONFIG_DEFAULTS: config.Config = new configBindings.Config({
    presetId: 'whatsapp-status',
    sizeMB: 10,
    crf: 23,
    outputDir: '',
    ffmpegPath: '',
    ffprobePath: '',
    language: 'pt',
    maxParallel: 1,
    notifyOnDone: false,
    openFolderOnDone: false,
  })

  function currentConfig(): config.Config {
    return new configBindings.Config({
      ...CONFIG_DEFAULTS,
      presetId: selectedPresetId,
      sizeMB,
      crf,
      outputDir,
      ffmpegPath,
      ffprobePath,
    })
  }

  // Evita gravar de volta o que acabou de ser lido e writes em mudanças que não
  // alteram nada (o Stepper emite o mesmo valor ao receber foco).
  let lastSavedConfig = ''

  function scheduleSaveConfig() {
    if (!configReady) return
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(async () => {
      saveTimer = undefined
      const snapshot = JSON.stringify(currentConfig())
      if (snapshot === lastSavedConfig) return
      try {
        await SaveConfig(currentConfig())
        lastSavedConfig = snapshot
      } catch (err) {
        error = 'falha ao salvar as preferências: ' + err
      }
    }, 500)
  }

  $effect(() => {
    currentConfig()
    scheduleSaveConfig()
  })

  function suggestedOutput(base: string): string {
    const name = base + '-compressed.mp4'
    if (!outputDir) return name
    const sep = outputDir.includes('\\') && !outputDir.includes('/') ? '\\' : '/'
    return outputDir.replace(/[\\/]+$/, '') + sep + name
  }

  let queue = $state<QueueItem[]>([])
  let error = $state('')
  let debug = false

  function dbg(msg: string) {
    if (!debug) return
    DebugLog(msg).catch(() => {})
  }
  let usage = $state<Awaited<ReturnType<typeof GetUsage>> | null>(null)
  let advice = $state<Awaited<ReturnType<typeof GetAdvice>> | null>(null)
  let adviceSeq = 0

  type EventPatch = { allowed: QueueState[]; fn: (item: QueueItem) => void }
  const bufferedEvents = new Map<string, EventPatch[]>()

  function renumber() {
    let pos = 0
    for (const item of queue) {
      if (item.state === 'queued' || item.state === 'running') {
        pos++
        item.position = pos
      } else {
        item.position = 0
      }
    }
  }

  function applyPatch(item: QueueItem, p: EventPatch) {
    if (!p.allowed.includes(item.state)) {
      dbg(`patch rejeitado job=${item.jobId} estado=${item.state} permitido=${p.allowed.join(',')}`)
      return
    }
    p.fn(item)
    dbg(`patch job=${item.jobId} estado=${item.state} percent=${item.percent} stage=${item.stage}`)
    renumber()
  }

  function patch(jobId: string, allowed: QueueState[], fn: (item: QueueItem) => void) {
    const item = queue.find((i) => i.jobId === jobId)
    if (item) {
      applyPatch(item, { allowed, fn })
      return
    }
    dbg(`patch aguardando item job=${jobId} permitido=${allowed.join(',')}`)
    const list = bufferedEvents.get(jobId) ?? []
    list.push({ allowed, fn })
    bufferedEvents.set(jobId, list)
  }

  function createItem(fields: Partial<QueueItem> & { jobId: string; inputPath: string; presetId: string }) {
    const item: QueueItem = {
      label: '',
      outputPath: '',
      state: 'queued',
      position: 0,
      stage: 'queue',
      percent: 0,
      sizeBytes: 0,
      partsTotal: 1,
      partsDone: 0,
      error: '',
      ...fields,
    }
    queue.push(item)
    const buffered = bufferedEvents.get(item.jobId) ?? []
    bufferedEvents.delete(item.jobId)
    for (const p of buffered) applyPatch(item, p)
    renumber()
    dbg(`item criado job=${item.jobId} estado=${item.state} percent=${item.percent} buffer=${buffered.length}`)
  }

  function fromTask(t: bindings.TaskStatus): QueueItem {
    return {
      jobId: t.jobId,
      label: t.label,
      inputPath: t.inputPath,
      outputPath: t.outputPath,
      presetId: t.presetId,
      state: t.state as QueueState,
      position: t.position,
      stage: t.stage,
      percent: t.percent,
      sizeBytes: t.sizeBytes,
      partsTotal: t.partsTotal,
      partsDone: t.partsDone,
      error: t.error,
    }
  }

  function fileName(path: string): string {
    return path.split(/[\\/]/).pop() || path
  }

  function presetName(id: string): string {
    return presetList.find((p) => p.id === id)?.name ?? id
  }

  function stateLabel(item: QueueItem): string {
    switch (item.state) {
      case 'queued':
        return item.position > 1 ? `aguardando · ${item.position}º da fila` : 'aguardando'
      case 'running':
        return `processando · ${item.percent.toFixed(0)}%`
      case 'done':
        return 'concluído'
      case 'error':
        return 'erro'
      case 'canceled':
        return 'cancelado'
    }
  }

  const previewItem = $derived(queue.filter((i) => i.inputPath === inputPath).at(-1) ?? null)
  const previewState = $derived(previewItem?.state ?? 'idle')
  const previewBusy = $derived(previewState === 'queued' || previewState === 'running')
  const previewDone = $derived(previewState === 'done')
  const anyBusy = $derived(queue.some((i) => i.state === 'queued' || i.state === 'running'))
  const hasFinished = $derived(queue.some((i) => i.state !== 'queued' && i.state !== 'running'))

  let prefsOpen = $state(false)
  let dragging = $state(false)

  const VIDEO_EXT = ['.mp4', '.mov', '.mkv', '.webm', '.m4v', '.avi', '.wmv', '.flv']

  function isVideoPath(p: string): boolean {
    const low = p.toLowerCase()
    return VIDEO_EXT.some((ext) => low.endsWith(ext))
  }

  async function adoptFile(path: string) {
    if (!isVideoPath(path)) {
      error = `extensão não suportada: ${path.split('.').pop() || path}`
      return
    }
    inputPath = path
    outputPath = ''
    error = ''
    thumb = ''
    try {
      info = await GetMediaInfo(path)
      const base = path.replace(/\.[^.]+$/, '')
      outputPath = suggestedOutput(base)
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
    const videos = paths.filter(isVideoPath)
    if (videos.length > 1) {
      void enqueuePaths(videos)
      return
    }
    const first = videos[0] ?? paths[0]
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
    const busy = previewBusy
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
    if (!anyBusy) {
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
  const savedPct = $derived(
    previewDone && previewItem && info && info.sizeMB > 0
      ? (1 - previewItem.sizeBytes / (1024 * 1024) / info.sizeMB) * 100
      : 0,
  )

  onMount(() => {
    load()
    DebugEnabled()
      .then((v) => (debug = v))
      .catch(() => {})
    window.addEventListener('error', (ev) => {
      dbg(`ERRO JS: ${ev.message} em ${ev.filename}:${ev.lineno}:${ev.colno}`)
    })
    window.addEventListener('unhandledrejection', (ev) => {
      dbg(`rejeição JS: ${String(ev.reason)}`)
    })
    EventsOn('compress:queued', (e: QueuedEv) => {
      dbg(`evento compress:queued job=${e.jobId} pos=${e.position}`)
      patch(e.jobId, ['queued'], () => {})
    })
    EventsOn('compress:start', (e: StartEv) => {
      dbg(`evento compress:start job=${e.jobId}`)
      patch(e.jobId, ['queued', 'running'], (i) => {
        i.state = 'running'
      })
    })
    EventsOn('compress:progress', (e: ProgressEv) => {
      dbg(`evento compress:progress job=${e.jobId} stage=${e.stage} percent=${e.percent}`)
      patch(e.jobId, ['running'], (i) => {
        i.percent = e.percent
        i.stage = e.stage
        const part = /parte (\d+)\/(\d+)/.exec(e.stage)
        if (part) i.partsTotal = Number(part[2])
      })
    })
    EventsOn('compress:done', (e: DoneEv) => {
      dbg(`evento compress:done job=${e.jobId} sizeBytes=${e.sizeBytes}`)
      patch(e.jobId, ['running'], (i) => {
        i.outputPath = e.outputPath
        i.sizeBytes += e.sizeBytes
        i.partsDone++
        if (i.partsDone < i.partsTotal) {
          i.percent = (i.partsDone / i.partsTotal) * 100
          return
        }
        i.state = 'done'
        i.percent = 100
        i.stage = 'done'
      })
    })
    EventsOn('compress:error', (e: ErrorEv) => {
      dbg(`evento compress:error job=${e.jobId} error=${e.error}`)
      patch(e.jobId, ['queued', 'running'], (i) => {
        i.state = 'error'
        i.error = e.error
        i.stage = ''
      })
    })
    EventsOn('wails:file-drop', (_x: number, _y: number, paths: string[]) => {
      handleDrop(paths)
    })
    return () => {
      EventsOff('compress:queued')
      EventsOff('compress:start')
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
    } catch (err) {
      error = 'falha ao carregar presets: ' + err
    }

    await loadConfig()
    await checkFFmpeg()
    await refreshQueue()
  }

  async function refreshQueue() {
    try {
      const tasks = await GetTasks()
      dbg(`refreshQueue -> ${tasks.length} tarefas`)
      if (tasks.length) queue = tasks.map(fromTask)
    } catch {
      /* fila indisponível: a sessão segue com a fila vazia */
    }
  }

  async function checkFFmpeg() {
    try {
      const st = await CheckFFmpeg()
      ffmpegOk = st.ffmpegOK
      ffmpegMsg = st.message
    } catch {
      ffmpegOk = false
      ffmpegMsg = 'não foi possível verificar o ffmpeg'
    }
  }

  async function loadConfig() {
    let saved: config.Config | undefined
    try {
      saved = await GetConfig()
    } catch (err) {
      error = 'falha ao carregar as preferências: ' + err
      configReady = true
      return
    }
    if (saved) {
      // Só aceita o preset se ele ainda existir na lista; um preset removido
      // numa versão futura não pode deixar a sidebar sem seleção.
      if (presetList.some((p) => p.id === saved.presetId)) {
        selectedPresetId = saved.presetId
      }
      // sizeMB 0 significa "sem alvo salvo" e cai no tamanho do preset.
      const hasSavedSize = saved.sizeMB > 0
      if (hasSavedSize) sizeMB = saved.sizeMB
      if (saved.crf > 0) crf = saved.crf
      outputDir = saved.outputDir ?? ''
      ffmpegPath = saved.ffmpegPath ?? ''
      ffprobePath = saved.ffprobePath ?? ''

      const p = presetList.find((x) => x.id === selectedPresetId)
      if (p?.mode === 'size' && !hasSavedSize) sizeMB = p.sizeMB
    }
    lastSavedConfig = JSON.stringify(currentConfig())
    configReady = true
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
    if (!ffmpegOk || !inputPath || !outputPath || !selectedPreset || splitBlocked) return false
    return !queue.some(
      (i) => i.inputPath === inputPath && (i.state === 'queued' || i.state === 'running'),
    )
  }

  function splitPayload(): { parts: number; minutesEach: number } | null {
    if (!splitOn || splitBlocked || !activeSplit) return null
    return splitAxis === 'parts'
      ? { parts: Math.floor(splitParts), minutesEach: 0 }
      : { parts: 0, minutesEach: splitMinutes }
  }

  function jobParts(): number {
    return splitPayload() && activeSplit ? activeSplit.count : 1
  }

  function buildJob(input: string, output: string): bindings.Job {
    return new bindings.Job({
      inputPath: input,
      outputPath: output,
      presetId: selectedPresetId,
      sizeMB: sizeMB > 0 ? sizeMB : 10,
      crf,
      split: splitPayload() ?? undefined,
    })
  }

  async function compress() {
    if (!canRun()) return
    error = ''
    dbg(`compress() input=${inputPath}`)
    try {
      const jobId = await StartCompress(buildJob(inputPath, outputPath))
      dbg(`StartCompress ok job=${jobId}`)
      createItem({
        jobId,
        label: fileName(inputPath),
        inputPath,
        outputPath,
        presetId: selectedPresetId,
        partsTotal: jobParts(),
      })
    } catch (err) {
      dbg(`compress() erro=${String(err)}`)
      error = String(err)
    }
  }

  async function enqueuePaths(paths: string[]) {
    const active = new Set(
      queue.filter((i) => i.state === 'queued' || i.state === 'running').map((i) => i.inputPath),
    )
    const usable = paths.filter(isVideoPath).filter((p) => !active.has(p))
    if (!usable.length) return

    const parts = jobParts()
    const jobs = usable.map((p) => buildJob(p, suggestedOutput(p.replace(/\.[^.]+$/, ''))))
    error = ''
    dbg(`enqueuePaths ${usable.length} vídeos`)
    try {
      const acks = await CompressMultiple(jobs)
      dbg(`CompressMultiple ok acks=${acks.length}`)
      if (usable[0]) await adoptFile(usable[0])
      acks.forEach((ack, idx) => {
        if (ack.error) {
          dbg(`ack com erro: ${ack.error}`)
          error = ack.error
          return
        }
        const job = jobs[idx]
        createItem({
          jobId: ack.jobId,
          label: fileName(job.inputPath),
          inputPath: job.inputPath,
          outputPath: job.outputPath,
          presetId: job.presetId,
          partsTotal: parts,
        })
      })
    } catch (err) {
      dbg(`enqueuePaths erro=${String(err)}`)
      error = String(err)
    }
  }

  async function addVideos() {
    const paths = await OpenMultipleDialog()
    if (!paths?.length) return
    await enqueuePaths(paths)
  }

  async function cancelItem(jobId: string) {
    try {
      await CancelJob(jobId)
    } catch (err) {
      error = 'falha ao cancelar: ' + err
      return
    }
    patch(jobId, ['queued', 'running'], (i) => {
      i.state = 'canceled'
      i.stage = ''
    })
  }

  async function clearFinished() {
    try {
      const remaining = await ClearFinished()
      queue = remaining.map(fromTask)
    } catch (err) {
      queue = queue.filter((i) => i.state === 'queued' || i.state === 'running')
      error = 'falha ao limpar a fila: ' + err
    }
  }

  function reset() {
    inputPath = ''
    info = null
    thumb = ''
    outputPath = ''
    error = ''
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
          status={previewBusy ? 'progress' : previewState === 'done' ? 'done' : previewState === 'error' ? 'error' : 'idle'}
          stage={previewItem?.stage ?? ''}
          percent={previewItem?.percent ?? 0}
          {savedPct}
          {thumb}
          onSwitch={pickInput}
          onClear={reset}
        />
      {/if}

      {#if previewItem && previewBusy}
        <div class="progress-card">
          <div class="progress-head">
            <span class="mono">
              {previewItem.state === 'queued' && previewItem.position > 1
                ? `AGUARDANDO · ${previewItem.position}º DA FILA`
                : stageLabel(previewItem.stage) || 'PROCESSANDO'}
            </span>
            <span class="progress-pct mono">{previewItem.percent.toFixed(1)}%</span>
          </div>
          <div class="track"><div class="fill" style="width:{previewItem.percent}%"></div></div>
          <div class="progress-foot">
            <button class="btn small" onclick={() => cancelItem(previewItem.jobId)}>cancelar</button>
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
      {:else if previewItem && previewDone}
        <div class="result">
          <div class="result-big">
            <span class="result-pct mono">−{savedPct.toFixed(0)}%</span>
            <span class="result-size mono">
              {(previewItem.sizeBytes / (1024 * 1024)).toFixed(1)} MB{#if previewItem.partsTotal > 1} · {previewItem.partsTotal} partes{/if}
            </span>
            <span class="result-from mono">de {originalMB.toFixed(1)} MB</span>
          </div>
          <div class="result-actions">
            <button class="btn solid" onclick={() => OpenFolder(previewItem.outputPath)}>abrir pasta</button>
            <button class="btn" onclick={reset}>novo vídeo</button>
          </div>
        </div>
      {/if}

      {#if queue.length > 0}
        <div class="queue-card">
          <div class="queue-head">
            <span class="group-card-title">Fila de conversões</span>
            <span class="queue-count mono">{queue.length} {queue.length === 1 ? 'vídeo' : 'vídeos'}</span>
            {#if hasFinished}
              <button class="btn subtle small" onclick={clearFinished}>limpar fila</button>
            {/if}
          </div>
          <ul class="queue-list">
            {#each queue as item (item.jobId)}
              <li class="queue-item" class:active={item.state === 'running'} class:muted={item.state === 'canceled'}>
                <span class="qi-dot" data-state={item.state}></span>
                <div class="qi-body">
                  <div class="qi-line">
                    <span class="qi-label" title={item.inputPath}>{item.label}</span>
                    <span class="qi-preset mono">{presetName(item.presetId)}</span>
                    <span class="qi-state mono">{stateLabel(item)}</span>
                  </div>
                  {#if item.state === 'running'}
                    <div class="track thin"><div class="fill" style="width:{item.percent}%"></div></div>
                  {/if}
                  {#if item.error}
                    <div class="qi-error mono">{item.error}</div>
                  {/if}
                </div>
                {#if item.state === 'queued' || item.state === 'running'}
                  <button class="btn subtle small" onclick={() => cancelItem(item.jobId)}>cancelar</button>
                {/if}
              </li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if info && !previewBusy && !previewDone}
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

      <div class="actionbar">
        <button class="pill-btn" onclick={compress} disabled={!canRun()}>Comprimir…</button>
        <button class="btn" onclick={addVideos} disabled={!ffmpegOk}>Adicionar vídeos…</button>
      </div>
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
  {outputDir}
  {ffmpegPath}
  {ffprobePath}
  onclose={() => (prefsOpen = false)}
  ontheme={setTheme}
  onRecheck={checkFFmpeg}
  onoutputdir={(v) => (outputDir = v)}
  onffmpegpath={(v) => (ffmpegPath = v)}
  onffprobepath={(v) => (ffprobePath = v)}
/>
