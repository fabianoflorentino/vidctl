<script lang="ts">
  interface Props {
    value: number
    min: number
    max: number
    step?: number
    disabled?: boolean
    ariaLabel?: string
    onchange?: (value: number) => void
  }

  let { value, min, max, step = 1, disabled = false, ariaLabel = 'valor', onchange }: Props = $props()

  function bump(delta: number) {
    const next = Math.min(max, Math.max(min, value + delta))
    if (next !== value) onchange?.(next)
  }
</script>

<div class="stepper">
  <button
    class="icon-round"
    disabled={disabled || value <= min}
    onclick={() => bump(-step)}
    aria-label={`Diminuir ${ariaLabel}`}
  >
    −
  </button>
  <span class="stepper-value">{value}</span>
  <button
    class="icon-round"
    disabled={disabled || value >= max}
    onclick={() => bump(step)}
    aria-label={`Aumentar ${ariaLabel}`}
  >
    +
  </button>
</div>
