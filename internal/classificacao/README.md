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
| `repository.go` | Dados | Upsert, remoção e listagem por organização (JOIN com colaboradores) |
| `usecase.go` | Negócio | Valida o colaborador e persiste/lista |
| `controller.go` | HTTP | Rotas protegidas por JWT |

## Entidade e tabela

Tabela **`tb_classificacoes`** (migration `006`): `colaborador_id` (PK),
`desempenho`, `potencial` (BAIXO/MEDIO/ALTO), `atualizado_em`.

## Endpoints

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| `PUT` | `/api/v1/colaboradores/{id}/classificacao` | Define/atualiza a posição 9-box (upsert) | JWT (LÍDER dono) |
| `DELETE` | `/api/v1/colaboradores/{id}/classificacao` | Remove da 9-box (volta o liderado para "A classificar") | JWT (LÍDER dono) |
| `GET` | `/api/v1/organizacoes/{id}/classificacoes` | Lista as classificações dos liderados da org | JWT (gestor dono ou RH) |

## DTOs

- **`DefinirClassificacaoDTO`**: `desempenho`, `potencial` (ambos `oneof=BAIXO MEDIO ALTO`).
- **`ClassificacaoRespostaDTO`**: `colaborador_id`, `desempenho`, `potencial`.

## Regras de negócio

- **Upsert**: `INSERT ... ON DUPLICATE KEY UPDATE` — uma classificação por liderado.
- **Remover** (`DELETE`) apaga a linha (sem soft delete: é avaliação mutável) — o
  liderado volta para a bandeja "A classificar" e pode ser reclassificado depois.
- **Posse**: `Definir` e `Remover` exigem o **LÍDER dono** (`PertenceAoLider`) +
  `ApenasLider`; recurso alheio → 404. A listagem usa `OrganizacaoPertenceAoLider`
  (RH-aware) + `PermitirGestaoOuRH`.
- A listagem traz só liderados **ativos** da organização (JOIN + `deletado_em IS NULL`).

## Dependências

`NovoUseCase(repo, colaboradorUC)` — repositório + UseCase de colaborador (validação).
Montado em `cmd/api/rotas.go` após o módulo `colaborador`.
