<script lang="ts">
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'
  import Toggle from './Toggle.svelte'
  import { t, LANGS, type LangId } from './i18n.svelte'
  import type { update } from '../../wailsjs/go/models.js'

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
    notifyOnDone: boolean
    openFolderOnDone: boolean
    updateInfo: update.Info | null
    checkingUpdate: boolean
    onclose?: () => void
    ontheme?: (theme: ThemeChoice) => void
    onlanguage?: (lang: LangId) => void
    onRecheck?: () => void
    onoutputdir?: (value: string) => void
    onffmpegpath?: (value: string) => void
    onffprobepath?: (value: string) => void
    onaudiosuffix?: (value: string) => void
    onnotifydone?: (value: boolean) => void
    onopenfolder?: (value: boolean) => void
    onCheckUpdate?: () => void
    onOpenRelease?: (url: string) => void
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
    notifyOnDone,
    openFolderOnDone,
    updateInfo,
    checkingUpdate,
    onclose = () => {},
    ontheme,
    onlanguage,
    onRecheck,
    onoutputdir,
    onffmpegpath,
    onffprobepath,
    onaudiosuffix,
    onnotifydone,
    onopenfolder,
    onCheckUpdate = () => {},
    onOpenRelease = () => {},
  }: Props = $props()

  const themeOptions = $derived<RadioOption[]>([
    { id: 'system', title: t('prefs.themeSystem'), description: t('prefs.themeSystemDesc') },
    { id: 'light', title: t('prefs.themeLight'), description: t('prefs.themeLightDesc') },
    { id: 'dark', title: t('prefs.themeDark'), description: t('prefs.themeDarkDesc') },
  ])

  const languageOptions = $derived<RadioOption[]>(LANGS.map((l) => ({ id: l.id, title: l.label, description: '' })))

  function knownVersion(v: string | undefined): boolean {
    return !!v && /[0-9]+\.[0-9]+\.[0-9]+/.test(v)
  }

  const updateState: string = $derived(
    checkingUpdate
      ? 'checking'
      : !updateInfo?.ok
        ? 'offline'
        : updateInfo.hasUpdate
          ? 'available'
          : knownVersion(updateInfo.current)
            ? 'uptodate'
            : 'dev',
  )
</script>

<Modal {open} title={t('prefs.title')} {onclose}>
  <div class="modal-section">
    <h2>{t('update.title')}</h2>
    <div class="modal-row">
      <span class="toggle-label">
        <span class="radio-card-title">
          {#if updateState === 'checking'}
            {t('update.checking')}
          {:else if updateState === 'available'}
            {t('update.available')}
          {:else if updateState === 'uptodate'}
            {t('update.upToDate')}
          {:else if updateState === 'dev'}
            {t('update.dev')}
          {:else}
            {t('update.offline')}
          {/if}
        </span>
        {#if updateInfo?.ok && updateInfo.latest}
          <span class="radio-card-desc">{updateInfo.latest}</span>
        {/if}
      </span>
      <div class="row-actions">
        <button class="btn small" onclick={onCheckUpdate} disabled={checkingUpdate}>
          {t('update.check')}
        </button>
        {#if updateState === 'available' && updateInfo?.url}
          <button class="btn small solid" onclick={() => onOpenRelease(updateInfo!.url)}>
            {t('update.open')}
          </button>
        {/if}
      </div>
    </div>
  </div>

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
    <h2>{t('prefs.notifications')}</h2>
    <Toggle
      title={t('prefs.notifyDone')}
      description={t('prefs.notifyDoneDesc')}
      checked={notifyOnDone}
      onchange={(v) => onnotifydone?.(v)}
    />
    <Toggle
      title={t('prefs.openFolder')}
      description={t('prefs.openFolderDesc')}
      checked={openFolderOnDone}
      onchange={(v) => onopenfolder?.(v)}
    />
  </div>

  <div class="modal-section">
    <h2>{t('prefs.ffmpegSection')}</h2>
    <div class="modal-row">
      <span class="toggle-label">
        <span class="radio-card-title" class:ok={ffmpegOk} class:bad={!ffmpegOk}>
          <span class="ok-dot"></span>
          {ffmpegOk ? t('prefs.ffmpegOk') : t('prefs.ffmpegBad')}
        </span>
        <span class="radio-card-desc">
          {ffmpegMsg || t('prefs.ffmpegDesc')}
        </span>
      </span>
      <button class="btn small" onclick={onRecheck}>{t('prefs.recheck')}</button>
    </div>
  </div>
</Modal>