<script lang="ts">
  interface Props {
    checked: boolean
    title?: string
    description?: string
    rowClass?: string
    disabled?: boolean
    ariaLabel?: string
    onchange?: (checked: boolean) => void
  }

  let {
    checked,
    title = '',
    description = '',
    rowClass = 'tune-row',
    disabled = false,
    ariaLabel = 'Alternar',
    onchange,
  }: Props = $props()

  function flip() {
    if (!disabled) onchange?.(!checked)
  }
</script>

{#if title}
  <div class={rowClass}>
    <span class="toggle-label">
      <span class="radio-card-title">{title}</span>
      {#if description}<span class="radio-card-desc">{description}</span>{/if}
    </span>
    <button
      class="toggle"
      role="switch"
      aria-checked={checked}
      aria-label={title}
      disabled={disabled}
      onclick={flip}
    ></button>
  </div>
{:else}
  <button
    class="toggle"
    role="switch"
    aria-checked={checked}
    aria-label={ariaLabel}
    disabled={disabled}
    onclick={flip}
  ></button>
{/if}
