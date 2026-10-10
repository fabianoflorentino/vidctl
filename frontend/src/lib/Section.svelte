<script lang="ts">
  import type { Snippet } from 'svelte'
  import InfoTip from './InfoTip.svelte'

  interface Props {
    title: string
    open: boolean
    onchange?: (open: boolean) => void
    infoTip?: string
    children?: Snippet
  }

  let { title, open, onchange = () => {}, infoTip = '', children }: Props = $props()
</script>

<section class="group-card">
  <div
    class="group-card-head section-head"
    role="button"
    tabindex="0"
    aria-expanded={open}
    onclick={() => onchange(!open)}
    onkeydown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onchange(!open)
      }
    }}
  >
    <span class="group-card-title">{title}</span>
    <span class="section-end">
      {#if infoTip}<InfoTip text={infoTip} />{/if}
      <span class="section-chevron mono" aria-hidden="true">{open ? '▾' : '▸'}</span>
    </span>
  </div>
  {#if open}
    <div class="group-card-body">
      {@render children?.()}
    </div>
  {/if}
</section>