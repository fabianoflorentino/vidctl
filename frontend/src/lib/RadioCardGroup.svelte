<script lang="ts">
  export interface RadioOption {
    id: string
    title: string
    description?: string
    disabled?: boolean
  }

  interface Props {
    options: RadioOption[]
    selected: string
    name?: string
    onchange?: (id: string) => void
  }

  let { options, selected, name = 'opções', onchange }: Props = $props()
</script>

<div class="radio-group" role="radiogroup" aria-label={name}>
  {#each options as opt (opt.id)}
    <button
      class="radio-card"
      class:selected={opt.id === selected}
      class:disabled={opt.disabled}
      role="radio"
      aria-checked={opt.id === selected}
      disabled={opt.disabled}
      onclick={() => onchange?.(opt.id)}
    >
      <span class="radio-card-text">
        <span class="radio-card-title">{opt.title}</span>
        {#if opt.description}<span class="radio-card-desc">{opt.description}</span>{/if}
      </span>
    </button>
  {/each}
</div>
