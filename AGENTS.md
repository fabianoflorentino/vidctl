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
- Cobertura do projeto deve ficar em **>= 80%** (gate no CI).
- Antes de abrir o PR: `make check` (vet + testes + gofmt + svelte-check).

## Documentação
- Comportamento novo ou alterado deve refletir no `README.md` e, quando
  aplicável, em `docs/`.
- Formatos, presets e comandos mudaram? Atualize a doc do respectivo recurso.
- Registrar a mudança em `CHANGELOG.md`, na seção `[Unreleased]`, enquanto não
  houver versão de release correspondente.

## Estilo
- Não adicionar comentários em código a menos que sejam realmente necessários.
- Seguir as convenções dos arquivos vizinhos (mesma linguagem, mesmos padrões).