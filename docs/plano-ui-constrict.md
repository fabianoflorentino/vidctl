# Plano de UI — redesign estilo Constrict

Referência de UX/visual: [Wartybix/Constrict](https://github.com/Wartybix/Constrict)
(GTK4 + libadwaita, app GNOME que comprime vídeo para um tamanho-alvo).
Este plano **reproduz o visual e o fluxo no frontend Svelte 5 + Wails** — não há
porta para GTK/Python. As dependências de backend já estão orquestradas em
[`plano-implementacao.md`](./plano-implementacao.md) (chamadas de ↗ Fase N).

Branch de trabalho: `feat/ui-constrict` (tudo aqui é validado aqui antes de ir à main).

---

## O que faz o Constrict parecer "o Constrict" (extraído do source)

Do `window.blp`, `sources_row.blp`, `drag_overlay.blp`, `style.css`:

1. **Duas vistas em stack**: `status_page` (vazio, drop zone gigante com
   "Drag and drop videos here" + botão _Open…`) → `queue_page` (trabalho).
2. **Split view com sidebar** (~300–360 px) só com controles de compressão,
   agrupados em cards com título (`Framerate Limit`, `Encoding Options`,
   `Advanced Options`) e um botão `?` no cabeçalho de cada grupo (popover de
   explicação).
3. **Lista de fontes**: uma `Adw.ActionRow` por vídeo — thumbnail à esquerda,
   título (nome do arquivo), subtítulo com a transformação (`1080p@30 →
   720p@30`), e **uma suíte de sufixos por estado**: ícone de erro,
   pie de progresso de 16 px clicável (popover com detalhes), spinner, ✔ de
   concluído (popover com "Show File Location") e menu ⋮ (Move Up / Move Down /
   Remove — reordenação por drag também).
4. **Estados por linha**: queued → compressing (pie+spinner) → done ✔ / error ✕
   (com details em popover). Banner de aviso global quando alguma fonte está
   inválida (`Adw.Banner` "fix before compressing").
5. **Ação primária única**: pill `Export To…` (escolhe pasta de saída para o
   lote inteiro); durante o processa a barra some e vira `Cancel…`.
6. **Feedback global**: barra de progresso OSD no topo da janela durante o lote;
   toasts para eventos.
7. **Responsivo**: breakpoint `max-width: 750sp` colapsa a sidebar em
   "show_sidebar_button" (vira página, estilo mobile).
8. **Preferências** em dialog próprio (GPU encoding, sufixo de exportação);
   menu hambúrguer com Preferences / Keyboard Shortcuts / About.
9. **Visual adwaita**: fundo claro (ou dark do sistema), cards com cantos
   ~12 px, labels "dim", botões pill, tipografia Cantarell-like.

## Princípios do redesign no vidctl

- Manter o backend atual intacto onde der: `GetMediaInfo`, `Compress`,
  `Cancel`, eventos `compress:*` já servem para U0–U2.
- Componentizar o `App.svelte` monolito (954 linhas) em `frontend/src/lib/`.
- Tokens de design em CSS (leve clone do vocabulário adwaita), com tema
  claro/escuro por `prefers-color-scheme` (o app hoje é só dark).
- Estado da fila no frontend como `$derived`/`$state` sobre os eventos
  `compress:progress|done|error` já existentes por `jobId`.
- Cada fase U é um PR pequeno, testável visualmente com `make build` +
  rodando o app; merge na main só quando o usuário aprovar o visual.

---

## Fases

### U0 — Tokens de design + tema adwaita-like (CSS puro)

- `frontend/src/style.css`: reescrever variáveis (`--bg`, `--card`, `--fg`,
  `--dim`, `--accent`, `--radius`, `--success`, `--error`), fontes do sistema
  (mantendo fallback mono), **light + dark** via `@media (prefers-color-scheme)`.
- Classes utilitárias de cards/rows/pills/banners/popovers.
- Restilizar o fluxo atual **sem mexer na lógica** (v1.0.17 com cara nova).
- Aceite: mesmos comportamentos de v1.0.17 funcionando; zero mudança em Go.
- Esforço: **0,5–1 d**.

### U1 — Estrutura de vistas + componentes

- Split de `App.svelte` em: `StatusPage` (drop zone vazio), `QueuePage`,
  `SidePanel` (groups com `?` popover), `VideoRow`, `ProgressPie` (SVG,
  stroke-dasharray), `Popover`, `Banner`, `Menu` (⋮).
- View stack controlado por estado: `appView = 'empty' | 'queue'`.
- Menu hambúrguer no masthead (Preferences placeholder, About).
- Aceite: fluxo atual (1 arquivo) roda todo pela nova estrutura.
- Esforço: **1–2 d**.

### U2 — Drag & drop real de arquivos (sem backend novo)

- `OnFileDrop` / `OnFileDropCallback` do runtime Wails + overlay "Drop Videos
  to Import" (clone do `drag_overlay.blp`); filtro por extensão no frontend e
  validação com `GetMediaInfo` por arquivo (rejeita com estado `broken` na
  linha, como o Constrict faz com `broken-video-symbolic`).
- Botão `+ Add Videos…` no fim da lista (chama `OpenInputDialog` multi-arquivo —
  precisa de `OpenMultipleDialog` no Go: método novo ~30 min).
- Aceite: arrastar 3 vídeos da mesa cria 3 linhas informadas; um `.txt` vira
  linha quebrada com tooltip.
- Esforço: **0,5–1 d** (+ Go `OpenMultipleDialog`).

### U3 — Fila com estado por linha (frontend; consome ↗ Fase 2)

- Depende da **↗ Fase 2** (backend enfileirar N jobs, eventos com `jobId`) e de
  um método novo `GetThumbnail(path) string` (frame via ffmpeg → PNG em
  `os.TempDir()` ou data URL; cache por path+mtime).
- VídeoRow com: thumbnail, nome, `WxHp@fps → alvo`, pie de progresso clicável
  (popover com stage/percent/ETA), ✔/✕, menu ⋮ com Remover/Reordenar,
  botão por linha para cancelar o job.
- Barra OSD no topo com progresso agregado do lote; `Clear All` só habilita
  sem job rodando; `Export To…` (↗ Fase 2: pasta de saída única).
- Aceite: 4 vídeos na fila, 2º em compressão com pie, 1º ✔, um com codec
  estranho ✕ com motivo no popover.
- Esforço frontend: **2–3 d** (+1 d backend `GetThumbnail`).

### U4 — Painel de opções estilo Constrict (consome ↗ Fases 3–4)

- `Target Size (MiB)` com stepper ±; `Framerate Limit` em radio group
  (Automatic / 30 / 60, com subtítulos); `Video Codec` dropdown
  (H.264/HEVC/AV1/VP9 — **↗ Fase 4**); `Extra Quality` switch (preset slow —
  **↗ Fase 4**); `Tolerance (%)` spin (**↗ Fase 3**, estimativa).
- Campos mapeiam 1:1 no `Job` (↗ Fase 2 já desenha `Job` extensível).
- Aceite: comprimir para 10 MiB com HEVC e limite 30 fps gera arquivo ≤ alvo.
- Esforço frontend: **1–2 d** (backend nas ↗ Fases 3–4).

### U5 — Preferences + dialogs (consome ↗ Fase 1)

- Dialog `Preferences`: GPU encoding (**↗ Fase 4**), sufixo de exportação
  (`-compressed` custom — novo campo em `config`), pasta inicial de
  open/export, paths de ffmpeg/ffprobe (↗ Fase 1).
- Dialog `Keyboard Shortcuts` + `About` (versão, licença).
- Aceite: config sobrevive a restart.
- Esforço: **1 d** (backend já é ↗ Fase 1).

### U6 — Responsivo + acessibilidade

- Breakpoints espelhando o do Constrict: `≤ 750 px` sidebar colapsa (toggle no
  header); grid de presets vira coluna.
- `focus-visible`, `aria-label` em botões de ícone, navegação por tab na fila,
  contraste AA nos dois temas.
- Esforço: **1 d**.

### U7 — i18n pt/en (↗ Fase 10, opcional)

- Mesmos textos de UI; usar catálogo `pt`/`en` com `locale` vindo do config.

---

## Ordem de execução na branch

```
U0 → U1 → U2          (só frontend + 1 método Go trivial)
   ↳ requer ↗Fase 1  → U5
   ↳ requer ↗Fase 2  → U3 → U4 (com ↗Fases 3–4)
U6 por último.
```

Sugestão de empacotamento de PRs desta branch:

1. **PR-A**: U0 + U1 (visual novo, sem fila) — testável à mão, risco baixo.
2. **PR-B**: U2 (drag&drop) + `OpenMultipleDialog`.
3. Depois: ↗Fase 1 e ↗Fase 2 no plano geral, e U3/U5 voltam para esta branch
   ou saem dela, conforme preferir.

## Testes

- **Backend** (regra do AGENTS.md): `OpenMultipleDialog`, `GetThumbnail` com
  `app_test.go`/`compress` usando fixtures pequenas; cobertura ≥ 80% mantida.
- **Frontend**: hoje o projeto não tem test runner de UI. Proponho `vitest` +
  `@testing-library/svelte` em U1 (mínimo: view stack, reducer de fila com
  eventos fake por `jobId`). Componentes visuais puros ficam para aceite manual
  com checklist neste doc.
- **Manual (por fase)**: `make build && build/bin/vidctl` e rodar o checklist
  de aceite da fase.

## Riscos

- `OnFileDrop` no Windows + acentos em caminhos (validar cedo na PR-B).
- Pie/SVG atualizando a cada frame de progresso pode custar — throttle dos
  eventos a ~10 Hz (debounce no emissor backend se necessário).
- Tema claro em cima do design atual (escuro, "neon") é reescrever a maior
  parte do `style.css` — por isso U0 isola isso e permite reverter fácil.
- Contrato dos eventos (`jobId`, stage) pode mudar na ↗Fase 2; U3 só começa
  com o contrato fechado.

## Esforço total estimado

| Bloco | Dias |
|---|---|
| U0–U2 (frontend visual + d&d) | 3–5 |
| U3 + thumbnail | 3–4 |
| U4 (UI; backend nas ↗Fases 3–4) | 1–2 |
| U5–U6 | 2 |
| **Total desta branch** | **~9–13 d úteis** |

Sem fila/codec (só U0–U2 + U6): **~4–6 d** para o app "parecer Constrict"
mantendo o fluxo de arquivo único — bom primeiro incremento de branch.
