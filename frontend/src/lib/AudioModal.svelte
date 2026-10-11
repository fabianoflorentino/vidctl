<script lang="ts">
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import Stepper from './Stepper.svelte'
  import Toggle from './Toggle.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import { ExtractAudio, OpenAudioDialog, SuggestAudioOutput } from '../../wailsjs/go/main/App.js'
  import type { audio } from '../../wailsjs/go/models.js'

  interface Props {
    open: boolean
    inputPath: string
    onclose?: () => void
    onExtract?: (spec: audio.Spec) => void
  }

  let { open, inputPath, onclose = () => {}, onExtract }: Props = $props()

  const FORMATS = [
    { id: 'mp3', title: 'MP3', description: 'padrão da UI · 192 kbps' },
    { id: 'opus', title: 'Opus', description: 'melhor voz · 128 kbps' },
    { id: 'aac', title: 'AAC', description: 'sem encoder extra · 160 kbps' },
    { id: 'flac', title: 'FLAC', description: 'lossless, sem bitrate' },
    { id: 'wav', title: 'WAV', description: 'master / edição' },
    { id: 'copy', title: 'Copiar', description: 'só remux, sem recodificar' },
  ]

  const LOSSLESS: Record<string, boolean> = { flac: true, wav: true, copy: true }

  const DEFAULT_BITRATE: Record<string, number> = { mp3: 192, opus: 128, aac: 160 }

  let codec = $state('mp3')
  let bitrateKbps = $state(192)
  let channels = $state(0)
  let normalize = $state(false)
  let trimStart = $state('')
  let trimEnd = $state('')
  let outputPath = $state('')
  let modalError = $state('')

  const codecOptions: RadioOption[] = FORMATS.map((f) => ({ id: f.id, title: f.title, description: f.description }))

  const channelOptions: RadioOption[] = [
    { id: '0', title: 'Manter', description: 'usa os canais do vídeo' },
    { id: '1', title: 'Mono', description: '1 canal' },
    { id: '2', title: 'Estéreo', description: '2 canais' },
  ]

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

  const lossless = $derived(LOSSLESS[codec] ?? false)

  const validation = $derived.by<string>(() => {
    if (Number.isNaN(trimStartSec) || Number.isNaN(trimEndSec)) {
      return 'tempo de extração inválido — use mm:ss (ex.: 1:30)'
    }
    if (trimStartSec > 0 && trimEndSec > 0 && trimEndSec <= trimStartSec) {
      return 'o fim da janela deve ser maior que o início'
    }
    if (!outputPath) return 'escolha o destino do áudio'
    return ''
  })

  const spec = $derived<audio.Spec>({
    inputPath,
    codec,
    bitrateKbps: lossless ? 0 : bitrateKbps,
    sampleRate: 0,
    channels,
    trimStartSec: trimStartSec > 0 ? trimStartSec : 0,
    trimEndSec: trimEndSec > 0 ? trimEndSec : 0,
    normalize: normalize && !lossless,
    outputPath,
  })

  $effect(() => {
    if (lossless) bitrateKbps = DEFAULT_BITRATE[codec] ?? 0
    if (codec === 'copy') normalize = false
  })

  let manual = $state(false)

  $effect(() => {
    if (open) manual = false
  })

  $effect(() => {
    const path = inputPath
    const fmt = codec
    if (!open || !path || manual) return
    let cancelled = false
    SuggestAudioOutput(path, fmt)
      .then((p) => {
        if (!cancelled && codec === fmt) outputPath = p
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  })

  async function pickOutput() {
    const name = outputPath.split(/[\\/]/).pop() || `audio-${codec}.${ext(codec)}`
    const path = await OpenAudioDialog(name, codec)
    if (path) {
      outputPath = path
      manual = true
    }
  }

  function ext(codec: string): string {
    const map: Record<string, string> = { mp3: 'mp3', opus: 'opus', aac: 'aac', flac: 'flac', wav: 'wav', copy: 'mka' }
    return map[codec] ?? 'mp3'
  }

  function selectCodec(id: string) {
    codec = id
    if (!LOSSLESS[id]) bitrateKbps = DEFAULT_BITRATE[id] ?? bitrateKbps
  }

  async function run() {
    if (validation || !inputPath) {
      modalError = validation || 'escolha um vídeo primeiro'
      return
    }
    modalError = ''
    onExtract?.(spec)
  }
</script>

<Modal {open} title="Extrair áudio" {onclose}>
  <div class="modal-section">
    <h2>Formato</h2>
    <RadioCardGroup options={codecOptions} selected={codec} name="formato" onchange={selectCodec} />
  </div>

  <div class="modal-section" class:hidden={lossless}>
    <h2>
      Qualidade
      <InfoTip text="Taxa de bits dos formatos com perda. FLAC, WAV e Copiar não aceitam bitrate: saem sem perda." />
    </h2>
    <div class="tune-row">
      <span class="meta-k">bitrate (kbps)</span>
      <Stepper value={bitrateKbps} min={32} max={320} step={16} ariaLabel="bitrate" onchange={(v) => (bitrateKbps = v)} />
    </div>
  </div>

  <div class="modal-section">
    <h2>Canais</h2>
    <RadioCardGroup options={channelOptions} selected={String(channels)} name="canais" onchange={(v) => (channels = Number(v))} />
  </div>

  <div class="modal-section">
    <h2>Normalizar</h2>
    <Toggle
      title="loudness EBU R128"
      description="af de loudnorm (single-pass) para volume consistente"
      disabled={lossless}
      checked={normalize}
      onchange={(v) => (normalize = v)}
    />
  </div>

  <div class="modal-section">
    <h2>
      Janela
      <InfoTip text="Extrai só o trecho entre início e fim (mm:ss). Deixe tudo vazio para o vídeo inteiro." />
    </h2>
    <div class="trim-row">
      <div class="trim-field">
        <span class="meta-k">início (mm:ss)</span>
        <input class="text-input mono" value={trimStart} aria-label="início da extração" placeholder="0:00" oninput={(e) => (trimStart = e.currentTarget.value)} />
      </div>
      <div class="trim-field">
        <span class="meta-k">fim (mm:ss)</span>
        <input class="text-input mono" value={trimEnd} aria-label="fim da extração" placeholder="fim do vídeo" oninput={(e) => (trimEnd = e.currentTarget.value)} />
      </div>
    </div>
  </div>

  <div class="modal-section">
    <h2>Destino</h2>
    <div class="out-row">
      <span class="out-path mono" class:empty={!outputPath} title={outputPath}>
        {outputPath || 'escolha o destino do áudio'}
      </span>
      <button class="btn subtle small" onclick={pickOutput} disabled={!inputPath}>salvar como…</button>
    </div>
  </div>

  {#if modalError || validation}
    <div class="split-error mono">{modalError || validation}</div>
  {/if}

  <div class="modal-actions">
    <button class="btn solid" onclick={run} disabled={!inputPath || !!validation}>EXTRAIR ÁUDIO</button>
  </div>
</Modal>

<style>
  .hidden {
    display: none;
  }
  .modal-actions {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }
</style>