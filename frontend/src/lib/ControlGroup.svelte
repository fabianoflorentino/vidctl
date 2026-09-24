<script lang="ts">
  interface Props {
    title: string
    help?: string
    children?: import('svelte').Snippet
  }

  let { title, help = '', children }: Props = $props()
</script>

<div class="group">
  <div class="group-head">
    <span class="group-title">{title}</span>
    {#if help}
      <details class="help">
        <summary aria-label={`Mais informações sobre ${title}`}>?</summary>
        <div class="help-box">{help}</div>
      </details>
    {/if}
  </div>
  <div class="group-body">
    {@render children?.()}
  </div>
</div>