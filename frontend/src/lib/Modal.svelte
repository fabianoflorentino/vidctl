<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    open: boolean
    title: string
    onclose?: () => void
    children?: Snippet
  }

  let { open, title, onclose = () => {}, children }: Props = $props()

  function onKey(e: KeyboardEvent) {
    if (open && e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={onKey} />

{#if open}
  <div
    class="modal-backdrop"
    role="presentation"
    onclick={(e) => {
      if (e.target === e.currentTarget) onclose()
    }}
  >
    <div class="modal" role="dialog" aria-modal="true" aria-label={title}>
      <div class="modal-head">
        <button class="icon-round modal-close" onclick={onclose} aria-label={`Fechar ${title}`}>
          <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
            <path d="M2 2 l8 8 M10 2 l-8 8" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
          </svg>
        </button>
        <span class="modal-title">{title}</span>
      </div>
      {@render children?.()}
    </div>
  </div>
{/if}
