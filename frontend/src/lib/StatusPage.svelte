<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    title: string
    subtitle: string
    actionLabel: string
    onAction?: () => void
    children?: Snippet
  }

  let { title, subtitle, actionLabel, onAction = () => {}, children }: Props = $props()
</script>

<div class="status-page">
  <svg viewBox="0 0 24 24" aria-hidden="true">
    <path
      d="M6 2 h12 v20 h-12 z M10 0 h4 v2 h-4 z M10 6 l6 4 a2 2 0 0 1 -2 3 l-2 3 a2 2 0 0 1 -4 0 v-8 a2 2 0 0 1 2 -2 z M10 12 l6 -4"
    />
  </svg>
  <h1>{title}</h1>
  <p>{subtitle}</p>
  <button class="btn solid" onclick={onAction}>{actionLabel}</button>
  {@render children?.()}
</div>

<style>
  .status-page {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 48px 24px;
    text-align: center;
    border: 2px dashed var(--border);
    border-radius: var(--radius);
    margin: 18px;
    background: var(--card);
  }
  .status-page svg {
    width: 72px;
    height: 72px;
    fill: var(--accent);
    opacity: 0.85;
  }
  .status-page h1 {
    margin: 6px 0 0;
    font-size: 24px;
    font-weight: 700;
    color: var(--fg);
  }
  .status-page p {
    margin: 0 0 14px;
    color: var(--fg-dim);
  }
</style>