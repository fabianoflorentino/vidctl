<script lang="ts">
  import ProgressPie from './ProgressPie.svelte'
  import type { media } from '../../wailsjs/go/models'

  type RowStatus = 'idle' | 'progress' | 'done' | 'error'

  interface Props {
    info: media.Info
    status?: RowStatus
    stage?: string
    percent?: number
    savedPct?: number
    onSwitch?: () => void
  }

  let { info, status = 'idle', stage = '', percent = 0, savedPct = 0, onSwitch = () => {} }: Props = $props()

  const orientation = $derived(info.width > info.height ? '16:9' : '9:16')

  function fmtMB(n: number): string {
    if (!n || n <= 0) return '—'
    return n.toFixed(1) + ' MB'
  }

  function fmtDuration(sec: number): string {
    if (!sec || sec < 0) return '—'
    const m = Math.floor(sec / 60)
    const s = Math.floor(sec % 60)
    return m > 0 ? `${m}m${String(s).padStart(2, '0')}s` : `${s}s`
  }

  function stageLabel(s: string): string {
    const m = s.match(/^(.*?) · parte (\d+\/\d+)$/)
    if (m) {
      return `${stageLabel(m[1])} · ${m[2]}`
    }
    switch (s) {
      case 'pass1/2':
        return 'ANALISANDO (1/2)'
      case 'pass2/2':
        return 'COMPRIMINDO (2/2)'
      case 'encoding':
        return 'ENCODANDO'
      case 'done':
        return 'CONCLUÍDO'
      case 'queue':
        return 'ENFILEIRADO'
      default:
        return s.toUpperCase()
    }
  }
</script>

<div class="video-row">
  <div class="video-thumb">
    <span class="thumb-badge">{info.width}×{info.height}</span>
  </div>

  <div class="video-info">
    <div class="video-name" title={info.path}>{info.path.split('/').pop()}</div>
    <div class="video-sub mono">
      {orientation} · {fmtDuration(info.durationSec)} · {fmtMB(info.sizeMB)} · {info.videoCodec || '—'}
      {#if status === 'done'}
        → <span class="sub-saved">−{savedPct.toFixed(0)}%</span>
      {/if}
    </div>
  </div>

  <div class="video-status">
    {#if status === 'progress'}
      <span class="status-txt">{[stageLabel(stage), percent.toFixed(0) + '%'].join(' · ')}</span>
      <ProgressPie value={percent} />
    {:else if status === 'done'}
      <span class="icon-check" aria-label="concluído">
        <svg width="22" height="22" viewBox="0 0 24 24"><path d="M4 12 l5 5 l11 -11" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" /></svg>
      </span>
    {:else if status === 'error'}
      <span class="icon-err" aria-label="erro">
        <svg width="22" height="22" viewBox="0 0 24 24"><path d="M12 4 l8 14 h-16 z" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" /><circle cx="12" cy="14" r="1" fill="currentColor" /><path d="M12 9 v3" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
      </span>
    {:else}
      <button class="btn subtle small" onclick={onSwitch}>trocar arquivo</button>
    {/if}
  </div>
</div>

<style>
  .video-row {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 14px;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    margin: 4px 0 0;
    min-height: 72px;
  }
  .video-thumb {
    width: 74px;
    height: 44px;
    flex: none;
    border-radius: 6px;
    background: var(--inset);
    border: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--fg-faint);
  }
  .thumb-badge {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--fg-dim);
  }
  .video-info {
    flex: 1;
    min-width: 0;
  }
  .video-name {
    font-size: 14px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .video-sub {
    font-size: 11px;
    color: var(--fg-dim);
    margin-top: 2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sub-saved {
    color: var(--success);
    font-weight: 700;
  }
  .video-status {
    flex: none;
    display: flex;
    align-items: center;
    gap: 9px;
  }
  .status-txt {
    font-size: 11px;
    color: var(--fg-dim);
    white-space: nowrap;
  }
  .icon-check {
    color: var(--success);
    display: inline-flex;
  }
  .icon-err {
    color: var(--error);
    display: inline-flex;
  }
</style>