<script lang="ts">
  interface Props {
    text: string
    label?: string
  }

  let { text, label = 'Mais informações' }: Props = $props()
</script>

<details class="help">
  <summary aria-label={label}>
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 11v5M12 7.5h.01" stroke-linecap="round" />
    </svg>
  </summary>
  <div class="help-box">{text}</div>
</details>
