<script lang="ts">
  import Modal from './Modal.svelte'
  import RadioCardGroup from './RadioCardGroup.svelte'
  import type { RadioOption } from './RadioCardGroup.svelte'

  type ThemeChoice = 'system' | 'light' | 'dark'

  interface Props {
    open: boolean
    theme: ThemeChoice
    ffmpegOk: boolean
    ffmpegMsg: string
    onclose?: () => void
    ontheme?: (theme: ThemeChoice) => void
    onRecheck?: () => void
  }

  let { open, theme, ffmpegOk, ffmpegMsg, onclose = () => {}, ontheme, onRecheck }: Props = $props()

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
