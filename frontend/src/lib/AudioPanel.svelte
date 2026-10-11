<script lang="ts">
  import RadioCardGroup from './RadioCardGroup.svelte'
  import Stepper from './Stepper.svelte'
  import Toggle from './Toggle.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import { SuggestAudioOutput } from '../../wailsjs/go/main/App.js'
  import type { audio } from '../../wailsjs/go/models.js'

  interface Props {
    inputPath: string
    hasAudio: boolean
    spec?: audio.Spec
    invalid?: string
  }

  let { inputPath, hasAudio, spec = $bindable(), invalid = $bindable('') }: Props = $props()

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

  const lossless = $derived(LOSSLESS[codec] ?? false)

  const validation = $derived.by<string>(() => {
    if (Number.isNaN(trimStartSec) || Number.isNaN(trimEndSec)) {
      return 'tempo de extração inválido — use mm:ss (ex.: 1:30)'
    }
    if (trimStartSec > 0 && trimEndSec > 0 && trimEndSec <= trimStartSec) {
      return 'o fim da janela deve ser maior que o início'
    }
    return ''
  })

  let suggestedOutput = $state('')

  // O destino segue a pasta da configuração e o sufixo das Preferências; o
  // painel não repete o "salvar como…" já presente no cartão principal.
  $effect(() => {
    const path = inputPath
    const fmt = codec
    if (!path) return
    let cancelled = false
    SuggestAudioOutput(path, fmt)
      .then((p) => {
        if (!cancelled && codec === fmt) suggestedOutput = p
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  })

  const specPlain = $derived.by<audio.Spec>(() => ({
    inputPath,
    codec,
    bitrateKbps: lossless ? 0 : bitrateKbps,
    sampleRate: 0,
    channels,
    trimStartSec: trimStartSec > 0 ? trimStartSec : 0,
    trimEndSec: trimEndSec > 0 ? trimEndSec : 0,
    normalize: normalize && !lossless,
    outputPath: suggestedOutput ?? '',
  }))

  $effect(() => {
    spec = specPlain
  })

  $effect(() => {
    invalid = validation
  })

  $effect(() => {
    if (lossless) bitrateKbps = DEFAULT_BITRATE[codec] ?? 0
    if (codec === 'copy') normalize = false
  })

  function selectCodec(id: string) {
    codec = id
    if (!LOSSLESS[id]) bitrateKbps = DEFAULT_BITRATE[id] ?? bitrateKbps
  }
</script>

<div class="meta-k spacer">formato</div>
<RadioCardGroup options={codecOptions} selected={codec} name="formato" onchange={selectCodec} />

{#if !lossless}
  <div class="tune-row">
    <span class="meta-k">bitrate (kbps)</span>
    <Stepper value={bitrateKbps} min={32} max={320} step={16} ariaLabel="bitrate" onchange={(v) => (bitrateKbps = v)} />
  </div>
{/if}

<div class="meta-k spacer">canais</div>
<RadioCardGroup options={channelOptions} selected={String(channels)} name="canais" onchange={(v) => (channels = Number(v))} />

<div class="spacer">
  <Toggle
    title="normalizar (EBU R128)"
    description="loudnorm single-pass para volume consistente"
    disabled={lossless}
    checked={normalize}
    onchange={(v) => (normalize = v)}
  />
  <InfoTip text="Normaliza o volume segundo o padrão EBU R128. Não se aplica a Copiar." />
</div>

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

{#if !hasAudio}
  <div class="hint">este vídeo não tem faixa de áudio</div>
{/if}

<style>
  .spacer {
    margin-top: 14px;
  }
</style>