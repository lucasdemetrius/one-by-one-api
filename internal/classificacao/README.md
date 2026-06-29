# Módulo `classificacao`

> A matriz **9-box** dos liderados: posição de cada colaborador por **desempenho**
> × **potencial** (cada eixo BAIXO/MEDIO/ALTO). Base do Monitor do gestor para
> acompanhar a evolução, destacar talentos e antecipar riscos.

## O que faz

Guarda e lista o posicionamento 9-box de cada liderado. O gestor arrasta o
liderado para a célula na tela (Monitor); aqui isso vira um `desempenho` +
`potencial` persistidos.

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---|---|---|
| `entity.go` | Domínio | Struct `Classificacao` (tabela `tb_classificacoes`) + níveis |
| `dto.go` | Contrato | `DefinirClassificacaoDTO`, `ClassificacaoRespostaDTO` |
| `repository.go` | Dados | Upsert e listagem por organização (JOIN com colaboradores) |
| `usecase.go` | Negócio | Valida o colaborador e persiste/lista |
| `controller.go` | HTTP | Rotas protegidas por JWT |

## Entidade e tabela

Tabela **`tb_classificacoes`** (migration `006`): `colaborador_id` (PK),
`desempenho`, `potencial` (BAIXO/MEDIO/ALTO), `atualizado_em`.

## Endpoints

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| `PUT` | `/api/v1/colaboradores/{id}/classificacao` | Define/atualiza a posição 9-box (upsert) | JWT |
| `GET` | `/api/v1/organizacoes/{id}/classificacoes` | Lista as classificações dos liderados da org | JWT |

## DTOs

- **`DefinirClassificacaoDTO`**: `desempenho`, `potencial` (ambos `oneof=BAIXO MEDIO ALTO`).
- **`ClassificacaoRespostaDTO`**: `colaborador_id`, `desempenho`, `potencial`.

## Regras de negócio

- **Upsert**: `INSERT ... ON DUPLICATE KEY UPDATE` — uma classificação por liderado.
- Valida que o colaborador existe (via `colaboradorUseCase.BuscarPorId`).
- A listagem traz só liderados **ativos** da organização (JOIN + `deletado_em IS NULL`).

## Dependências

`NovoUseCase(repo, colaboradorUC)` — repositório + UseCase de colaborador (validação).
Montado em `cmd/api/rotas.go` após o módulo `colaborador`.
