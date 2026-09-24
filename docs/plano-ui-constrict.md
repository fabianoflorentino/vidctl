# Plano de implementação — redesign estilo Constrict + drag and drop

Referência de UX/visual: [Wartybix/Constrict](https://github.com/Wartybix/Constrict)
(app de compressão de vídeo com sidebar de cards, fila de fontes, modal de
Preferências e empty state com drop zone). Este plano **reproduz o visual e o
fluxo no frontend Svelte 5 + Wails** — não há porta para GTK. Dependências de
backend futuras continuam orquestradas em
[`plano-implementacao.md`](./plano-implementacao.md) (chamadas de ↗ Fase N).

Branch de trabalho: `feat/ui-constrict`. Cada fase termina com o app
compilando e o fluxo atual funcionando; merge só com aceite visual do usuário.

---

## Decisões técnicas (verificadas no source do Wails v2.16.0)

- **Drag and drop nativo do Wails**, não HTML5: em `main.go`,
  `options.App.DragAndDrop = &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true}`.
  O alvo do drop é marcado no CSS com a propriedade customizada
  `--wails-drop-target: drop` (defaults `CSSDropProperty`/`CSSDropValue`); o
  frontend recebe os caminhos absolutos pelo evento `wails:file-drop` via
  `EventsOn('wails:file-drop', (x, y, paths) => …)` — mesmo `EventsOn` já
  importado em `App.svelte`.
- **Thumbnails**: binding Go novo `GetThumbnail(path)` extrai frame via ffmpeg
  (`-ss <min(1s,dur/2)> -frames:v 1`) e retorna **data-URL base64** (evita
  servir arquivo local no webview); cache em `os.TempDir()/vidctl-thumbs`
  chaveado por path+mtime+size.
- **Escopo monovídeo mantido**: o visual de fila do Constrict é reproduzido
  com 1 vídeo ativo (drop substitui a seleção). Fila real de N jobs é a
  ↗ Fase 2 de `plano-implementacao.md` (evolução futura, fora deste plano).
- **Tema**: dark é o alvo visual do redesign; o tema claro continua existindo,
  rederivado dos mesmos tokens (o app já tem claro/escuro persistido em
  `localStorage['vidctl-theme']`).
- **Preferências**: enquanto a ↗ Fase 1 (config persistente no Go) não chega,
  o modal persiste em `localStorage` (chave de sufixo nova); migrar depois.

## O que faz o visual ser "Constrict" (checklist de referência)

1. Empty state: ícone de câmera, título "Comprimir Vídeos", subtítulo
   "Drag and drop videos here", botão pill "Open…" — tudo é drop zone.
2. Split view: sidebar (~300–360 px) com cards empilhados, cada um com título
   e ícone ⓘ de ajuda; controles dentro dos cards.
3. Controles característicos: stepper −/+ redondo (Target Size, Tolerance),
   radio-cards com título+descrição (Framerate Limit), toggle switch
   (Extra Quality), select (Video Codec).
4. Lista de fontes: linha com thumbnail, nome, transformação
   (`720p@24 → 144p@24`) e menu ⋮; header da lista com "Clear All".
5. Ação primária única em pill no rodapé ("Export To…" / aqui "Comprimir…").
6. Modal "Preferências" centralizado com cards internos e toggles.
7. Paleta dark fosca (janela `#141414`, sidebar `#1e1e1e`, cards `#262626`),
   cantos 12–16 px, tipografia sans (mono só para números/dados).

---

## Fases

### Fase 0 — Baseline (0,5 h)

- Criar branch `feat/ui-constrict`; rodar `go test ./...` e
  `npm run check` em `frontend/`; screenshot do UI atual para comparação.
- Aceite: ponto de partida verde, sem mudança de código.

### Fase 1 — Design system em `frontend/src/style.css` (0,5–1 d)

- Novos tokens: paleta Constrict (ver checklist §7), radius 12–16 px, botões
  pill, ícones redondos; sans como fonte principal (IBM Plex Mono restrito a
  dados numéricos); tema claro rederivado dos mesmos tokens.
- Classes base novas: `.card`, `.group-card`, `.stepper`, `.toggle`,
  `.radio-card`, `.select`, `.modal`, `.pill-btn`, `.icon-round`, `.kebab`,
  `.dropzone` (com `--wails-drop-target: drop`).
- Restiliza o fluxo atual **sem mexer em lógica**.
- Aceite: `npm run check` verde; fluxo v-atual funcionando com a pele nova.

### Fase 2 — Componentes em `frontend/src/lib/` (1 d)

- `Stepper.svelte` (valor + botões −/+ redondos, min/max) → substitui sliders
  de tamanho alvo, partes e min/parte.
- `Toggle.svelte` (switch com título+descrição).
- `RadioCardGroup.svelte` (radio cards com título+descrição) → presets e modo
  de corte.
- `Modal.svelte` (overlay, título, botão fechar) → Preferências.
- `InfoTip.svelte` (ⓘ com tooltip) → substitui o `help` do `ControlGroup`.
- `KebabMenu.svelte` (⋮: trocar vídeo, abrir pasta, limpar).
- Aceite: `svelte-check` verde; componentes consumidos nas Fases 3–5
  (projeto não tem test runner de UI; ver §Testes).

### Fase 3 — Layout + drag and drop (1–1,5 d)

- `App.svelte`: topbar mínima (título + hambúrguer → Preferências); sidebar em
  cards (Tamanho alvo, Presets, Corte, Saída) usando os componentes da Fase 2;
  main com lista estilo "Video Sources" (linha do vídeo ativo + header com
  limpar); pill "Comprimir…" no rodapé substituindo a actionbar.
- `StatusPage.svelte` vira o empty state do checklist §1 (ícone, textos, pill
  "Open…"), inteiro como drop zone.
- Drag and drop:
  - `main.go`: opção `DragAndDrop` (ver §Decisões).
  - `--wails-drop-target: drop` no empty state e na área da lista.
  - `App.svelte`: `EventsOn('wails:file-drop', …)` → filtra extensão de vídeo,
    roda o fluxo `GetMediaInfo` existente; drop substitui o vídeo atual;
    não-vídeo → alert com motivo.
  - Highlight durante dragover via listeners `window.dragover/dragleave`
    (fallback aceitável: sem highlight, drop funciona).
- Aceite: arrastar `.mp4/.mov/.mkv/.webm` seleciona e informa o vídeo;
  arrastar `.txt` mostra erro; clique em "Open…" continua funcionando.

### Fase 4 — Thumbnails via ffmpeg (0,5–1 d)

- `internal/media/thumbnail.go`: extração de frame + cache (ver §Decisões);
  binding `GetThumbnail(path)` em `app.go` retornando data-URL base64.
- `VideoRow.svelte`: thumbnail + `WxHp@fps → alvo` + `KebabMenu`.
- Testes em `internal/media` com fixture gerado por ffmpeg lavfi (padrão dos
  testes de integração existentes); `make cover` ≥ 80%.
- Aceite: linha do vídeo mostra thumb em < 1 s após seleção; segundo select do
  mesmo arquivo usa cache.

### Fase 5 — Modal Preferências (0,5 d)

- `PreferencesModal.svelte` (aberto pelo hambúrguer): tema
  (sistema/claro/escuro), sufixo de saída (default `-compressed`), botão
  "verificar ffmpeg de novo".
- Persistência em `localStorage` (tema já existe; sufixo novo); aplicação do
  sufixo no caminho sugerido em `pickInput`.
- Aceite: preferências sobrevivem a restart; sufixo aparece no outputPath
  sugerido.

### Fase 6 — Docs, qualidade e release (0,5 d)

- README: drag and drop, thumbnails, novo UI; CHANGELOG; screenshots no
  `site/`; atualizar este doc com o que divergir na execução.
- `go test -race -shuffle=on -count=1 ./...`, `make cover` (≥ 80%),
  `npm run check`, `wails build -clean -tags webkit2_41`, smoke manual
  (`VIDCTL_SMOKE=… go test -run TestSmokeReal ./internal/compress/`).
- Aceite: CI verde + checklist visual §"O que faz o visual ser Constrict"
  conferido no binário buildado.

---

## Ordem de execução e empacotamento

```
Fase 0 → Fase 1 → Fase 2 → Fase 3 → Fase 4 → Fase 5 → Fase 6
```

PRs sugeridos: **PR-A** Fases 0–2 (pele + componentes, risco baixo);
**PR-B** Fase 3 (layout + drag and drop); **PR-C** Fases 4–5 (thumbs +
Preferências); **PR-D** Fase 6 (docs/release).

## Testes

- **Backend** (regra do AGENTS.md): `GetThumbnail` e filtros de extensão com
  testes em `internal/media`/`app_test.go`; cobertura ≥ 80% mantida
  (`go test -race ./...` + gate no CI).
- **Frontend**: sem test runner de UI hoje; garantia via `svelte-check`
  (`npm run check`) + checklist de aceite manual por fase com
  `make build && build/bin/vidctl`. Introduzir `vitest` +
  `@testing-library/svelte` fica como evolução opcional (não bloqueia).
- **Manual por fase**: rodar o aceite listado na fase antes de considerar
  concluída.

## Riscos

- `wails:file-drop` no Windows com caminhos contendo acentos/espaços —
  validar cedo na Fase 3 (PR-B).
- Highlight de dragover pode não firing em algum WebKit/WebView2 (DOM events
  suprimidos com `DisableWebViewDrop`) — degradar sem highlight é aceitável.
- Base64 de thumb de vídeos 4K pode pesar memória — limitar lado maior a
  ~320 px no ffmpeg (`-vf scale`) já na Fase 4.
- Reescrever `style.css` inteiro toca os dois temas — Fase 1 isolada em PR-A
  permite revert rápido.
- Eventos `compress:*`/`Job` podem mudar na ↗ Fase 2 (fila real); este plano
  não depende deles além do contrato atual.

## Esforço total

| Bloco | Dias |
|---|---|
| Fase 0–2 (baseline, tokens, componentes) | 1,5–2,5 |
| Fase 3 (layout + drag and drop) | 1–1,5 |
| Fase 4 (thumbnails) | 0,5–1 |
| Fase 5–6 (Preferências, docs/release) | 1 |
| **Total** | **~4–6 d úteis** (~1,5–2 d com agente executando) |

## Evoluções futuras (fora deste plano)

- Fila real de N vídeos com progresso por linha: ↗ Fase 2 de
  [`plano-implementacao.md`](./plano-implementacao.md).
- Codec/GPU encoding e tolerance: ↗ Fases 3–4 do mesmo plano.
- Config persistente no Go substituindo `localStorage`: ↗ Fase 1.
- Responsivo (sidebar colapsada ≤ 750 px) e i18n pt/en: opcional, pós-Fase 6.
