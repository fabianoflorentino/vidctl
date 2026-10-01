<script lang="ts">
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import InfoTip from './InfoTip.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'

  type ThemeChoice = 'system' | 'light' | 'dark'

  interface Props {
    open: boolean
    theme: ThemeChoice
    ffmpegOk: boolean
    ffmpegMsg: string
    outputDir: string
    ffmpegPath: string
    ffprobePath: string
    onclose?: () => void
    ontheme?: (theme: ThemeChoice) => void
    onRecheck?: () => void
    onoutputdir?: (value: string) => void
    onffmpegpath?: (value: string) => void
    onffprobepath?: (value: string) => void
  }

  let {
    open,
    theme,
    ffmpegOk,
    ffmpegMsg,
    outputDir,
    ffmpegPath,
    ffprobePath,
    onclose = () => {},
    ontheme,
    onRecheck,
    onoutputdir,
    onffmpegpath,
    onffprobepath,
  }: Props = $props()

  const themeOptions: RadioOption[] = [
    { id: 'system', title: 'Sistema', description: 'Segue o tema claro/escuro do sistema' },
    { id: 'light', title: 'Claro', description: 'Tema claro fixo' },
    { id: 'dark', title: 'Escuro', description: 'Tema escuro fixo' },
  ]
</script>

<Modal {open} title="Preferências" {onclose}>
  <div class="modal-section">
    <h2>Aparência</h2>
    <RadioCardGroup
      options={themeOptions}
      selected={theme}
      name="tema"
      onchange={(id) => ontheme?.(id as ThemeChoice)}
    />
  </div>

  <div class="modal-section">
    <h2>
      Binários
      <InfoTip text="Deixe em branco para usar o ffmpeg e o ffprobe do PATH do sistema. Preencha apenas se eles estiverem em outro lugar." />
    </h2>
    <label class="field">
      <span class="field-label">ffmpeg</span>
      <input
        type="text"
        class="input mono"
        placeholder="procurado no PATH"
        value={ffmpegPath}
        oninput={(e) => onffmpegpath?.(e.currentTarget.value)}
      />
    </label>
    <label class="field">
      <span class="field-label">ffprobe</span>
      <input
        type="text"
        class="input mono"
        placeholder="procurado no PATH"
        value={ffprobePath}
        oninput={(e) => onffprobepath?.(e.currentTarget.value)}
      />
    </label>
  </div>

  <div class="modal-section">
    <h2>
      Saída
      <InfoTip text="Pasta onde o vídeo comprimido é salvo. Vazio significa ao lado do arquivo original." />
    </h2>
    <label class="field">
      <span class="field-label">pasta de destino</span>
      <input
        type="text"
        class="input mono"
        placeholder="ao lado do vídeo original"
        value={outputDir}
        oninput={(e) => onoutputdir?.(e.currentTarget.value)}
      />
    </label>
  </div>

  <div class="modal-section">
    <h2>ffmpeg</h2>
    <div class="modal-row">
      <span class="toggle-label">
        <span class="radio-card-title">{ffmpegOk ? 'ffmpeg disponível' : 'ffmpeg ausente'}</span>
        <span class="radio-card-desc">
          {ffmpegMsg || 'Binários ffmpeg/ffprobe procurados no PATH do sistema.'}
        </span>
      </span>
      <button class="btn small" onclick={onRecheck}>verificar de novo</button>
    </div>
  </div>
</Modal>
