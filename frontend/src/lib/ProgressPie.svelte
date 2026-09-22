<script lang="ts">
  let { value, size = 20 }: { value: number; size?: number } = $props()

  const stroke = 3
  const r = $derived((size - stroke) / 2)
  const c = $derived(2 * Math.PI * r)
  const clamped = $derived(Math.min(100, Math.max(0, value)))
</script>

<svg
  width={size}
  height={size}
  viewBox={`0 0 ${size} ${size}`}
  role="progressbar"
  aria-valuenow={Math.round(clamped)}
  aria-valuemin={0}
  aria-valuemax={100}
>
  <circle class="pie-bg" cx={size / 2} cy={size / 2} r={r} stroke-width={stroke} />
  <circle
    class="pie-val"
    cx={size / 2}
    cy={size / 2}
    r={r}
    stroke-width={stroke}
    stroke-dasharray={c}
    stroke-dashoffset={c * (1 - clamped / 100)}
    transform={`rotate(-90 ${size / 2} ${size / 2})`}
  />
</svg>

<style>
  .pie-bg {
    fill: none;
    stroke: var(--border);
  }
  .pie-val {
    fill: none;
    stroke: var(--accent);
    stroke-linecap: round;
    transition: stroke-dashoffset 0.2s ease;
  }
</style>