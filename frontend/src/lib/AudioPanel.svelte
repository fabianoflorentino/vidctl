<script lang="ts">
  import RadioCardGroup from './RadioCardGroup.svelte'
  import Stepper from './Stepper.svelte'
  import Toggle from './Toggle.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import { SuggestAudioOutput } from '../../wailsjs/go/main/App.js'
  import type { audio } from '../../wailsjs/go/models.js'
  import { t } from './i18n.svelte'

  interface Props {
    inputPath: string
    hasAudio: boolean
    spec?: audio.Spec
    invalid?: string
    outputPath?: string
    manual?: boolean
  }

  let {
    inputPath,
    hasAudio,
    spec = $bindable(),
    invalid = $bindable(''),
    outputPath = $bindable(''),
    manual = $bindable(false),
  }: Props = $props()

  const FORMATS = $derived([
    { id: 'mp3', title: 'MP3', description: t('audio.fmtMp3') },
    { id: 'opus', title: 'Opus', description: t('audio.fmtOpus') },
    { id: 'aac', title: 'AAC', description: t('audio.fmtAac') },
    { id: 'flac', title: 'FLAC', description: t('audio.fmtFlac') },
    { id: 'wav', title: 'WAV', description: t('audio.fmtWav') },
    { id: 'copy', title: t('audio.fmtCopy'), description: t('audio.fmtCopy') },
  ])

  const LOSSLESS: Record<string, boolean> = { flac: true, wav: true, copy: true }

  const DEFAULT_BITRATE: Record<string, number> = { mp3: 192, opus: 128, aac: 160 }

  let codec = $state('mp3')
  let bitrateKbps = $state(192)
  let channels = $state(0)
  let normalize = $state(false)
  let trimStart = $state('')
  let trimEnd = $state('')

  const codecOptions: RadioOption[] = $derived(FORMATS.map((f) => ({ id: f.id, title: f.title, description: f.description })))

  const channelOptions: RadioOption[] = $derived([
    { id: '0', title: t('audio.keep'), description: t('audio.keepDesc') },
    { id: '1', title: t('audio.mono'), description: t('audio.monoDesc') },
    { id: '2', title: t('audio.stereo'), description: t('audio.stereoDesc') },
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

  const lossless = $derived(LOSSLESS[codec] ?? false)

  const validation = $derived.by<string>(() => {
    if (Number.isNaN(trimStartSec) || Number.isNaN(trimEndSec)) {
      return t('audio.trimInvalid')
    }
    if (trimStartSec > 0 && trimEndSec > 0 && trimEndSec <= trimStartSec) {
      return t('audio.trimOrder')
    }
    return ''
  })

  // O destino segue a pasta da configuração e o sufixo das Preferências. Com
  // "salvar como…" o usuário escolhe um caminho próprio (manual) que não é
  // sobrescrito ao trocar o formato.
  $effect(() => {
    const path = inputPath
    const fmt = codec
    if (!path || manual) return
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

  const specPlain = $derived.by<audio.Spec>(() => ({
    inputPath,
    codec,
    bitrateKbps: lossless ? 0 : bitrateKbps,
    sampleRate: 0,
    channels,
    trimStartSec: trimStartSec > 0 ? trimStartSec : 0,
    trimEndSec: trimEndSec > 0 ? trimEndSec : 0,
    normalize: normalize && !lossless,
    outputPath: outputPath ?? '',
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

<div class="meta-k spacer">{t('audio.format')}</div>
<RadioCardGroup options={codecOptions} selected={codec} name={t('audio.format')} onchange={selectCodec} />

{#if !lossless}
  <div class="tune-row">
    <span class="meta-k">{t('audio.bitrate')}</span>
    <Stepper value={bitrateKbps} min={32} max={320} step={16} ariaLabel={t('audio.bitrate')} onchange={(v) => (bitrateKbps = v)} />
  </div>
{/if}

<div class="meta-k spacer">{t('audio.channels')}</div>
<RadioCardGroup options={channelOptions} selected={String(channels)} name={t('audio.channels')} onchange={(v) => (channels = Number(v))} />

<div class="spacer">
  <Toggle
    title={t('audio.normalize')}
    description={t('audio.normalizeDesc')}
    disabled={lossless}
    checked={normalize}
    onchange={(v) => (normalize = v)}
  />
  <InfoTip text={t('audio.normalizeTip')} />
</div>

<div class="trim-row">
  <div class="trim-field">
    <span class="meta-k">{t('audio.start')}</span>
    <input class="text-input mono" value={trimStart} aria-label={t('audio.start')} placeholder={t('audio.startPh')} oninput={(e) => (trimStart = e.currentTarget.value)} />
  </div>
  <div class="trim-field">
    <span class="meta-k">{t('audio.end')}</span>
    <input class="text-input mono" value={trimEnd} aria-label={t('audio.end')} placeholder={t('audio.endPh')} oninput={(e) => (trimEnd = e.currentTarget.value)} />
  </div>
</div>

{#if !hasAudio}
  <div class="hint">{t('audio.noTrack')}</div>
{/if}

<style>
  .spacer {
    margin-top: 14px;
  }
</style>