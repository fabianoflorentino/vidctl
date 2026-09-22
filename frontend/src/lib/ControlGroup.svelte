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

<style>
  .group {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: visible;
  }
  .group-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 10px 14px 6px;
  }
  .group-title {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--fg-dim);
  }
  .group-body {
    padding: 2px 8px 10px;
  }
  .help {
    position: relative;
  }
  .help summary {
    list-style: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    border: 1px solid var(--border);
    color: var(--fg-faint);
    font-family: var(--mono);
    font-size: 11px;
    cursor: pointer;
    user-select: none;
  }
  .help summary::-webkit-details-marker {
    display: none;
  }
  .help[open] summary {
    color: var(--accent);
    border-color: var(--accent);
  }
  .help-box {
    position: absolute;
    top: 26px;
    right: 0;
    z-index: 20;
    width: 240px;
    padding: 10px 12px;
    background: var(--card);
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    box-shadow: var(--shadow);
    color: var(--fg-dim);
    font-size: 12px;
    line-height: 1.4;
  }
</style>