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
      d="M4 6.5 h11 a2.5 2.5 0 0 1 2.5 2.5 v6 a2.5 2.5 0 0 1 -2.5 2.5 H4 a2.5 2.5 0 0 1 -2.5 -2.5 V9 a2.5 2.5 0 0 1 2.5 -2.5 z M18.5 10.8 L23 8.2 v7.6 l-4.5 -2.6 z"
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