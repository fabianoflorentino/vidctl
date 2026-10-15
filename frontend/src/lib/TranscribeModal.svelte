<script lang="ts">
  import { untrack } from 'svelte'
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import Toggle from './Toggle.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import { SuggestTranscribeOutput } from '../../wailsjs/go/main/App.js'
  import type { transcribe } from '../../wailsjs/go/models.js'
  import { t } from './i18n.svelte'

  interface Props {
    inputPath: string
    engines: string[]
    engine?: string
    language?: string
    format?: string
    wordTimestamps?: boolean
    modelPath?: string
    onclose?: () => void
    onRun?: (spec: transcribe.Spec) => Promise<void>
  }

  let {
    inputPath,
    engines,
    engine: engineDefault = 'auto',
    language: languageDefault = 'auto',
    format: formatDefault = 'all',
    wordTimestamps: wordsDefault = false,
    modelPath: modelDefault = '',
    onclose = () => {},
    onRun,
  }: Props = $props()

  const ENGINE_LABEL: Record<string, string> = { whispercpp: 'whisper.cpp', vosk: 'Vosk' }

  let engine = $state(untrack(() => engineDefault))
  let language = $state(untrack(() => languageDefault))
  let format = $state(untrack(() => formatDefault))
  let wordTimestamps = $state(untrack(() => wordsDefault))
  let modelPath = $state(untrack(() => modelDefault))
  let trimStart = $state('')
  let trimEnd = $state('')
  let outputBase = $state('')
  let busy = $state(false)
  let error = $state('')

  const engineOptions = $derived<RadioOption[]>([
    { id: 'auto', title: t('transcribe.engineAuto'), description: engines.map((e) => ENGINE_LABEL[e] ?? e).join(' · ') },
    ...engines.map((e) => ({ id: e, title: ENGINE_LABEL[e] ?? e, description: '' })),
  ])

  const languageOptions = $derived<RadioOption[]>([
    { id: 'auto', title: t('transcribe.langAuto'), description: '' },
    { id: 'pt', title: t('transcribe.langPt'), description: '' },
    { id: 'en', title: t('transcribe.langEn'), description: '' },
  ])

  const formatOptions = $derived<RadioOption[]>([
    { id: 'all', title: t('transcribe.formatAll'), description: '' },
    { id: 'srt', title: 'SRT', description: '' },
    { id: 'vtt', title: 'VTT', description: '' },
    { id: 'txt', title: 'TXT', description: '' },
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
  const noEngine = $derived(engines.length === 0)

  const validation = $derived.by<string>(() => {
    if (Number.isNaN(trimStartSec) || Number.isNaN(trimEndSec)) return t('transcribe.trimInvalid')
    if (trimStartSec > 0 && trimEndSec > 0 && trimEndSec <= trimStartSec) return t('transcribe.trimOrder')
    if (noEngine) return t('transcribe.needEngine')
    return ''
  })

  const canRun = $derived(!busy && !noEngine && !validation && !!outputBase)

  $effect(() => {
    const path = inputPath
    if (!path) return
    let cancelled = false
    SuggestTranscribeOutput(path)
      .then((p) => {
        if (!cancelled) outputBase = p
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    if (format === 'txt') wordTimestamps = false
  })

  const spec = $derived.by<transcribe.Spec>(() => ({
    inputPath,
    outputBase,
    engine,
    language,
    modelPath,
    format,
    wordTimestamps,
    trimStartSec: trimStartSec > 0 ? trimStartSec : 0,
    trimEndSec: trimEndSec > 0 ? trimEndSec : 0,
  }))

  async function run() {
    if (!onRun || !canRun) return
    busy = true
    error = ''
    try {
      await onRun(spec)
      onclose()
    } catch (err) {
      error = String(err)
    } finally {
      busy = false
    }
  }
</script>

<Modal open={true} title={t('transcribe.title')} {onclose}>
  <div class="modal-section">
    <h2>{t('transcribe.engine')}</h2>
    <RadioCardGroup options={engineOptions} selected={engine} name={t('transcribe.engine')} onchange={(v) => (engine = v)} />
    {#if noEngine}
      <div class="hint">{t('transcribe.engineNone')}</div>
    {/if}
  </div>

  <div class="modal-section">
    <h2>{t('transcribe.model')}</h2>
    <input
      class="input mono"
      value={modelPath}
      placeholder={t('transcribe.modelPlaceholder')}
      aria-label={t('transcribe.model')}
      oninput={(e) => (modelPath = e.currentTarget.value)}
    />
    <div class="hint with-tip">
      <span>{t('transcribe.modelHint')}</span>
      <InfoTip text={t('transcribe.modelHint')} />
    </div>
  </div>

  <div class="modal-section">
    <h2>{t('transcribe.language')}</h2>
    <RadioCardGroup options={languageOptions} selected={language} name={t('transcribe.language')} onchange={(v) => (language = v)} />
  </div>

  <div class="modal-section">
    <h2>{t('transcribe.format')}</h2>
    <RadioCardGroup options={formatOptions} selected={format} name={t('transcribe.format')} onchange={(v) => (format = v)} />
    <div class="spacer">
      <Toggle
        title={t('transcribe.words')}
        description={t('transcribe.wordsDesc')}
        disabled={format === 'txt'}
        checked={wordTimestamps}
        onchange={(v) => (wordTimestamps = v)}
      />
    </div>
  </div>

  <div class="modal-section">
    <h2>{t('transcribe.window')}</h2>
    <div class="trim-row">
      <div class="trim-field">
        <span class="meta-k">{t('transcribe.start')}</span>
        <input class="text-input mono" value={trimStart} aria-label={t('transcribe.start')} placeholder="0:00" oninput={(e) => (trimStart = e.currentTarget.value)} />
      </div>
      <div class="trim-field">
        <span class="meta-k">{t('transcribe.end')}</span>
        <input class="text-input mono" value={trimEnd} aria-label={t('transcribe.end')} placeholder="0:00" oninput={(e) => (trimEnd = e.currentTarget.value)} />
      </div>
    </div>
  </div>

  <div class="modal-section">
    <h2>{t('transcribe.output')}</h2>
    <div class="out-row">
      <span class="out-path mono" class:empty={!outputBase} title={outputBase}>{outputBase || '—'}</span>
    </div>
  </div>

  {#if validation}
    <div class="alert">
      <p class="mono">{validation}</p>
    </div>
  {/if}

  {#if error}
    <div class="alert">
      <p class="mono">{error}</p>
    </div>
  {/if}

  <div class="modal-row actions">
    <div></div>
    <div class="row-actions">
      <button class="btn small" onclick={onclose}>{t('transcribe.cancel')}</button>
      <button class="btn small solid" onclick={run} disabled={!canRun}>{t('transcribe.run')}</button>
    </div>
  </div>

  <style>
    .spacer {
      margin-top: 14px;
    }
    .hint.with-tip {
      display: flex;
      align-items: center;
      gap: 6px;
    }
  </style>
</Modal>
