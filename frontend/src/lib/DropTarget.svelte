<script lang="ts">
  interface Props {
    hot?: boolean
    onOpen: () => void
  }

  let { hot = false, onOpen }: Props = $props()
</script>

<div
  class="drop-target"
  class:hot
  role="button"
  tabindex="0"
  aria-label="Arraste um vídeo aqui ou clique para abrir"
  onclick={onOpen}
  onkeydown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onOpen()
    }
  }}
>
  <svg viewBox="0 0 24 24" aria-hidden="true">
    <path
      d="M12 16V6M7 10l5-5 5 5M5 16v2a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-2"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
  <span class="drop-target-hint"><strong>arraste um vídeo aqui</strong> ou clique para escolher</span>
  <button class="btn solid small" onclick={(e) => { e.stopPropagation(); onOpen() }}>abrir vídeo…</button>
</div>