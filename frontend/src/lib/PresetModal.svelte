<script lang="ts">
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import { t } from './i18n.svelte'
  import { presets } from '../../wailsjs/go/models.js'

  interface Props {
    preset: presets.Preset | null
    onclose?: () => void
    onSave?: (p: presets.Preset) => Promise<void>
    onDelete?: (id: string) => Promise<void>
  }

  let { preset, onclose = () => {}, onSave, onDelete }: Props = $props()

  let name = $state('')
  let mode = $state<'size' | 'crf'>('size')
  let sizeMB = $state(10)
  let crf = $state(23)
  let audioBitrate = $state('96k')
  let codec = $state<'h264' | 'h265'>('h264')
  let hardware = $state('')
  let saving = $state(false)
  let error = $state('')

  $effect(() => {
    const base = preset ?? ({} as presets.Preset)
    name = base.name ?? ''
    mode = (base.mode === 'crf' ? 'crf' : 'size') as 'size' | 'crf'
    sizeMB = base.sizeMB ?? 10
    crf = base.crf ?? 23
    audioBitrate = base.audioBitrate ?? '96k'
    codec = (base.codec === 'h265' ? 'h265' : 'h264') as 'h264' | 'h265'
    hardware = base.hardware ?? ''
  })

  const modeOptions = $derived<RadioOption[]>([
    { id: 'size', title: t('sec.size'), description: t('preset.mb', { mb: Math.round(sizeMB) }) },
    { id: 'crf', title: 'CRF', description: t('quality.crfLabel') },
  ])

  const codecOptions = $derived<RadioOption[]>([
    { id: 'h264', title: 'H.264', description: '' },
    { id: 'h265', title: 'H.265 / HEVC', description: '' },
  ])

  const hardwareOptions = $derived<RadioOption[]>([
    { id: '', title: t('preset.hwSoftware'), description: '' },
    { id: 'nvenc', title: 'NVENC', description: '' },
    { id: 'amf', title: 'AMF', description: '' },
    { id: 'qsv', title: 'QSV', description: '' },
    { id: 'videotoolbox', title: 'VideoToolbox', description: '' },
  ])

  function buildValue(): presets.Preset {
    return new presets.Preset({
      id: preset?.id ?? '',
      name,
      description: '',
      mode,
      sizeMB: mode === 'size' ? sizeMB : 0,
      crf: mode === 'crf' ? crf : 0,
      audioBitrate,
      codec,
      hardware,
    })
  }

  async function save() {
    if (!onSave) return
    if (saving) return
    saving = true
    error = ''
    try {
      await onSave(buildValue())
      onclose()
    } catch (err) {
      error = String(err)
    } finally {
      saving = false
    }
  }

  async function remove() {
    if (!onDelete || !preset) return
    if (saving) return
    saving = true
    error = ''
    try {
      await onDelete(preset.id)
      onclose()
    } catch (err) {
      error = String(err)
    } finally {
      saving = false
    }
  }
</script>

<Modal open={true} title={preset ? t('preset.titleEdit') : t('preset.titleNew')} {onclose}>
  <div class="modal-section">
    <h2>{t('preset.name')}</h2>
    <input
      class="input mono"
      value={name}
      placeholder="My preset"
      aria-label={t('preset.name')}
      oninput={(e) => (name = e.currentTarget.value)}
    />
  </div>

  <div class="modal-section">
    <h2>{t('preset.mode')}</h2>
    <RadioCardGroup options={modeOptions} selected={mode} name={t('preset.mode')} onchange={(v) => (mode = v as 'size' | 'crf')} />
    {#if mode === 'size'}
      <label class="field">
        <span class="field-label">{t('size.megabytes')}</span>
        <input class="input mono" type="number" min={1} max={4096} step={1} value={sizeMB} oninput={(e) => (sizeMB = Number(e.currentTarget.value))} />
      </label>
    {:else}
      <label class="field">
        <span class="field-label">CRF</span>
        <input class="input mono" type="number" min={0} max={51} step={1} value={crf} oninput={(e) => (crf = Number(e.currentTarget.value))} />
      </label>
    {/if}
    <label class="field">
      <span class="field-label">{t('audio.bitrate')}</span>
      <input class="input mono" value={audioBitrate} placeholder="96k" aria-label={t('audio.bitrate')} oninput={(e) => (audioBitrate = e.currentTarget.value)} />
    </label>
  </div>

  <div class="modal-section">
    <h2>{t('preset.codec')}</h2>
    <RadioCardGroup options={codecOptions} selected={codec} name={t('preset.codec')} onchange={(v) => (codec = v as 'h264' | 'h265')} />
  </div>

  <div class="modal-section">
    <h2>{t('preset.hw')}</h2>
    <RadioCardGroup options={hardwareOptions} selected={hardware} name={t('preset.hw')} onchange={(v) => (hardware = v)} />
  </div>

  {#if error}
    <div class="alert">
      <p class="mono">{error}</p>
    </div>
  {/if}

  <div class="modal-row actions">
    <div>
      {#if preset}
        <button class="btn danger small" onclick={remove} disabled={saving}>{t('preset.delete')}</button>
      {/if}
    </div>
    <div class="row-actions">
      <button class="btn small" onclick={onclose}>{t('preset.cancel')}</button>
      <button class="btn small solid" onclick={save} disabled={saving || !name.trim()}>{t('preset.save')}</button>
    </div>
  </div>
</Modal>