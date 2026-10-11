import { t } from './i18n.svelte'

const BASE_KEYS: Record<string, string> = {
  'pass1/2': 'stage.pass1',
  'pass2/2': 'stage.pass2',
  encoding: 'stage.encoding',
  extracting: 'stage.extracting',
  done: 'stage.done',
  queue: 'stage.queue',
}

export function stageLabel(s: string): string {
  const m = s.match(/^(.*?) · parte (\d+\/\d+)$/)
  if (m) {
    return `${stageLabel(m[1])}${t('stage.part', { part: m[2] })}`
  }
  const key = BASE_KEYS[s]
  return key ? t(key) : s.toUpperCase()
}