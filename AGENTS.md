# Diretrizes para agentes (opencode e qualquer automação de código)

Regras antes de considerar uma tarefa concluída:

## Branches e pull requests
- **Uma branch por fase.** Cada fase do `docs/plano-implementacao.md` é
  implementada na sua própria branch, Naming: `feat/<slug>`, `fix/<slug>` ou
  `chore/<slug>`, descrevendo o escopo da fase (ex.: `feat/fase1`,
  `feat/audio`, `feat/transcricao`).
- Branch nova sai da `main` atual: nunca de outra branch de fase, para que o PR
  não acumule implementações de fases anteriores.
- A implementação é registrada como **commits na branch da fase**, com mensagens
  no Conventional Commits em português (ex.: `feat(config): persiste
  preferências em config.json`). Um commit por unidade lógica de mudança, e
  cada commit deve compilar e passar nos testes sozinho.
- A fase só está entregue quando o PR para a `main` está aberto. Não fazer merge
  direto na `main`, não dar push da branch sem pedido explícito.
- Atualizar a fase correspondente em `docs/plano-implementacao.md` **no mesmo
  PR** da implementação, para que o plano e o código não divirjam.

## Testes
- Toda alteração de código (fix, feature, refactor) deve vir acompanhada de
  testes que cubram o comportamento alterado.
- Rodar sempre: `go test ./...`
- Cobertura do projeto deve ficar em **>= 80%** (gate no CI e no `make check`).
- Manter os testes unitários acima de 80% a cada mudança; se a alteração ou
  cena nova baixar a cobertura, completar com testes antes de abrir o PR.

## Internacionalização (4 idiomas)
- Toda feature nova vira conteúdo em **4 idiomas** (pt-BR, en, es e zh-CN): o
  app usa os dicionários de `frontend/src/lib/i18n.svelte.ts` para strings da
  UI; o backend continua com mensagens-fonte em inglês (contrato com o front).
- O site (`site/index.html` + `site/css/main.css`) reflete as features e seus
  textos nas 4 línguas, incluindo o seletor de idiomas.
- Não traduzir: código, comentários, mensagens de commit, identificadores e o
  plano interno (`docs/plano-implementacao.md`), que permanecem como estão.

## Documentação
- Comportamento novo ou alterado deve refletir na documentação **em 4 idiomas:
  `README.md` (pt fica na raiz; `.en/.es/.zh-CN` ficam em `docs/README.*.md`),
  o mesmo para `CHANGELOG.md` (`docs/CHANGELOG.*.md`),
  `docs/instalacao.*.md` e o site.
- Formatos, presets e comandos mudaram? Atualize a doc do respectivo recurso em
  todas as línguas, nunca só na versão em português.
- Registrar a mudança em `CHANGELOG.md`, na seção `[Unreleased]`, enquanto não
  houver versão de release correspondente.

## Estilo
- Não adicionar comentários em código a menos que sejam realmente necessários.
- Seguir as convenções dos arquivos vizinhos (mesma linguagem, mesmos padrões).