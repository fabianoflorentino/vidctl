# Diretrizes para agentes (opencode e qualquer automação de código)

Regras antes de considerar uma tarefa concluída:

## Testes
- Toda alteração de código (fix, feature, refactor) deve vir acompanhada de
  testes que cubram o comportamento alterado.
- Rodar sempre: `go test ./...`
- Cobertura do projeto deve ficar em **>= 80%** (gate no CI).

## Documentação
- Comportamento novo ou alterado deve refletir no `README.md` e, quando
  aplicável, em `docs/`.
- Formatos, presets e comandos mudaram? Atualize a doc do respectivo recurso.

## Estilo
- Não adicionar comentários em código a menos que sejam realmente necessários.
- Seguir as convenções dos arquivos vizinhos (mesma linguagem, mesmos padrões).