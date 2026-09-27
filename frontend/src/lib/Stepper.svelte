<script lang="ts">
  interface Props {
    value: number
    min: number
    max: number
    step?: number
    fineStep?: number
    digits?: number
    disabled?: boolean
    ariaLabel?: string
    onchange?: (value: number) => void
  }

  let {
    value,
    min,
    max,
    step = 1,
    fineStep,
    digits = 0,
    disabled = false,
    ariaLabel = 'valor',
    onchange,
  }: Props = $props()

  let editing = $state(false)
  let draft = $state('')

  const fine = $derived(fineStep ?? step / 10)

  function disp(n: number): string {
    return Number(n.toFixed(digits)).toString()
  }

  function clamp(n: number): number {
    const r = Number(Math.min(max, Math.max(min, n)).toFixed(digits))
    if (r === value || Number.isNaN(r)) return value
    return r
  }

  function bump(delta: number, fine_: boolean) {
    const d = fine_ ? fine : step
    const next = clamp(value + (Math.sign(delta) < 0 ? -d : d))
    if (next !== value) onchange?.(next)
  }

  function commit(raw: string) {
    editing = false
    const n = Number(String(raw).trim().replace(',', '.'))
    if (Number.isNaN(n)) return
    const next = clamp(n)
    if (next !== value) onchange?.(next)
  }

  function startEdit(e: FocusEvent) {
    editing = true
    draft = disp(value)
    const el = e.currentTarget as HTMLInputElement
    requestAnimationFrame(() => el.select())
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      commit(draft)
      ;(e.currentTarget as HTMLInputElement).blur()
    } else if (e.key === 'Escape') {
      editing = false
      ;(e.currentTarget as HTMLInputElement).blur()
    } else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      e.preventDefault()
      bump(e.key === 'ArrowUp' ? 1 : -1, e.shiftKey)
    }
  }
</script>

<div class="stepper">
  <button
    class="icon-round"
    disabled={disabled || value <= min}
    onclick={(e) => bump(-1, e.shiftKey)}
    aria-label={`Diminuir ${ariaLabel}`}
  >
    −
  </button>
  <input
    class="stepper-value"
    type="text"
    inputmode="decimal"
    value={editing ? draft : disp(value)}
    disabled={disabled}
    aria-label={ariaLabel}
    onfocus={startEdit}
    onblur={() => commit(draft)}
    oninput={(e) => (draft = e.currentTarget.value)}
    onkeydown={onKeydown}
  />
  <button
    class="icon-round"
    disabled={disabled || value >= max}
    onclick={(e) => bump(1, e.shiftKey)}
    aria-label={`Aumentar ${ariaLabel}`}
  >
    +
  </button>
</div>