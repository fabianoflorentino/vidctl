<script lang="ts">
  export interface KebabItem {
    id: string
    label: string
  }

  interface Props {
    items: KebabItem[]
    label?: string
    onselect?: (id: string) => void
  }

  let { items, label = 'Mais ações', onselect }: Props = $props()

  let open = $state(false)

  function pick(id: string) {
    open = false
    onselect?.(id)
  }

  function close() {
    open = false
  }
</script>

<svelte:window onclick={open ? close : undefined} />

<div class="kebab-wrap">
  <button
    class="kebab"
    aria-label={label}
    aria-expanded={open}
    onclick={(e) => {
      e.stopPropagation()
      open = !open
    }}
  >
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <circle cx="12" cy="5" r="1.8" />
      <circle cx="12" cy="12" r="1.8" />
      <circle cx="12" cy="19" r="1.8" />
    </svg>
  </button>
  {#if open}
    <div
      class="kebab-menu"
      role="menu"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => {
        if (e.key === 'Escape') close()
      }}
    >
      {#each items as item (item.id)}
        <button role="menuitem" onclick={() => pick(item.id)}>{item.label}</button>
      {/each}
    </div>
  {/if}
</div>
