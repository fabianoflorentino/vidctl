<script lang="ts">
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import { t, LANGS, type LangId } from './i18n.svelte'

  type ThemeChoice = 'system' | 'light' | 'dark'

  interface Props {
    open: boolean
    theme: ThemeChoice
    language: LangId
    ffmpegOk: boolean
    ffmpegMsg: string
    outputDir: string
    ffmpegPath: string
    ffprobePath: string
    audioSuffix: string
    onclose?: () => void
    ontheme?: (theme: ThemeChoice) => void
    onlanguage?: (lang: LangId) => void
    onRecheck?: () => void
    onoutputdir?: (value: string) => void
    onffmpegpath?: (value: string) => void
    onffprobepath?: (value: string) => void
    onaudiosuffix?: (value: string) => void
  }

  let {
    open,
    theme,
    language,
    ffmpegOk,
    ffmpegMsg,
    outputDir,
    ffmpegPath,
    ffprobePath,
    audioSuffix,
    onclose = () => {},
    ontheme,
    onlanguage,
    onRecheck,
    onoutputdir,
    onffmpegpath,
    onffprobepath,
    onaudiosuffix,
  }: Props = $props()

  const themeOptions = $derived<RadioOption[]>([
    { id: 'system', title: t('prefs.themeSystem'), description: t('prefs.themeSystemDesc') },
    { id: 'light', title: t('prefs.themeLight'), description: t('prefs.themeLightDesc') },
    { id: 'dark', title: t('prefs.themeDark'), description: t('prefs.themeDarkDesc') },
  ])

  const languageOptions = $derived<RadioOption[]>(LANGS.map((l) => ({ id: l.id, title: l.label, description: '' })))
</script>

<Modal {open} title={t('prefs.title')} {onclose}>
  <div class="modal-section">
    <h2>
      {t('prefs.language')}
      <InfoTip text={t('prefs.languageTip')} />
    </h2>
    <RadioCardGroup options={languageOptions} selected={language} name={t('prefs.language')} onchange={(id) => onlanguage?.(id as LangId)} />
  </div>

  <div class="modal-section">
    <h2>{t('prefs.appearance')}</h2>
    <RadioCardGroup
      options={themeOptions}
      selected={theme}
      name={t('prefs.appearance')}
      onchange={(id) => ontheme?.(id as ThemeChoice)}
    />
  </div>

  <div class="modal-section">
    <h2>
      {t('prefs.binaries')}
      <InfoTip text={t('prefs.binariesTip')} />
    </h2>
    <label class="field">
      <span class="field-label">{t('prefs.ffmpeg')}</span>
      <input
        type="text"
        class="input mono"
        placeholder={t('prefs.pathPlaceholder')}
        value={ffmpegPath}
        oninput={(e) => onffmpegpath?.(e.currentTarget.value)}
      />
    </label>
    <label class="field">
      <span class="field-label">{t('prefs.ffprobe')}</span>
      <input
        type="text"
        class="input mono"
        placeholder={t('prefs.pathPlaceholder')}
        value={ffprobePath}
        oninput={(e) => onffprobepath?.(e.currentTarget.value)}
      />
    </label>
  </div>

  <div class="modal-section">
    <h2>
      {t('prefs.output')}
      <InfoTip text={t('prefs.outputTip')} />
    </h2>
    <label class="field">
      <span class="field-label">{t('prefs.outputDir')}</span>
      <input
        type="text"
        class="input mono"
        placeholder={t('prefs.outputDirPh')}
        value={outputDir}
        oninput={(e) => onoutputdir?.(e.currentTarget.value)}
      />
    </label>
  </div>

  <div class="modal-section">
    <h2>
      {t('prefs.audio')}
      <InfoTip text={t('prefs.audioTip')} />
    </h2>
    <label class="field">
      <span class="field-label">{t('prefs.audioSuffix')}</span>
      <input
        type="text"
        class="input mono"
        placeholder="-audio"
        value={audioSuffix}
        oninput={(e) => onaudiosuffix?.(e.currentTarget.value)}
      />
    </label>
  </div>

  <div class="modal-section">
    <h2>{t('prefs.ffmpegSection')}</h2>
    <div class="modal-row">
      <span class="toggle-label">
        <span class="radio-card-title">{ffmpegOk ? t('prefs.ffmpegOk') : t('prefs.ffmpegBad')}</span>
        <span class="radio-card-desc">
          {ffmpegMsg || t('prefs.ffmpegDesc')}
        </span>
      </span>
      <button class="btn small" onclick={onRecheck}>{t('prefs.recheck')}</button>
    </div>
  </div>
</Modal>