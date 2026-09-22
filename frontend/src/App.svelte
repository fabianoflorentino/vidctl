<script lang="ts">
  import { onMount } from 'svelte'
  import {
    CheckFFmpeg,
    GetPresets,
    OpenInputDialog,
    GetMediaInfo,
    OpenOutputDialog,
    Compress as StartCompress,
    Cancel as CancelJob,
    OpenFolder,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js'
  import type { main, presets, media } from '../wailsjs/go/models.js'
  import StatusPage from './lib/StatusPage.svelte'
  import ControlGroup from './lib/ControlGroup.svelte'
  import VideoRow from './lib/VideoRow.svelte'

  type ProgressEv = { jobId: string; stage: string; percent: number }
  type DoneEv = { jobId: string; outputPath: string; sizeMB: number; sizeBytes: number }
  type ErrorEv = { jobId: string; error: string }

  let presetList = $state<presets.Preset[]>([])
  let ffmpegOk = $state(true)
  let ffmpegMsg = $state('')

  let theme = $state<'light' | 'dark' | ''>('')

  try {
    const saved = localStorage.getItem('vidctl-theme')
    if (saved === 'light' || saved === 'dark') {
      theme = saved
      document.documentElement.dataset.theme = saved
    }
  } catch {
    /* localStorage indisponível: segue o tema do sistema */
  }

  function currentTheme(): 'light' | 'dark' {
    if (theme) return theme
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
  }

  function toggleTheme() {
    theme = currentTheme() === 'light' ? 'dark' : 'light'
    document.documentElement.dataset.theme = theme
    try {
      localStorage.setItem('vidctl-theme', theme)
    } catch {
      /* sem persistência disponível; tema vale para a sessão */
    }
  }

  let inputPath = $state('')
  let info = $state<media.Info | null>(null)
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

  const selectedPreset = $derived(presetList.find((p) => p.id === selectedPresetId) ?? null)

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
      running = false
      percent = 100
      stage = 'done'
      done = e
      currentJobId = ''
    })
    EventsOn('compress:error', (e: ErrorEv) => {
      if (e.jobId !== currentJobId) return
      running = false
      error = e.error
      currentJobId = ''
    })
    return () => {
      EventsOff('compress:progress')
      EventsOff('compress:done')
      EventsOff('compress:error')
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
    inputPath = path
    outputPath = ''
    done = null
    error = ''
    try {
      info = await GetMediaInfo(path)
      const base = path.replace(/\.[^.]+$/, '')
      outputPath = base + '-compressed.mp4'
    } catch (err) {
      error = String(err)
    }
  }

  async function pickOutput() {
    const suggested = outputPath.split('/').pop() || 'comprimido.mp4'
    const path = await OpenOutputDialog(suggested)
    if (!path) return
    outputPath = path
  }

  function selectPreset(p: presets.Preset) {
    selectedPresetId = p.id
    if (p.mode === 'size') sizeMB = p.sizeMB
    else crf = p.crf
  }

  function canRun(): boolean {
    return !running && !!inputPath && !!outputPath && !!selectedPreset && ffmpegOk
  }

  async function compress() {
    if (!canRun()) return
    error = ''
    done = null
    percent = 0
    stage = 'queue'
    running = true
    try {
      const jobId = await StartCompress({
        inputPath,
        outputPath,
        presetId: selectedPresetId,
        sizeMB: sizeMB > 0 ? sizeMB : 10,
        crf,
      })
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
    outputPath = ''
    done = null
    error = ''
    percent = 0
    stage = ''
  }
</script>

<header class="topbar">
  <div class="brand">
    <span class="brand-title">VIDCTL</span>
    <span class="brand-sub">compressor de vídeo</span>
  </div>
  <div class="topbar-end">
    <button
      class="icon-btn"
      onclick={toggleTheme}
      aria-label="Alternar tema claro/escuro"
      title={`Tema: ${currentTheme() === 'light' ? 'claro' : 'escuro'} — clicar para alternar`}
    >
      {#if currentTheme() === 'light'}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8z" />
        </svg>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
        </svg>
      {/if}
    </button>
    <span class="ffmpeg-pill" class:bad={!ffmpegOk}>
      <span class="dot"></span>
      {ffmpegOk ? 'ffmpeg OK' : 'ffmpeg ausente'}
    </span>
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
    subtitle="Escolha um vídeo para começar"
    actionLabel="Escolher vídeo…"
    onAction={pickInput}
  />
{:else}
  <div class="split">
    <aside class="sidebar">
      <ControlGroup title="Destino" help="Escolha um preset de saída. Presets de tamanho fazem o ffmpeg calcular o bitrate pela duração para caber no alvo (2-pass).">
        {#each presetList as p (p.id)}
          <button class="row" class:selected={p.id === selectedPresetId} onclick={() => selectPreset(p)}>
            <span class="row-name">
              {p.name}
              <span class="row-desc">{p.description}</span>
            </span>
            <span class="row-val">{p.mode === 'size' ? `${Math.round(p.sizeMB)} MB` : `CRF ${p.crf}`}</span>
          </button>
        {/each}
      </ControlGroup>

      {#if selectedPreset}
        <ControlGroup title={selectedPreset.mode === 'size' ? 'Tamanho alvo' : 'Qualidade'}>
          {#if selectedPreset.mode === 'size'}
            <label class="tune-row">
              <span class="meta-k">tamanho</span>
              <div class="tune-control">
                <input
                  type="range"
                  min="2"
                  max="100"
                  step="1"
                  value={sizeMB}
                  oninput={(e) => (sizeMB = Number((e.currentTarget as HTMLInputElement).value))}
                />
                <input class="num" type="number" min="2" max="100" bind:value={sizeMB} />
                <span class="unit">MB</span>
              </div>
            </label>
            <div class="hint mono">
              {#if info}
                ≈ video
                {Math.max(1, Math.round(((sizeMB * 8 * 1024 * 1024 * 0.95) / info.durationSec - 96000) / 1000))}
                kbps · áudio 96k
              {/if}
            </div>
          {:else}
            <div class="tune-row">
              <span class="meta-k">CRF — quanto menor, melhor</span>
              <span class="row-val mono">{crf}</span>
            </div>
          {/if}
        </ControlGroup>
      {/if}

      <ControlGroup title="Saída">
        <div class="out-row">
          <span class="out-path mono" class:empty={!outputPath}>
            {outputPath || 'escolha o vídeo para gerar o caminho de saída'}
          </span>
          <button class="btn subtle small" onclick={pickOutput} disabled={!inputPath}>salvar como…</button>
        </div>
      </ControlGroup>
    </aside>

    <main class="content">
      {#if info}
        <VideoRow
          info={info}
          status={running ? 'progress' : done ? 'done' : error ? 'error' : 'idle'}
          stage={stage}
          percent={percent}
          savedPct={savedPct}
          onSwitch={pickInput}
        />
      {/if}

      {#if error}
        <div class="alert">
          <p class="mono">{error}</p>
        </div>
      {/if}

      <div class="actionbar">
        {#if running}
          <div class="progress-block">
            <div class="progress-head mono">
              <span>{stage || 'PROCESSANDO'}</span>
              <span>{percent.toFixed(1)}%</span>
            </div>
            <div class="track"><div class="fill" style="width:{percent}%"></div></div>
            <div class="progress-actions"><button class="btn small" onclick={cancel}>cancelar</button></div>
          </div>
        {:else if done}
          <div class="result">
            <div class="result-big">
              <span class="result-pct mono">−{savedPct.toFixed(0)}%</span>
              <span class="result-size mono">{done.sizeMB.toFixed(1)} MB</span>
              <span class="result-from mono">de {originalMB.toFixed(1)} MB</span>
            </div>
            <div class="result-actions">
              <button class="btn solid" onclick={() => OpenFolder(done!.outputPath)}>abrir pasta</button>
              <button class="btn" onclick={reset}>novo vídeo</button>
            </div>
          </div>
        {:else}
          <span class="spacer"></span>
          <button class="btn solid" onclick={compress} disabled={!canRun()}>
            COMPRIMIR{selectedPreset ? ' → ' + selectedPreset.name.toUpperCase() : ''}
          </button>
        {/if}
      </div>
    </main>
  </div>
{/if}

<footer class="foot mono">
  offline · h264 + aac · 2-pass quando há limite de tamanho
</footer>