<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    title: string
    subtitle: string
    actionLabel: string
    onAction?: () => void
    hot?: boolean
    children?: Snippet
  }

  let { title, subtitle, actionLabel, onAction = () => {}, hot = false, children }: Props = $props()
</script>

<div class="status-page dropzone" class:hot>
  <svg viewBox="0 0 24 24" aria-hidden="true">
    <path
      fill-rule="evenodd"
      d="M12 3.000 A9.0 9.0 0 1 0 12 21.000 A9.0 9.0 0 1 0 12 3.000 Z M12.000 8.047 L15.424 10.023 L15.424 13.977 L12.000 15.953 L8.576 13.977 L8.576 10.023 Z M11.743 7.714 L12.257 8.380 L17.026 4.701 L16.512 4.035 Z M15.584 9.634 L15.264 10.412 L20.835 12.703 L21.154 11.925 Z M15.840 13.921 L15.007 14.033 L15.808 20.002 L16.642 19.891 Z M12.257 16.286 L11.743 15.620 L6.974 19.299 L7.488 19.965 Z M8.416 14.366 L8.736 13.588 L3.165 11.297 L2.846 12.075 Z M8.160 10.079 L8.993 9.967 L8.192 3.998 L7.358 4.109 Z"
    />
  </svg>
  <h1>{title}</h1>
  <p>{subtitle}</p>
  <button class="pill-btn" onclick={onAction}>{actionLabel}</button>
  {@render children?.()}
</div>
