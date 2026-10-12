<script lang="ts">
  import { onMount } from 'svelte'
  import {
    CheckFFmpeg,
    GetPresets,
    OpenInputDialog,
    GetMediaInfo,
    GetThumbnail,
    OpenOutputDialog,
    OpenThumbnailDialog,
    Compress as StartCompress,
    DebugEnabled,
    DebugLog,
    GetTasks,
    ClearFinished,
    Cancel as CancelJob,
    OpenFolder,
    GetUsage,
    GetAdvice,
    EstimateSize,
    GetConfig,
    SaveConfig,
    GetEncoders,
    RefreshEncoders,
    ExtractAudio,
    OpenAudioDialog,
    SystemTheme,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff, Environment } from '../wailsjs/runtime/runtime.js'
  import type { main, presets, media, config, encode, audio } from '../wailsjs/go/models.js'
  import { compress as bindings, config as configBindings } from '../wailsjs/go/models.js'
  import StatusPage from './lib/StatusPage.svelte'
  import VideoRow from './lib/VideoRow.svelte'
  import Stepper from './lib/Stepper.svelte'
import RadioCardGroup from './lib/RadioCardGroup.svelte'
  import Toggle from './lib/Toggle.svelte'
  import Section from './lib/Section.svelte'
  import PreferencesModal from './lib/PreferencesModal.svelte'
  import AudioPanel from './lib/AudioPanel.svelte'
  import { stageLabel } from './lib/stages'
  import { t, setLang, langId, type LangId } from './lib/i18n.svelte'

  type ProgressEv = { jobId: string; stage: string; percent: number }
  type DoneEv = { jobId: string; outputPath: string; sizeMB: number; sizeBytes: number }
  type ErrorEv = { jobId: string; error: string }
  type QueuedEv = { jobId: string; position: number }
  type StartEv = { jobId: string }
  type QueueState = 'queued' | 'running' | 'done' | 'error' | 'canceled'
  type QueueItem = {
    jobId: string
    kind: 'compress' | 'audio'
    meta: string
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
    startedAt: number
    elapsedSec: number
  }
  type ThemeChoice = 'system' | 'light' | 'dark'

  let presetList = $state<presets.Preset[]>([])
  let ffmpegOk = $state(true)
  let ffmpegMsg = $state('')

  let theme = $state<ThemeChoice>('system')
  let language = $state<LangId>('pt')
  let systemDark = $state(false)
  let systemResolved = $state(false)

  $effect(() => {
    setLang(language)
  })

  try {
    const saved = localStorage.getItem('vidctl-theme')
    if (saved === 'light' || saved === 'dark') {
      theme = saved
    }
  } catch {
    /* localStorage indisponível: segue o sistema */
  }

  // O WebKitGTK (Linux) costuma reportar prefers-color-scheme: light mesmo com
  // o desktop escuro; o tema real vem do backend (gsettings/registry/defaults)
  // e o matchMedia só acompanha mudanças ao vivo.
  $effect(() => {
    if (theme !== 'system') return
    if (!systemResolved) return
    document.documentElement.dataset.theme = systemDark ? 'dark' : 'light'
  })

  function setTheme(next: ThemeChoice) {
    theme = next
    if (next === 'system') {
      document.documentElement.dataset.theme = systemDark ? 'dark' : 'light'
    } else {
      document.documentElement.dataset.theme = next
    }
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
  let codec = $state<'h264' | 'h265'>('h264')
  let hardware = $state('')
  let nvencPreset = $state('p4')
  let encoders = $state<encode.Availability | null>(null)
  let encodersError = $state('')

  let outputDir = $state('')
  let ffmpegPath = $state('')
  let ffprobePath = $state('')
  let audioSuffix = $state('-audio')
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
    audioSuffix: '-audio',
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
      audioSuffix,
      language,
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
        error = t('prefs.saveFail', { err: String(err) })
      }
    }, 500)
  }

  $effect(() => {
    currentConfig()
    scheduleSaveConfig()
  })

  function sepFor(dir: string): string {
    return dir.includes('\\') && !dir.includes('/') ? '\\' : '/'
  }

  // Pasta de destino: a preferência salva quando existe; senão, a pasta do
  // arquivo de origem (nunca o diretório de trabalho do app).
  function targetDir(): string {
    if (outputDir) return outputDir.replace(/[\\/]+$/, '') + sepFor(outputDir)
    const last = Math.max(inputPath.lastIndexOf('/'), inputPath.lastIndexOf('\\'))
    return last >= 0 ? inputPath.slice(0, last + 1) : ''
  }

  function suggestedOutput(base: string): string {
    const name = base.split(/[\\/]/).pop() + '-compressed.mp4'
    return targetDir() + name
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
  let sizeEstimate = $state<Awaited<ReturnType<typeof EstimateSize>> | null>(null)
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
      kind: 'compress',
      meta: '',
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
      startedAt: 0,
      elapsedSec: 0,
      ...fields,
    }
    queue.push(item)
    const created = queue[queue.length - 1]
    const buffered = bufferedEvents.get(created.jobId) ?? []
    bufferedEvents.delete(created.jobId)
    for (const p of buffered) applyPatch(created, p)
    renumber()
    dbg(`item criado job=${created.jobId} estado=${created.state} percent=${created.percent} buffer=${buffered.length}`)
  }

  function fromTask(t: bindings.TaskStatus): QueueItem {
    return {
      jobId: t.jobId,
      kind: t.kind === 'audio' ? 'audio' : 'compress',
      meta: '',
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
      startedAt: 0,
      elapsedSec: 0,
    }
  }

  function fileName(path: string): string {
    return path.split(/[\\/]/).pop() || path
  }

  function presetName(id: string): string {
    const key = 'preset.' + id
    const localized = t(key)
    if (localized !== key) return localized
    return presetList.find((p) => p.id === id)?.name ?? id
  }

  function fmtDuration(sec: number): string {
    if (!sec || !Number.isFinite(sec) || sec < 0) return '—'
    const m = Math.floor(sec / 60)
    const s = Math.round(sec % 60)
    return m > 0 ? `${m}min ${String(s).padStart(2, '0')}s` : `${s}s`
  }

  function stateLabel(item: QueueItem): string {
    switch (item.state) {
      case 'queued':
        return item.position > 1 ? t('queue.state.waitingPos', { pos: item.position }) : t('queue.state.waiting')
      case 'running': {
        const time = item.startedAt ? fmtDuration((Date.now() - item.startedAt) / 1000) : ''
        return time ? t('queue.state.runningTime', { pct: item.percent.toFixed(0), t: time }) : t('queue.state.running', { pct: item.percent.toFixed(0) })
      }
      case 'done':
        return item.elapsedSec ? t('queue.state.doneTime', { t: fmtDuration(item.elapsedSec) }) : t('queue.state.done')
      case 'error':
        return t('queue.state.error')
      case 'canceled':
        return t('queue.state.canceled')
    }
  }

  const previewItem = $derived(queue.filter((i) => i.kind === 'compress' && i.inputPath === inputPath).at(-1) ?? null)
  const previewState = $derived(previewItem?.state ?? 'idle')
  const previewBusy = $derived(previewState === 'queued' || previewState === 'running')
  const previewDone = $derived(previewState === 'done')
  const anyBusy = $derived(queue.some((i) => i.state === 'queued' || i.state === 'running'))
  const hasFinished = $derived(queue.some((i) => i.state !== 'queued' && i.state !== 'running'))

  let now = $state(Date.now())
  $effect(() => {
    if (!anyBusy) return
    const id = setInterval(() => (now = Date.now()), 500)
    return () => clearInterval(id)
  })

  let prefsOpen = $state(false)
  let audioSpec = $state<audio.Spec | undefined>(undefined)
  let audioInvalid = $state('')
  let audioOutputPath = $state('')
  let audioManual = $state(false)
  let dragging = $state(false)
  let nativeFileDrop = true

  const VIDEO_EXT = ['.mp4', '.mov', '.mkv', '.webm', '.m4v', '.avi', '.wmv', '.flv']

  function isVideoPath(p: string): boolean {
    const low = p.toLowerCase()
    return VIDEO_EXT.some((ext) => low.endsWith(ext))
  }

  async function adoptFile(path: string) {
    if (!isVideoPath(path)) {
      const msg = `extensão não suportada: ${path.split('.').pop() || path}`
      dbg(`adoptFile rejeitado path=${path}`)
      error = msg
      return
    }
    // Novo vídeo: descarta o ajuste do anterior e volta as opções por arquivo
    // ao default (corte, split, escala, áudio, codec/hardware).
    resetPerFileOptions()
    inputPath = path
    outputPath = ''
    error = ''
    thumb = ''
    try {
      const got = await GetMediaInfo(path)
      dbg(`adoptFile ok path=${path} duration=${got.durationSec} w=${got.width}`)
      info = got
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
      dbg(`adoptFile erro path=${path} err=${String(err)}`)
      info = null
      error = String(err)
    }
  }

  function resetPerFileOptions() {
    scaleMode = 'default'
    scaleCustom = '1280x720'
    trimStart = ''
    trimEnd = ''
    removeAudio = false
    fps = 0
    rotate = 0
    thumbnailOn = false
    thumbnailPath = ''
    splitOn = false
    splitAxis = 'parts'
    splitParts = 6
    splitMinutes = 5
    codec = 'h264'
    hardware = ''
    nvencPreset = 'p4'
    audioOutputPath = ''
    audioManual = false
  }

  function handleDrop(paths: string[]) {
    // Modelo de alvo único: soltar vídeo troca o atual, sem enfileirar lote.
    const first = paths.find(isVideoPath) ?? paths[0]
    if (first) adoptFile(first)
  }

  function isFileDrag(t: DataTransfer): boolean {
    const types = t.types
    return types.includes('Files') || types.includes('text/uri-list')
  }

  function onDragEnter(e: DragEvent) {
    if (!e.dataTransfer || !isFileDrag(e.dataTransfer)) return
    e.preventDefault()
    dragging = true
  }

  function onDragOver(e: DragEvent) {
    if (!e.dataTransfer || !isFileDrag(e.dataTransfer)) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
    dragging = true
  }

  function onDragLeave(e: DragEvent) {
    if (e.relatedTarget === null) dragging = false
  }

  // No Linux o drop nativo do Wails faz o WebKit navegar para o arquivo solto
  // ("tocar" o vídeo e derrubar a UI). Lá, o front lê text/uri-list do próprio
  // evento de drop, que entrega os caminhos sem navegação.
  function handleWindowDrop(e: DragEvent) {
    if (!e.dataTransfer || !isFileDrag(e.dataTransfer)) return
    e.preventDefault()
    dragging = false
    if (nativeFileDrop) return
    try {
      const uri = e.dataTransfer.getData('text/uri-list') ?? ''
      const paths = uri
        .split('\n')
        .map((p) => p.trim())
        .filter((p) => p && !p.startsWith('#'))
        .map((p) => (p.startsWith('file://') ? decodeURIComponent(p.slice(7)) : p))
      const first = e.dataTransfer.files.length ? (e.dataTransfer.files[0] as { path?: string }).path : undefined
      const dropped = paths.length ? paths : first ? [first] : []
      dbg(`drop uri=${JSON.stringify(uri)} paths=${JSON.stringify(dropped)}`)
      if (dropped.length) handleDrop(dropped)
    } catch (err) {
      dbg(`drop erro ${String(err)}`)
    }
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
      sizeEstimate = null
      return
    }
    const seq = ++adviceSeq
    const timer = setTimeout(async () => {
      const job = new bindings.Job({
        inputPath: path,
        outputPath: 'advice://placeholder',
        presetId,
        sizeMB: targetMB > 0 ? targetMB : 10,
        crf: qualityCRF,
        split: split ?? undefined,
        codec,
        hardware,
        nvencPreset,
        ...adjustPayload(),
      })
      const [got, estimated] = await Promise.allSettled([GetAdvice(job), EstimateSize(job)])
      if (seq !== adviceSeq) return
      advice = got.status === 'fulfilled' ? got.value : null
      sizeEstimate = estimated.status === 'fulfilled' ? estimated.value : null
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

  function fmtHumanSec(sec: number): string {
    const m = Math.floor(sec / 60)
    const s = Math.round(sec % 60)
    return m === 0 ? `${s}s` : `${m}min${String(s).padStart(2, '0')}s`
  }

  // Monta a mensagem do conselho no idioma atual usando o payload estruturado
  // do backend; cai no texto legado (Message) quando o código é desconhecido.
  function adviceMessage(a: Awaited<ReturnType<typeof GetAdvice>>): string {
    if (a.code !== 'low') return a.message
    const p = a.params ?? {}
    const scopeKey = a.scope === 'part' ? 'advice.scopePart' : 'advice.scopeFull'
    let msg = t('advice.lowBase', {
      kbps: Math.round(p.kbps ?? a.kbps),
      minKbps: Math.round(p.minKbps ?? a.minKbps),
      height: Math.round(p.height ?? 0),
      scope: t(scopeKey, { sec: fmtHumanSec(p.sec ?? 0) }),
    })
    if (a.sug === 'size') msg += t('advice.raiseTarget', { mb: Math.round(p.mb ?? 0) })
    else if (a.sug === 'split') msg += t('advice.splitParts', { min: Math.round(p.min ?? 0), parts: Math.round(p.parts ?? 0) })
    else if (a.sug === 'crf') msg += t('advice.useCrf')
    return msg
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
      return { count: 0, sliceSec: 0, error: t('split.tooShort') }
    }
    if (splitAxis === 'parts') {
      const parts = Math.floor(splitParts)
      if (parts < 2) return { count: 0, sliceSec: 0, error: t('split.minParts') }
      if (parts > MAX_PARTS) return { count: 0, sliceSec: 0, error: t('split.maxParts', { n: MAX_PARTS }) }
      const slice = durationSec / parts
      if (slice < MIN_PART_SEC) {
        return { count: 0, sliceSec: 0, error: t('split.partsShort') }
      }
      return { count: parts, sliceSec: slice, error: '' }
    }
    const slice = Math.max(1, splitMinutes) * 60
    const full = Math.floor(durationSec / slice)
    const tail = durationSec - full * slice
    let count = full
    if (tail >= MIN_PART_SEC) count = full + 1
    if (count < 2) return { count: 0, sliceSec: slice, error: t('split.tooFew') }
    if (count > MAX_PARTS) return { count: 0, sliceSec: slice, error: t('split.maxPartsGen', { n: count, n2: MAX_PARTS }) }
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

  type ScaleMode = 'original' | 'default' | 'custom'
  const sectionState = $state({
    config: true,
    avancado: false,
    presets: false,
    split: false,
    ajustes: false,
    audio: false,
  })

  function toggleSection(key: keyof typeof sectionState) {
    const next = !sectionState[key]
    for (const k of Object.keys(sectionState) as (keyof typeof sectionState)[]) sectionState[k] = false
    sectionState[key] = next
  }

  // Scrollbar própria da sidebar: a scrollbar nativa (GTK/WebKit) vaza do
  // painel e cresce demais no hover; aqui controlamos tamanho e posição.
  let sidebarEl = $state<HTMLElement | undefined>(undefined)
  let sbScrollTop = $state(0)
  let sbClientH = $state(0)
  let sbScrollH = $state(0)
  let sbHover = $state(false)
  let sbDragging = $state(false)
  let sbAwake = $state(false)
  let sbSleepTimer: ReturnType<typeof setTimeout> | undefined

  $effect(() => {
    const el = sidebarEl
    if (!el) return
    const measure = () => {
      sbClientH = el.clientHeight
      sbScrollH = el.scrollHeight
    }
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  })

  const sbVisible = $derived(sbScrollH > sbClientH)
  const sbThumbH = $derived(sbVisible ? Math.max(28, (sbClientH * sbClientH) / sbScrollH) : 0)
  const sbThumbTop = $derived(
    sbVisible && sbScrollH > sbClientH ? (sbScrollTop / (sbScrollH - sbClientH)) * (sbClientH - sbThumbH) : 0,
  )
  const sbShow = $derived(sbVisible && (sbHover || sbDragging || sbAwake))

  function onSidebarScroll() {
    if (sidebarEl) sbScrollTop = sidebarEl.scrollTop
    sbAwake = true
    clearTimeout(sbSleepTimer)
    sbSleepTimer = setTimeout(() => (sbAwake = false), 800)
  }

  function onSBDragStart(e: MouseEvent) {
    if (!sidebarEl) return
    sbDragging = true
    const bar = e.currentTarget as HTMLElement
    const rect = bar.getBoundingClientRect()
    const move = (ev: MouseEvent) => {
      const maxTop = sbScrollH - sbClientH
      const ratio = Math.min(1, Math.max(0, (ev.clientY - rect.top) / rect.height))
      sidebarEl!.scrollTop = ratio * maxTop
    }
    const up = () => {
      sbDragging = false
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
    move(e)
  }
  let scaleMode = $state<ScaleMode>('default')
  let scaleCustom = $state('1280x720')
  let trimStart = $state('')
  let trimEnd = $state('')
  let removeAudio = $state(false)
  let fps = $state(0)
  let rotate = $state(0)
  let thumbnailOn = $state(false)
  let thumbnailPath = $state('')

  const scaleOptions = $derived([
    { id: 'default' as ScaleMode, title: t('adjust.default'), description: t('adjust.defaultDesc') },
    { id: 'original' as ScaleMode, title: t('adjust.original'), description: t('adjust.originalDesc') },
    { id: 'custom' as ScaleMode, title: t('adjust.custom'), description: t('adjust.customDesc') },
  ])

  const rotateOptions = $derived([
    { id: '0', title: t('adjust.deg0'), description: t('adjust.deg0Desc') },
    { id: '90', title: t('adjust.deg90'), description: t('adjust.deg90Desc') },
    { id: '180', title: t('adjust.deg180'), description: t('adjust.deg180Desc') },
    { id: '270', title: t('adjust.deg270'), description: t('adjust.deg270Desc') },
  ])

  function parseClock(input: string): number {
    const s = input.trim()
    if (!s) return 0
    const parts = s.split(':')
    if (parts.length > 3) return Number.NaN
    const nums = parts.map((p) => Number(p.replace(',', '.')))
    if (nums.some((n) => Number.isNaN(n) || n < 0)) return Number.NaN
    if (parts.length === 1) return nums[0]
    if (parts.length === 2) return nums[0] * 60 + nums[1]
    return nums[0] * 3600 + nums[1] * 60 + nums[2]
  }

  const trimStartSec = $derived(parseClock(trimStart))
  const trimEndSec = $derived(parseClock(trimEnd))
  const trimming = $derived(trimStart.trim() !== '' || trimEnd.trim() !== '')

  const adjustError = $derived.by<string>(() => {
    if (scaleMode === 'custom' && !/^\s*\d+x\d+\s*$/.test(scaleCustom)) {
      return t('adjust.customScaleErr')
    }
    if (Number.isNaN(trimStartSec) || Number.isNaN(trimEndSec)) {
      return t('adjust.trimInvalid')
    }
    if (trimStartSec > 0 && trimEndSec > 0 && trimEndSec <= trimStartSec) {
      return t('adjust.trimOrder')
    }
    if (trimming && splitOn) {
      return t('adjust.trimSplit')
    }
    if (thumbnailOn && !thumbnailPath) {
      return t('adjust.thumbNeedPath')
    }
    return ''
  })

  const adjustSummary = $derived.by(() => {
    const parts: string[] = []
    if (scaleMode === 'original') parts.push(t('adjust.resOriginal'))
    else if (scaleMode === 'custom') parts.push(scaleCustom.trim())
    if (trimStartSec > 0 || trimEndSec > 0) {
      parts.push(t('adjust.trimRange', { start: trimStartSec > 0 ? fmtClock(trimStartSec) : '0:00', end: trimEndSec > 0 ? fmtClock(trimEndSec) : t('adjust.endPh') }))
    }
    if (fps > 0) parts.push(`${fps} fps`)
    if (rotate !== 0) parts.push(`${rotate}°`)
    if (removeAudio) parts.push(t('adjust.noAudio'))
    if (thumbnailOn) parts.push(t('adjust.thumbShort'))
    return parts.join(' · ')
  })

  function adjustPayload() {
    return {
      scale: scaleMode === 'default' ? '' : scaleMode === 'original' ? 'original' : scaleCustom.trim(),
      trimStartSec: trimStartSec > 0 ? trimStartSec : undefined,
      trimEndSec: trimEndSec > 0 ? trimEndSec : undefined,
      removeAudio,
      fps: fps > 0 ? fps : undefined,
      rotate: rotate !== 0 ? rotate : undefined,
      thumbnailPath: thumbnailOn && thumbnailPath ? thumbnailPath : undefined,
    }
  }

  function suggestedThumbnailPath(): string {
    const base = inputPath ? inputPath.replace(/\.[^.]+$/, '').split(/[\\/]/).pop() : 'video'
    return targetDir() + base + '-thumb.png'
  }

  function toggleThumbnail(v: boolean) {
    thumbnailOn = v
    if (v && !thumbnailPath) thumbnailPath = suggestedThumbnailPath()
    if (!v) thumbnailPath = ''
  }

  async function pickThumbnail() {
    const path = await OpenThumbnailDialog(thumbnailPath.split(/[\\/]/).pop() || 'thumbnail.png')
    if (path) thumbnailPath = path
  }

  const selectedPreset = $derived(presetList.find((p) => p.id === selectedPresetId) ?? null)

  const presetOptions = $derived(
    presetList.map((p) => ({
      id: p.id,
      title: presetName(p.id),
      description: `${tPresetDesc(p.id)} · ${p.mode === 'size' ? t('preset.mb', { mb: Math.round(p.sizeMB) }) : t('preset.crf', { crf: p.crf })}`,
    })),
  )

  function tPresetDesc(id: string): string {
    const key = 'preset.' + id + '.desc'
    const localized = t(key)
    if (localized !== key) return localized
    return presetList.find((p) => p.id === id)?.description ?? ''
  }

  type HardwareOption = { id: string; title: string; description: string; disabled: boolean }

  const hardwareOptions = $derived.by<HardwareOption[]>(() => {
    const opts: HardwareOption[] = [
      { id: '', title: t('adv.software'), description: t('adv.softwareDesc'), disabled: false },
    ]
    opts.push({ id: 'auto', title: t('adv.auto'), description: t('adv.autoDesc'), disabled: false })
    for (const info of encoders?.hardware ?? []) {
      const codecs = info.codecs ?? []
      const labels = codecs.map((c) => c.toUpperCase())
      const desc = labels.length
        ? t('adv.gpuDesc', { codecs: labels.join(' / ') })
        : info.error
          ? t('adv.detectError')
          : t('adv.noCodecs')
      opts.push({
        id: info.id,
        title: info.label,
        description: desc,
        disabled: !codecs.includes(codec),
      })
    }
    return opts
  })

  function shortErr(s: string): string {
    const line = s.split('\n').find((l) => l.trim()) ?? s
    return line.length > 110 ? line.slice(0, 107) + '…' : line
  }

  const failingEncoders = $derived(
    (encoders?.hardware ?? [])
      .filter((i) => (i.codecs ?? []).length === 0 && i.error)
      .map((i) => `${i.label}: ${shortErr(i.error)}`)
      .join(' · '),
  )

  const gpuActive = $derived(hardware !== '')

  const codecOptions = $derived<{ id: "h264" | "h265"; title: string; description: string }[]>([
    { id: 'h264', title: t('adv.h264'), description: t('adv.h264Desc') },
    { id: 'h265', title: t('adv.h265'), description: t('adv.h265Desc') },
  ])

  const nvencPresetOptions = $derived([
    { id: 'p1', title: t('adv.nvencFastest') },
    { id: 'p2', title: t('adv.nvencVeryFast') },
    { id: 'p3', title: t('adv.nvencFast') },
    { id: 'p4', title: t('adv.nvencBalanced') },
    { id: 'p5', title: t('adv.nvencBetter') },
    { id: 'p6', title: t('adv.nvencHigh') },
    { id: 'p7', title: t('adv.nvencMax') },
  ])

  const appView = $derived(info ? 'queue' : 'empty')

  const originalMB = $derived(info ? info.sizeMB : 0)
  const savedPct = $derived(
    previewDone && previewItem && info && info.sizeMB > 0
      ? (1 - previewItem.sizeBytes / (1024 * 1024) / info.sizeMB) * 100
      : 0,
  )
  const savedLabel = $derived(
    Math.abs(savedPct) < 0.5
      ? '0%'
      : savedPct > 0
        ? `−${Math.round(savedPct)}%`
        : `+${Math.round(-savedPct)}%`,
  )

  onMount(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onMode = () => {
      if (systemResolved) systemDark = mq.matches
    }
    if (mq.addEventListener) mq.addEventListener('change', onMode)
    SystemTheme()
      .then((t) => {
        systemDark = t === 'dark'
        systemResolved = true
      })
      .catch(() => {
        systemDark = !!mq.matches
        systemResolved = true
      })

    load()
    Environment()
      .then((env) => (nativeFileDrop = env.platform !== 'linux'))
      .catch(() => {})
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
        i.startedAt = i.startedAt || Date.now()
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
        i.elapsedSec = i.startedAt ? (Date.now() - i.startedAt) / 1000 : 0
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
    OnFileDrop((_x: number, _y: number, paths: string[]) => {
      handleDrop(paths)
    }, true)
    return () => {
      EventsOff('compress:queued')
      EventsOff('compress:start')
      EventsOff('compress:progress')
      EventsOff('compress:done')
      EventsOff('compress:error')
      OnFileDropOff()
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
    await loadEncoders()
    await refreshQueue()
  }

  async function loadEncoders() {
    try {
      encoders = await GetEncoders()
      encodersError = ''
    } catch (err) {
      encoders = null
      encodersError = String(err)
    }
  }

  async function refreshEncoders() {
    try {
      encoders = await RefreshEncoders()
      encodersError = ''
    } catch (err) {
      encoders = null
      encodersError = String(err)
    }
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
      error = t('prefs.loadFail', { err: String(err) })
      configReady = true
      return
    }
    if (saved) {
      language = langId((saved.language ?? 'pt') as string)
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
      audioSuffix = saved.audioSuffix ?? '-audio'

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
    codec = p.codec === 'h265' ? 'h265' : 'h264'
    hardware = p.hardware ?? ''
  }

  function canRun(): boolean {
    if (!ffmpegOk || !inputPath || !outputPath || !selectedPreset || splitBlocked || adjustError) return false
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
      codec,
      hardware,
      nvencPreset,
      ...adjustPayload(),
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

  async function extractAudio(spec: audio.Spec) {
    try {
      const jobId = await ExtractAudio(spec)
      dbg(`ExtractAudio ok job=${jobId} codec=${spec.codec}`)
      createItem({
        jobId,
        kind: 'audio',
        meta: spec.codec,
        label: fileName(spec.inputPath),
        inputPath: spec.inputPath,
        presetId: '',
        outputPath: spec.outputPath,
      })
    } catch (err) {
      dbg(`ExtractAudio erro=${String(err)}`)
      error = String(err)
    }
  }

  const AUDIO_EXTS: Record<string, string> = { mp3: 'mp3', opus: 'opus', aac: 'aac', flac: 'flac', wav: 'wav', copy: 'mka' }

  function canExtractAudio(): boolean {
    return !!ffmpegOk && !!info?.hasAudio && !!audioSpec && !audioInvalid && !!audioSpec.outputPath
  }

  // Rótulo da caixa de áudio: usa o caminho sugerido pelo backend e, enquanto
  // ele não chega, um placeholder com a mesma convenção (pasta + sufixo + ext).
  function audioOutputLabel(): string {
    if (audioSpec?.outputPath) return audioSpec.outputPath
    const codec = audioSpec?.codec ?? 'mp3'
    const base = (inputPath.split(/[\\/]/).pop() || 'video').replace(/\.[^.]+$/, '')
    return targetDir() + base + (audioSuffix || '-audio') + '.' + (AUDIO_EXTS[codec] ?? 'mp3')
  }

  function extractAudioNow() {
    if (!canExtractAudio() || !audioSpec) return
    void extractAudio(audioSpec)
  }

  async function pickAudioOutput() {
    const codec = audioSpec?.codec ?? 'mp3'
    const suggested = (audioSpec?.outputPath || audioOutputLabel()).split(/[\\/]/).pop() || `audio.${AUDIO_EXTS[codec] ?? 'mp3'}`
    const path = await OpenAudioDialog(suggested, codec)
    if (path) {
      audioOutputPath = path
      audioManual = true
    }
  }

  async function clearFinished() {
    try {
      const remaining = await ClearFinished()
      queue = remaining.map(fromTask)
    } catch (err) {
      queue = queue.filter((i) => i.state === 'queued' || i.state === 'running')
      error = t('queue.clearFail', { err: String(err) })
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

<svelte:window ondragenter={onDragEnter} ondragover={onDragOver} ondragleave={onDragLeave} ondrop={handleWindowDrop} />

<div class="drop-overlay" class:show={dragging} aria-hidden="true"></div>

<header class="topbar">
  <span class="ffmpeg-pill" class:bad={!ffmpegOk}>
    <span class="dot"></span>
    {ffmpegOk ? t('topbar.ffmpegOk') : t('topbar.ffmpegMissing')}
  </span>
  <span class="topbar-title">vidctl</span>
  <div class="topbar-end">
    <button class="icon-btn" onclick={() => (prefsOpen = true)} aria-label={t('topbar.prefs')} title={t('topbar.prefs')}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <path d="M4 7h16M4 12h16M4 17h16" />
      </svg>
    </button>
  </div>
</header>

{#if !ffmpegOk}
  <div class="banner">
    <p>{ffmpegMsg || t('banner.msg')}</p>
    <button class="btn subtle small" onclick={() => load()}>{t('banner.recheck')}</button>
  </div>
{/if}

{#if appView === 'empty'}
  <StatusPage
    title={t('empty.title')}
    subtitle={t('empty.subtitle')}
    actionLabel={t('empty.open')}
    onAction={pickInput}
    hot={dragging}
  />
{:else}
  <div class="split">
    <aside
      class="sidebar"
      bind:this={sidebarEl}
      onscroll={onSidebarScroll}
      onmouseenter={() => (sbHover = true)}
      onmouseleave={() => (sbHover = false)}
    >
      {#if selectedPreset?.mode === 'size'}
        <Section title={t('sec.size')} open={sectionState.config} onchange={() => toggleSection("config")} infoTip={t('sec.sizeTip')}>
          <div class="tune-row">
            <span class="meta-k">{t('size.megabytes')}</span>
            <Stepper value={sizeMB} min={2} max={100} digits={1} ariaLabel={t('size.target')} onchange={(v) => (sizeMB = v)} />
          </div>
            {#if sizeEstimate?.available}
              <span class="est-badge mono">
                {t('size.expected', { mb: sizeEstimate.targetSizeMB.toFixed(1) })}{#if sizeEstimate.audioKbps > 0}{t('size.audioKbps', { kbps: sizeEstimate.audioKbps })}{/if}
                {#if sizeEstimate.savedPct >= 0.5}{t('size.savedPct', { pct: Math.round(sizeEstimate.savedPct) })}{/if}
              </span>
            {/if}
            <div class="hint mono">
              {#if info}
                {#if sizeEstimate?.available}
                  {t('size.hintBudget', { kbps: sizeEstimate.videoKbps })}{#if sizeEstimate.audioKbps > 0}{t('size.hintAudio', { kbps: sizeEstimate.audioKbps })}{/if}{#if splitOn}{t('size.hintPerPart')}{/if}
                {:else if advice && advice.kbps > 0}
                  {t('size.hintFallback', { kbps: advice.kbps })}{#if splitOn}{t('size.hintPerPart')}{/if}
                {:else}
                  {t('size.hintFallback', { kbps: Math.max(1, Math.round(((sizeMB * 8 * 1024 * 1024 * 0.95) / info.durationSec - 96000) / 1000)) })}
                {/if}
              {/if}
            </div>
            {#if advice && !advice.ok}
              <div class="advice">
                <p class="advice-msg">{adviceMessage(advice)}</p>
                {#if (advice.suggestion.sizeMB ?? 0) > 0 || (advice.suggestion.minutesEach ?? 0) > 0}
                  <button class="btn small" onclick={applyAdvice}>{t('advice.apply')}</button>
                {/if}
              </div>
            {/if}
          </Section>
      {:else if selectedPreset}
        <Section title={t('sec.quality')} open={sectionState.config} onchange={() => toggleSection("config")} infoTip={t('sec.qualityTip')}>
          <div class="tune-row">
            <span class="meta-k">{t('quality.crfLabel')}</span>
            <Stepper value={crf} min={16} max={34} ariaLabel="CRF" onchange={(v) => (crf = v)} />
          </div>
          </Section>
      {/if}

      <Section title={t('sec.advanced')} open={sectionState.avancado} onchange={() => toggleSection("avancado")} infoTip={t('sec.advancedTip')}>
          <div class="meta-k">{t('adv.codec')}</div>
          <div class="codec-row">
            {#each codecOptions as opt (opt.id)}
              <label class="codec-chip" class:selected={codec === opt.id}>
                <input type="radio" name="codec" value={opt.id} checked={codec === opt.id} onchange={() => (codec = opt.id)} />
                <span class="codec-chip-title">{opt.title}</span>
                <span class="codec-chip-desc">{opt.description}</span>
              </label>
            {/each}
          </div>

          <div class="meta-k spacer">{t('adv.encoder')}</div>
          <RadioCardGroup options={hardwareOptions} selected={hardware} name={t('adv.encoder')} onchange={(v) => (hardware = v)} />

          {#if gpuActive}
            <div class="advice">
              <p class="advice-msg">{t('adv.gpuActive')}</p>
            </div>
          {/if}

          {#if hardware === 'nvenc'}
            <div class="meta-k spacer">{t('adv.nvencPreset')}</div>
            <div class="preset-row" role="radiogroup" aria-label={t('adv.nvencPreset')}>
              {#each nvencPresetOptions as o (o.id)}
                <label class="preset-chip" class:selected={nvencPreset === o.id}>
                  <input type="radio" name="nvencPreset" value={o.id} checked={nvencPreset === o.id} onchange={() => (nvencPreset = o.id)} />
                  {o.title}
                </label>
              {/each}
            </div>
          {/if}

          {#if encodersError}
            <div class="hint">
              {t('adv.detectFail', { err: encodersError })}
              <button class="btn subtle small" onclick={refreshEncoders}>{t('banner.recheck')}</button>
            </div>
          {:else}
            <div class="hint">
              {encoders ? t('adv.backends', { n: encoders.hardware.length }) : t('adv.checking')}
            </div>
            {#if failingEncoders}
              <div class="enc-fail mono" title={failingEncoders}>{failingEncoders}</div>
            {/if}
          {/if}
          </Section>

      <Section title={t('sec.presets')} open={sectionState.presets} onchange={() => toggleSection("presets")} infoTip={t('sec.presetsTip')}>
          <RadioCardGroup options={presetOptions} selected={selectedPresetId} name={t('sec.presets')} onchange={selectPresetById} />
          </Section>

      <Section title={t('sec.split')} open={sectionState.split} onchange={() => toggleSection("split")} infoTip={t('sec.splitTip')}>
          <div class="split-head">
            <div class="split-toggle">
              <button class="btn small" class:solid={!splitOn} onclick={() => (splitOn = false)}>
                {t('split.off')}
              </button>
              <button class="btn small" class:solid={splitOn} onclick={() => (splitOn = true)}>
                {t('split.on')}
              </button>
            </div>
            {#if splitOn && activeSplit && !splitBlocked}
              <span class="split-tag mono">{t('split.count', { n: activeSplit.count, slice: fmtClock(activeSplit.sliceSec) })}</span>
            {/if}
          </div>

          <div class="tune-row" class:active={splitOn && splitAxis === 'parts'} class:dim={!splitOn}>
            <span class="meta-k">{t('split.parts')}</span>
            <Stepper value={splitParts} min={2} max={60} ariaLabel="partes" onchange={setParts} />
          </div>

          <div class="tune-row" class:active={splitOn && splitAxis === 'minutes'} class:dim={!splitOn}>
            <span class="meta-k">{t('split.minEach')}</span>
            <Stepper value={splitMinutes} min={1} max={60} digits={1} ariaLabel={t('split.minEach')} onchange={setMinutes} />
          </div>

          {#if splitOn && !info}
            <div class="hint">{t('split.needVideo')}</div>
          {:else if splitOn && activeSplit?.error}
            <div class="split-error mono">{activeSplit.error}</div>
          {/if}
          </Section>

      <Section title={t('sec.audio')} open={sectionState.audio} onchange={() => toggleSection("audio")} infoTip={t('sec.audioTip')}>
          {#if !info}
            <div class="hint">{t('audio.needVideo')}</div>
          {:else}
            {#key inputPath}
              <AudioPanel
                {inputPath}
                hasAudio={info.hasAudio}
                bind:spec={audioSpec}
                bind:invalid={audioInvalid}
                bind:outputPath={audioOutputPath}
                bind:manual={audioManual}
              />
            {/key}
          {/if}
          </Section>

      <Section title={t('sec.adjust')} open={sectionState.ajustes} onchange={() => toggleSection("ajustes")} infoTip={t('sec.adjustTip')}>
          <div class="meta-k">{t('adjust.scale')}</div>
            <RadioCardGroup options={scaleOptions} selected={scaleMode} name={t('adjust.scale')} onchange={(v) => (scaleMode = v as ScaleMode)} />
            {#if scaleMode === 'custom'}
              <input
                class="text-input mono"
                value={scaleCustom}
                aria-label={t('adjust.custom')}
                placeholder={t('adjust.customPlaceholder')}
                oninput={(e) => (scaleCustom = e.currentTarget.value)}
              />
            {/if}

            <div class="trim-row">
              <div class="trim-field">
                <span class="meta-k">{t('adjust.start')}</span>
                <input
                  class="text-input mono"
                  value={trimStart}
                  aria-label={t('adjust.start')}
                  placeholder={t('adjust.startPh')}
                  oninput={(e) => (trimStart = e.currentTarget.value)}
                />
              </div>
              <div class="trim-field">
                <span class="meta-k">{t('adjust.end')}</span>
                <input
                  class="text-input mono"
                  value={trimEnd}
                  aria-label={t('adjust.end')}
                  placeholder={t('adjust.endPh')}
                  oninput={(e) => (trimEnd = e.currentTarget.value)}
                />
              </div>
            </div>

            <div class="tune-row">
              <span class="meta-k">{t('adjust.fps')}</span>
              <Stepper value={fps} min={0} max={120} digits={1} ariaLabel={t('adjust.fps')} onchange={(v) => (fps = v)} />
            </div>

            <div class="meta-k spacer">{t('adjust.orientation')}</div>
            <RadioCardGroup options={rotateOptions} selected={String(rotate)} name={t('adjust.orientation')} onchange={(v) => (rotate = Number(v))} />

            <Toggle title={t('adjust.removeAudio')} description={t('adjust.removeAudioDesc')} checked={removeAudio} onchange={(v) => (removeAudio = v)} />

            <Toggle title={t('adjust.thumbnail')} description={t('adjust.thumbnailDesc')} checked={thumbnailOn} onchange={toggleThumbnail} />
            {#if thumbnailOn}
              <div class="thumb-actions">
                <button class="btn subtle small" onclick={pickThumbnail} disabled={!inputPath}>{t('adjust.pickThumb')}</button>
                {#if thumbnailPath}
                  <span class="thumb-path mono" title={thumbnailPath}>{thumbnailPath}</span>
                {/if}
              </div>
            {/if}

            {#if adjustError}
              <div class="split-error mono">{adjustError}</div>
            {/if}
          </Section>

      {#if sbVisible}
        <div
          class="sidebar-scrollbar"
          class:show={sbShow}
          style="--sb-h:{sbThumbH}px; --sb-top:{sbThumbTop}px"
          onmousedown={onSBDragStart}
          aria-hidden="true"
        >
          <div class="sidebar-scrollbar-thumb"></div>
        </div>
      {/if}
    </aside>

    <main class="content dropzone" class:hot={dragging}>
      <div class="sources-head">
        <span class="group-card-title">{t('sources.title')}</span>
        {#if info}
          <button class="btn subtle small" onclick={reset}>{t('sources.clear')}</button>
        {/if}
      </div>

      {#if info}
        <VideoRow
          {info}
          status={previewBusy ? 'progress' : previewState === 'done' ? 'done' : previewState === 'error' ? 'error' : 'idle'}
          stage={previewItem?.stage ?? ''}
          percent={previewItem?.percent ?? 0}
          {savedLabel}
          {thumb}
          onSwitch={pickInput}
          onClear={reset}
        />
        <div class="out-card group-card">
          <div class="group-card-body">
            <div class="out-row">
              <span class="out-path mono" class:empty={!outputPath}>
                {outputPath || t('out.empty')}
              </span>
              <button class="btn subtle small" onclick={pickOutput} disabled={!inputPath}>{t('out.saveAs')}</button>
            </div>
          </div>
        </div>
        <div class="out-card group-card">
          <div class="group-card-body">
            <div class="out-row">
              <button class="btn solid small" onclick={extractAudioNow} disabled={!canExtractAudio()}>{t('out.extractAudio')}</button>
              <span class="out-path mono" class:empty={!audioSpec?.outputPath} title={audioOutputLabel()}>
                {audioOutputLabel()}
              </span>
              <button class="btn subtle small" onclick={pickAudioOutput} disabled={!info?.hasAudio}>{t('out.saveAs')}</button>
            </div>
          </div>
        </div>
      {/if}

      {#if previewItem && previewBusy}
        <div class="progress-card">
          <div class="progress-head">
            <span class="mono">
              {previewItem.state === 'queued' && previewItem.position > 1
                ? t('prog.waiting', { pos: previewItem.position })
                : stageLabel(previewItem.stage) || t('prog.processing')}
            </span>
            <span class="progress-pct mono">{previewItem.percent.toFixed(1)}%</span>
          </div>
          <div class="track"><div class="fill" style="width:{previewItem.percent}%"></div></div>
          <div class="progress-foot">
            <button class="btn small" onclick={() => cancelItem(previewItem.jobId)}>{t('prog.cancel')}</button>
            <span class="progress-time mono" aria-live="polite">
              {previewItem.startedAt ? `⏱ ${fmtDuration((now - previewItem.startedAt) / 1000)}` : ''}
            </span>
            <div class="usage mono" aria-live="polite">
              {#if usage}
                CPU {Math.round(usage.cpu)}% · RAM {(usage.memUsedMB / 1024).toFixed(1)}/{(usage.memTotalMB / 1024).toFixed(1)} GB
                {#if usage.ffmpegCpu > 0.5}· ffmpeg {Math.round(usage.ffmpegCpu)}%{/if}
                {#if usage.gpu >= 0}· GPU {Math.round(usage.gpu)}%{/if}
              {:else}
                {t('prog.measuring')}
              {/if}
            </div>
          </div>
        </div>
      {:else if previewItem && previewDone}
        <div class="result">
          <div class="result-big">
            <span class="result-pct mono">{savedLabel}</span>
            <span class="result-size mono">
              {t('result.sizeMb', { mb: (previewItem.sizeBytes / (1024 * 1024)).toFixed(1) })}{#if previewItem.partsTotal > 1}{t('result.parts', { n: previewItem.partsTotal })}{/if}
            </span>
            <span class="result-from mono">{t('result.from', { mb: originalMB.toFixed(1) })}</span>
            <span class="result-time mono">{t('result.duration', { t: fmtDuration(previewItem.elapsedSec) })}</span>
          </div>
          <div class="result-actions">
            <button class="btn solid" onclick={() => OpenFolder(previewItem.outputPath)}>{t('result.openFolder')}</button>
            <button class="btn" onclick={reset}>{t('result.newVideo')}</button>
          </div>
        </div>
      {/if}

      {#if queue.length > 0}
        <div class="queue-card">
          <div class="queue-head">
            <span class="group-card-title">{t('queue.title')}</span>
            <span class="queue-count mono">{queue.length} {queue.length === 1 ? t('queue.taskOne') : t('queue.taskMany')}</span>
            {#if hasFinished}
              <button class="btn subtle small" onclick={clearFinished}>{t('queue.clear')}</button>
            {/if}
          </div>
          <ul class="queue-list">
            {#each queue as item (item.jobId)}
              <li class="queue-item" class:muted={item.state === 'canceled'}>
                <span class="qi-dot" data-state={item.state}></span>
                <div class="qi-body">
                  <div class="qi-line">
                    <span class="qi-label" title={item.inputPath}>{item.label}</span>
                    <span class="qi-preset mono">{item.kind === 'audio' ? (item.meta || t('queue.audio')).toUpperCase() : presetName(item.presetId)}</span>
                    <span class="qi-state mono">{stateLabel(item)}</span>
                  </div>
                  {#if item.state === 'running' && item.jobId !== previewItem?.jobId}
                    <div class="track thin"><div class="fill" style="width:{item.percent}%"></div></div>
                  {/if}
                  {#if item.error}
                    <div class="qi-error mono">{item.error}</div>
                  {/if}
                </div>
                {#if item.state === 'done' && item.kind === 'audio' && item.outputPath}
                  <button class="btn subtle small" onclick={() => OpenFolder(item.outputPath)}>{t('queue.openFolder')}</button>
                {:else if item.state === 'queued' || item.state === 'running'}
                  <button class="btn subtle small" onclick={() => cancelItem(item.jobId)}>{t('queue.cancel')}</button>
                {/if}
              </li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if info && !previewBusy && !previewDone}
        <div class="summary-card">
          <div class="sum-row">
            <span class="meta-k">{t('sum.config')}</span>
            <span class="sum-v">
              {#if selectedPreset}
                {selectedPreset.mode === 'size'
                  ? (splitOn ? t('sum.configSizePart', { name: presetName(selectedPreset.id), mb: sizeMB }) : t('sum.configSize', { name: presetName(selectedPreset.id), mb: sizeMB }))
                  : t('sum.configCrf', { name: presetName(selectedPreset.id), crf })}
              {:else}
                —
              {/if}
            </span>
          </div>
          <div class="sum-row">
            <span class="meta-k">{t('sum.split')}</span>
            <span class="sum-v">
              {#if splitOn && activeSplit && !splitBlocked}
                {t('sum.splitPlan', { n: activeSplit.count, slice: fmtClock(activeSplit.sliceSec) })}
              {:else if splitOn}
                {t('sum.splitAdjust')}
              {:else}
                {t('sum.splitNone')}
              {/if}
            </span>
          </div>
          {#if adjustSummary}
            <div class="sum-row">
              <span class="meta-k">{t('sum.adjust')}</span>
              <span class="sum-v">{adjustSummary}</span>
            </div>
          {/if}
          {#if advice && advice.kbps > 0}
            <div class="sum-row">
              <span class="meta-k">{t('sum.bitrate')}</span>
              <span class="sum-v">{t('sum.bitrateVal', { kbps: advice.kbps })}{#if splitOn}{t('size.hintPerPart')}{/if}{t('sum.bitrateMin', { minKbps: advice.minKbps })}</span>
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
        <button class="pill-btn" onclick={compress} disabled={!canRun()}>{t('action.compress')}</button>
        <button class="btn" onclick={pickInput} disabled={!ffmpegOk}>{t('action.switchVideo')}</button>
      </div>
    </main>
  </div>
{/if}

<footer class="foot mono">
  {t('footer.engine')}
</footer>

<PreferencesModal
  open={prefsOpen}
  {theme}
  {language}
  {ffmpegOk}
  {ffmpegMsg}
  {outputDir}
  {ffmpegPath}
  {ffprobePath}
  {audioSuffix}
  onclose={() => (prefsOpen = false)}
  ontheme={setTheme}
  onlanguage={(l) => (language = l)}
  onRecheck={checkFFmpeg}
  onoutputdir={(v) => (outputDir = v)}
  onffmpegpath={(v) => (ffmpegPath = v)}
  onffprobepath={(v) => (ffprobePath = v)}
  onaudiosuffix={(v) => (audioSuffix = v)}
/>
