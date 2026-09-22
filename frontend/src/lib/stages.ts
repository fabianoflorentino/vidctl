const BASE_LABELS: Record<string, string> = {
  'pass1/2': 'ANALISANDO (1/2)',
  'pass2/2': 'COMPRIMINDO (2/2)',
  encoding: 'ENCODANDO',
  done: 'CONCLUÍDO',
  queue: 'ENFILEIRADO',
}

export function stageLabel(s: string): string {
  const m = s.match(/^(.*?) · parte (\d+\/\d+)$/)
  if (m) {
    return `${stageLabel(m[1])} · parte ${m[2]}`
  }
  return BASE_LABELS[s] ?? s.toUpperCase()
}
