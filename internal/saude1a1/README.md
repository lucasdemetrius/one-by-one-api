# Módulo `saude1a1`

"Saúde do 1:1" do gestor: um resumo de **cadência** que fecha o ciclo de engajamento —
o produto coleta muito (humor, PDI, registros) e aqui devolve ao gestor um retorno
**motivacional** sobre o próprio hábito de fazer 1:1. Alimenta o card do `/painel`.

> Leitura agregada, **somente leitura** (sem mapper, como o `auditoria`). O catálogo
> geral está na [seção 12 do CLAUDE.md](../../CLAUDE.md) → [docs/CATALOGO.md](../../docs/CATALOGO.md).

## Endpoint

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /saude-1a1` | Retorna `{ percentual_em_dia, total_agendados, atrasados, realizados_ult_30, streak_semanas }` do gestor logado | **ApenasLider** — só o gestor, escopo `usuario_id` do token |

## Como os números saem (do modelo REAL)

`tb_onebyone` e `tb_agendamentos` são **silos independentes** (sem FK). Por isso:

- **Cadência esperada** vem de `tb_agendamentos` (ativos do gestor).
- **Realizados** vêm de `tb_onebyone` com `status='REALIZADO'` e `realizado_em` (livro-razão
  alimentado pelo ritual `POST /onebyone/encerrar`).
- **percentual_em_dia** = `(agendados − atrasados) / agendados × 100` (100 se não há agenda).
- **atrasados** = agendamentos ativos com `data_hora < agora` (ocorrência vencida).
- **streak_semanas** = semanas ISO consecutivas (andando para trás a partir de hoje) com
  ≥1 realizado. **Tolerante**: a semana atual ainda sem 1:1 não quebra a sequência.
  Calculado em Go (`calcularStreak`), não em SQL — coberto por `usecase_test.go`.

## Posse / segurança

Cadeia A: tudo escopado por `usuario_id = JWT`. `ApenasLider` barra contas COLABORADOR.
Não recebe `:id` de terceiro. Repositório só faz `COUNT`/datas (sem carregar listas).

## Dependências

Lê `tb_onebyone` e `tb_agendamentos`. Relaciona-se com o ritual de encerrar do módulo
[onebyone](../onebyone/README.md), que grava os realizados.
