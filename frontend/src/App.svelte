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

  type ProgressEv = { jobId: string; stage: string; percent: number }
  type DoneEv = { jobId: string; outputPath: string; sizeMB: number; sizeBytes: number }
  type ErrorEv = { jobId: string; error: string }

  let presetList = $state<presets.Preset[]>([])
  let ffmpegOk = $state(true)
  let ffmpegMsg = $state('')

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

  function fmtDuration(sec: number): string {
    if (!sec || sec < 0) return '—'
    const m = Math.floor(sec / 60)
    const s = Math.floor(sec % 60)
    return m > 0 ? `${m}m${String(s).padStart(2, '0')}s` : `${s}s`
  }

  function fmtMB(n: number): string {
    if (!n || n <= 0) return '—'
    return n.toFixed(1) + ' MB'
  }

  function stageLabel(s: string): string {
    switch (s) {
      case 'pass1/2':
        return 'ANALISANDO (1/2)'
      case 'pass2/2':
        return 'COMPRIMINDO (2/2)'
      case 'encoding':
        return 'ENCODANDO'
      case 'done':
        return 'CONCLUÍDO'
      case 'queue':
        return 'ENFILEIRADO'
      default:
        return s.toUpperCase()
    }
  }
</script>

<div class="film"></div>

<main class="stage">
  <header class="masthead">
    <div class="brand">
      <span class="brand-title">VIDCTL</span>
      <span class="brand-sub">compressor de vídeo · h264 + aac</span>
    </div>
    <div class="ffmpeg" class:bad={!ffmpegOk}>
      <span class="dot"></span>
      {ffmpegOk ? 'ffmpeg OK' : 'ffmpeg ausente'}
    </div>
  </header>

  {#if !ffmpegOk}
    <div class="warnbar">
      <span class="warn-code">ERR</span>
      <p>{ffmpegMsg || 'ffmpeg não encontrado. Instale o ffmpeg e reinicie o app.'}</p>
      <button class="btn ghost" onclick={() => load()}>verificar de novo</button>
    </div>
  {/if}

  <!-- 01 · entrada -->
  <section class="step">
    <div class="step-head">
      <span class="step-num">01</span>
      <span class="step-label">entrada</span>
    </div>

    {#if !info}
      <button class="picker" onclick={pickInput}>
        <span class="picker-plus">+</span>
        <span class="picker-main">ESCOLHER VÍDEO</span>
        <span class="picker-sub">mp4 · mkv · mov · avi · webm</span>
      </button>
    {:else}
      <div class="card">
        <div class="card-head">
          <div class="file-name" title={info.path}>{info.path.split('/').pop()}</div>
          <span class="chip">{info.width}×{info.height}</span>
          <span class="chip vertical">{info.width > info.height ? '16:9' : '9:16'}</span>
        </div>
        <div class="meta-grid">
          <div class="meta"><span class="meta-k">duração</span><span class="meta-v">{fmtDuration(info.durationSec)}</span></div>
          <div class="meta"><span class="meta-k">tamanho</span><span class="meta-v">{fmtMB(info.sizeMB)}</span></div>
          <div class="meta"><span class="meta-k">vídeo</span><span class="meta-v">{info.videoCodec || '—'}</span></div>
          <div class="meta"><span class="meta-k">áudio</span><span class="meta-v">{info.audioCodec || 'sem áudio'}</span></div>
        </div>
        <div class="card-foot">
          <button class="btn ghost small" onclick={pickInput}>trocar arquivo</button>
        </div>
      </div>
    {/if}
  </section>

  <!-- 02 · preset -->
  <section class="step">
    <div class="step-head">
      <span class="step-num">02</span>
      <span class="step-label">destino</span>
    </div>

    <div class="presets">
      {#each presetList as p (p.id)}
        <button
          class="preset"
          class:picked={p.id === selectedPresetId}
          onclick={() => selectPreset(p)}
        >
          <span class="preset-name">{p.name}</span>
          <span class="preset-desc">{p.description}</span>
          <span class="preset-target">
            {p.mode === 'size' ? `${Math.round(p.sizeMB)} MB` : `CRF ${p.crf}`}
          </span>
        </button>
      {/each}
    </div>

    {#if selectedPreset}
      <div class="tune">
        {#if selectedPreset.mode === 'size'}
          <label class="tune-row">
            <span class="meta-k">tamanho alvo</span>
            <div class="tune-control">
              <input
                type="range"
                min="2"
                max="100"
                step="1"
                value={sizeMB}
                oninput={(e) => (sizeMB = Number((e.currentTarget as HTMLInputElement).value))}
              />
              <input
                class="num"
                type="number"
                min="2"
                max="100"
                bind:value={sizeMB}
              />
              <span class="unit">MB</span>
            </div>
          </label>
        {:else}
          <div class="tune-row">
            <span class="meta-k">qualidade (CRF)</span>
            <span class="meta-v">quanto menor, melhor · {crf}</span>
          </div>
        {/if}
        <div class="hint mono">
          {#if info && selectedPreset.mode === 'size'}
            ≈ bitrate alvo: video
            {Math.max(1, Math.round(((sizeMB * 8 * 1024 * 1024 * 0.95) / info.durationSec - 96000) / 1000))}
            kbps · áudio 96k
          {:else if selectedPreset.mode === 'crf'}
            compressão de qualidade, sem limite de tamanho
          {/if}
        </div>
      </div>
    {/if}
  </section>

  <!-- 03 · saída -->
  <section class="step">
    <div class="step-head">
      <span class="step-num">03</span>
      <span class="step-label">saída</span>
    </div>

    <div class="card">
      <div class="output-row">
        <span class="mono outpath" class:empty={!outputPath}>
          {outputPath || 'escolha o vídeo para gerar o caminho de saída'}
        </span>
        <button class="btn ghost small" onclick={pickOutput} disabled={!inputPath}>salvar como…</button>
      </div>
    </div>
  </section>

  <!-- 04 · executar -->
  <section class="step run">
    <div class="step-head">
      <span class="step-num">04</span>
      <span class="step-label">executar</span>
    </div>

    <div class="exec">
      {#if running}
        <div class="progress">
          <div class="progress-head mono">
            <span>{stageLabel(stage)}</span>
            <span>{percent.toFixed(1)}%</span>
          </div>
          <div class="track">
            <div class="fill" style="width:{percent}%"></div>
          </div>
          <div class="progress-actions">
            <button class="btn ghost small" onclick={cancel}>cancelar</button>
          </div>
        </div>
      {:else if done}
        <div class="result">
          <div class="result-big">
            <span class="result-pct mono">−{savedPct.toFixed(0)}%</span>
            <span class="result-size mono">{fmtMB(done.sizeMB)}</span>
            <span class="result-from mono">de {fmtMB(originalMB)}</span>
          </div>
          <div class="result-actions">
            <button class="btn solid" onclick={() => OpenFolder(done!.outputPath)}>abrir pasta</button>
            <button class="btn ghost" onclick={reset}>novo vídeo</button>
          </div>
        </div>
      {:else}
        <button class="cta" class:off={!canRun()} onclick={compress} disabled={!canRun()}>
          <span class="cta-shadow">COMPRIMIR</span>
          <span class="cta-face">COMPRIMIR{selectedPreset ? ' → ' + selectedPreset.name.toUpperCase() : ''}</span>
        </button>
      {/if}
    </div>

    {#if error}
      <div class="alert">
        <span class="warn-code">✕</span>
        <p class="mono">{error}</p>
      </div>
    {/if}
  </section>

  <footer class="foot mono">
    ffmpeg calcula o bitrate pela duração para caber no alvo · 2-pass quando há limite de tamanho
  </footer>
</main>

<style>
  .stage {
    max-width: 940px;
    margin: 0 auto;
    padding: 22px 32px 16px;
    position: relative;
  }

  /* masthead */
  .masthead {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 12px;
    animation: rise 0.5s ease both;
  }
  .brand-title {
    display: block;
    font-size: 30px;
    font-weight: 800;
    letter-spacing: 0.02em;
    line-height: 0.95;
    text-transform: uppercase;
  }
  .brand-title::after {
    content: "⌗";
    color: var(--acid);
    margin-left: 6px;
  }
  .brand-sub {
    display: block;
    margin-top: 6px;
    color: var(--ink-dim);
    font-family: var(--mono);
    font-size: 11px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .ffmpeg {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    padding: 5px 10px;
    border: 1px solid var(--line);
    border-radius: 999px;
    font-family: var(--mono);
    font-size: 10px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--ink-dim);
    background: var(--panel);
  }
  .ffmpeg .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--acid);
    box-shadow: 0 0 10px var(--acid);
  }
  .ffmpeg.bad .dot {
    background: var(--danger);
    box-shadow: 0 0 10px var(--danger);
  }
  .ffmpeg.bad {
    color: var(--danger);
  }

  /* warnbar */
  .warnbar {
    display: flex;
    align-items: center;
    gap: 14px;
    border: 1px solid rgba(255, 93, 77, 0.4);
    background: rgba(255, 93, 77, 0.08);
    border-radius: var(--radius);
    padding: 14px 18px;
    margin-bottom: 32px;
  }
  .warnbar p {
    margin: 0;
    color: var(--ink);
    font-family: var(--mono);
    font-size: 12px;
    flex: 1;
  }
  .warn-code {
    font-family: var(--mono);
    font-weight: 700;
    color: var(--danger);
    font-size: 12px;
  }

  /* steps */
  .step {
    position: relative;
    margin-bottom: 10px;
    animation: rise 0.55s ease both;
  }
  .step:nth-of-type(2) {
    animation-delay: 0.05s;
  }
  .step:nth-of-type(3) {
    animation-delay: 0.1s;
  }
  .step:nth-of-type(4) {
    animation-delay: 0.15s;
  }
  .step-head {
    display: flex;
    align-items: baseline;
    gap: 12px;
    margin-bottom: 4px;
  }
  .step-num {
    font-family: var(--mono);
    font-size: 12px;
    font-weight: 700;
    color: var(--acid);
  }
  .step-label {
    font-size: 16px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
  }
  .step-head::after {
    content: "";
    flex: 1;
    height: 1px;
    background: var(--line);
    transform: translateY(-2px);
  }

  /* picker */
  .picker {
    width: 100%;
    border: 2px dashed var(--line);
    border-radius: var(--radius);
    background: transparent;
    color: var(--ink);
    padding: 18px 20px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    transition: border-color 0.2s ease, background 0.2s ease;
  }
  .picker:hover {
    border-color: var(--acid);
    background: var(--acid-dim);
  }
  .picker-plus {
    font-size: 24px;
    line-height: 1;
    color: var(--acid);
  }
  .picker-main {
    font-size: 15px;
    font-weight: 700;
    letter-spacing: 0.06em;
  }
  .picker-sub {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--ink-faint);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }

  /* cards */
  .card {
    border: 1px solid var(--line);
    background: var(--panel);
    border-radius: var(--radius);
    padding: 12px 18px;
  }
  .card-head {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }
  .file-name {
    font-size: 15px;
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  }
  .chip {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--ink-dim);
    border: 1px solid var(--line);
    border-radius: 6px;
    padding: 2px 7px;
  }
  .chip.vertical {
    color: var(--amber);
    border-color: rgba(255, 176, 0, 0.35);
  }
  .meta-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
    margin: 9px 0;
  }
  .meta {
    border-left: 2px solid var(--line);
    padding-left: 10px;
  }
  .meta-k {
    display: block;
    font-family: var(--mono);
    font-size: 9px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--ink-faint);
    margin-bottom: 2px;
  }
  .meta-v {
    font-family: var(--mono);
    font-size: 12px;
    color: var(--ink);
  }
  .card-foot {
    display: flex;
    justify-content: flex-start;
  }

  /* presets */
  .presets {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: minmax(125px, 1fr);
    grid-template-rows: 1fr;
    gap: 8px;
    overflow-x: auto;
  }
  .preset {
    text-align: left;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: var(--panel);
    color: var(--ink);
    padding: 8px 13px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 3px;
    position: relative;
    transition: transform 0.15s ease, border-color 0.15s ease, background 0.15s ease;
  }
  .preset:hover {
    border-color: var(--line-strong);
    background: var(--panel-2);
    box-shadow: 0 0 0 1px rgba(215, 243, 77, 0.15);
  }
  .preset.picked {
    border-color: var(--acid);
    background: var(--panel-2);
    box-shadow: inset 3px 0 0 var(--acid);
  }
  .preset-name {
    font-size: 13px;
    font-weight: 700;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .preset-desc {
    font-size: 10.5px;
    color: var(--ink-dim);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .preset-target {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--acid);
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  /* tune */
  .tune {
    margin-top: 12px;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    padding: 12px 18px;
    background: var(--panel);
  }
  .tune-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
  .tune-control {
    display: flex;
    align-items: center;
    gap: 14px;
    flex: 1;
    max-width: 480px;
  }
  input[type='range'] {
    flex: 1;
    appearance: none;
    height: 3px;
    background: var(--line-strong);
    border-radius: 2px;
  }
  input[type='range']::-webkit-slider-thumb {
    appearance: none;
    width: 15px;
    height: 15px;
    border-radius: 50%;
    background: var(--acid);
    cursor: pointer;
    box-shadow: 0 0 0 4px rgba(215, 243, 77, 0.15);
  }
  input[type='number'].num {
    width: 64px;
    background: var(--bg-hi);
    border: 1px solid var(--line);
    color: var(--ink);
    border-radius: 6px;
    padding: 7px 10px;
    font-family: var(--mono);
    font-size: 14px;
    text-align: center;
  }
  .unit {
    font-family: var(--mono);
    font-size: 12px;
    color: var(--ink-dim);
  }
  .hint {
    margin-top: 14px;
    font-size: 11px;
    color: var(--ink-faint);
  }

  /* output */
  .output-row {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .outpath {
    flex: 1;
    font-size: 12px;
    word-break: break-all;
    color: var(--ink);
  }
  .outpath.empty {
    color: var(--ink-faint);
  }

  /* buttons */
  .btn {
    border: 1px solid var(--line);
    background: transparent;
    color: var(--ink);
    border-radius: 8px;
    padding: 8px 14px;
    font-size: 12px;
    font-weight: 600;
    transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
  }
  .btn:hover:not(:disabled) {
    border-color: var(--acid);
    color: var(--acid);
  }
  .btn:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }
  .btn.ghost.small {
    padding: 6px 10px;
    font-size: 11px;
  }
  .btn.solid {
    background: var(--acid);
    border-color: var(--acid);
    color: #0d0c0a;
    font-weight: 700;
  }
  .btn.solid:hover {
    background: #e6ff67;
    color: #0d0c0a;
  }

  /* exec: fixed height so states never change the page height */
  .exec {
    min-height: 96px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  .exec .progress,
  .exec .result {
    margin: 0;
  }

  /* alert */
  .alert {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    border: 1px solid rgba(255, 93, 77, 0.4);
    background: rgba(255, 93, 77, 0.07);
    border-radius: var(--radius);
    padding: 12px 16px;
    margin-bottom: 16px;
  }
  .alert p {
    margin: 0;
    font-size: 12px;
  }

  /* progress */
  .progress {
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: var(--panel);
    padding: 12px 18px;
    margin-bottom: 12px;
  }
  .progress-head {
    display: flex;
    justify-content: space-between;
    font-size: 10px;
    color: var(--ink-dim);
    margin-bottom: 8px;
    letter-spacing: 0.06em;
  }
  .track {
    height: 9px;
    border: 1px solid var(--line);
    border-radius: 3px;
    overflow: hidden;
    background:
      repeating-linear-gradient(
        90deg,
        var(--bg-hi) 0 12px,
        var(--panel-2) 12px 16px
      );
  }
  .fill {
    height: 100%;
    background: repeating-linear-gradient(
      90deg,
      var(--acid) 0 12px,
      #c4dd3c 12px 16px
    );
    transition: width 0.25s ease;
  }
  .progress-actions {
    margin-top: 10px;
    text-align: right;
  }

  /* result */
  .result {
    border: 1px solid var(--acid);
    border-radius: var(--radius);
    background: var(--acid-dim);
    padding: 14px 18px;
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
    animation: rise 0.3s ease both;
  }
  .result-big {
    display: flex;
    align-items: baseline;
    gap: 12px;
    flex-wrap: wrap;
  }
  .result-pct {
    font-size: 28px;
    font-weight: 700;
    color: var(--acid);
  }
  .result-size {
    font-size: 18px;
    color: var(--ink);
  }
  .result-from {
    font-size: 12px;
    color: var(--ink-dim);
  }
  .result-actions {
    display: flex;
    gap: 10px;
  }

  /* cta */
  .cta {
    width: 100%;
    position: relative;
    border: none;
    background: none;
    padding: 0;
    font-family: var(--disp);
  }
  .cta-shadow {
    position: absolute;
    inset: 0;
    transform: translate(6px, 6px);
    border: 1px solid var(--acid);
    border-radius: 10px;
  }
  .cta-face {
    display: block;
    position: relative;
    background: var(--acid);
    color: #0d0c0a;
    border-radius: 10px;
    padding: 13px;
    font-size: 17px;
    font-weight: 800;
    letter-spacing: 0.06em;
    text-align: center;
    transition: transform 0.12s ease;
  }
  .cta:not(:disabled):hover .cta-face {
    transform: translate(-3px, -3px);
  }
  .cta.off .cta-face {
    background: var(--panel-2);
    color: var(--ink-faint);
  }
  .cta.off .cta-shadow {
    border-color: var(--line);
  }
  .cta:disabled {
    cursor: not-allowed;
  }

  .foot {
    margin-top: 20px;
    font-size: 11px;
    color: var(--ink-faint);
    letter-spacing: 0.04em;
    text-align: center;
  }

  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (max-width: 640px) {
    .meta-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .stage {
      padding: 56px 20px 40px;
    }
  }
</style>