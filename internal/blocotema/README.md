# Módulo `blocotema`

> Conteúdo rico de um tema de 1:1, por liderado — a "mini-apresentação" de cada
> assunto (ex.: "Plano de Carreira"): blocos de **texto**, **link/curso**,
> **imagem** (no S3) e **marco** com datas.

## O que faz

Cada tema, para um colaborador, pode ter vários blocos de conteúdo ordenados. O
gestor monta isso no editor de tema (drawer no tabuleiro do 1:1). As imagens vão
para o S3 e voltam como URL presignada.

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---|---|---|
| `entity.go` | Domínio | Struct `BlocoTema` (tabela `tb_blocos_tema`) + tipos de bloco |
| `dto.go` | Contrato | `CriarBlocoDTO`, `BlocoRespostaDTO` |
| `repository.go` | Dados | Listar/criar/deletar + próxima ordem |
| `usecase.go` | Negócio | Cria blocos, faz upload S3 da imagem, gera URLs presignadas |
| `controller.go` | HTTP | Rotas protegidas por JWT |

## Entidade e tabela

Tabela **`tb_blocos_tema`** (migration `005`): `id`, `colaborador_id`, `tema`,
`tipo` (TEXTO/LINK/IMAGEM/MARCO), `texto`, `url`, `imagem_key`, `data_inicio`,
`data_fim`, `ordem`, `criado_em`.

## Endpoints

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| `GET` | `/api/v1/colaboradores/{id}/blocos?tema=` | Lista os blocos de um tema | JWT |
| `POST` | `/api/v1/colaboradores/{id}/blocos` | Cria bloco de texto/link/marco | JWT |
| `POST` | `/api/v1/colaboradores/{id}/blocos-imagem` | Upload de imagem (multipart) → bloco IMAGEM | JWT |
| `DELETE` | `/api/v1/colaboradores/{id}/blocos/{blocoId}` | Remove um bloco | JWT |

## DTOs

- **`CriarBlocoDTO`**: `tema`, `tipo` (`oneof=TEXTO LINK MARCO`), `texto?`, `url?`,
  `data_inicio?`, `data_fim?` (datas `YYYY-MM-DD`).
- **`BlocoRespostaDTO`**: idem + `imagem_url` (presignada) no lugar da chave.

## Regras de negócio

- A imagem vai para o S3 (`temas/{colaboradorId}/{blocoId}.{ext}`) e é devolvida
  como **URL presignada** (expira ~2h).
- Datas de **marco** são guardadas **ao meio-dia** para conversões de fuso não
  virarem o dia.
- `ordem` é calculada como `MAX(ordem)+1` por (colaborador, tema).
- A rota de imagem fica em `/blocos-imagem` (fora do grupo `/blocos`) para não
  colidir com `/blocos/{blocoId}` no roteador do Gin.

## Dependências

`NovoUseCase(repo, armazenamento, colaboradorUC)` — repositório + S3 + UseCase de
colaborador (validação). Montado em `cmd/api/rotas.go` após `colaborador`.
