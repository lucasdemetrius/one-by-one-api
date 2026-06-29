# Módulo `onebyone`

> Representa as reuniões 1:1 (one-on-one) agendadas entre um líder (gestor) e um colaborador (liderado) dentro do produto OneByOne.

## O que faz

Este módulo cuida do **agendamento e gerenciamento das reuniões one-on-one**. Por meio dele um líder cria uma reunião com um colaborador, informando a organização, a equipe, a recorrência (nenhuma, mensal ou quinzenal) e a data planejada. Também permite listar as reuniões do próprio líder, buscar uma reunião pelo identificador, atualizar campos (status, recorrência, data) e excluir a reunião de forma lógica (sem apagar o registro do banco).

Além disso, o módulo concentra a regra de **herança de template**: ao abrir uma reunião para preenchimento, o sistema decide qual formulário (template) usar seguindo uma ordem de prioridade — colaborador, equipe, organização e, por fim, o template padrão do líder.

> Observação para quem vem do .NET: a arquitetura aqui é em camadas separadas por responsabilidade, parecida com uma divisão `Controller → Service → Repository`. Cada camada fala com a próxima através de uma **interface** (um contrato), o que facilita a substituição e os testes. O `Controller` nunca acessa o banco diretamente.

## Arquivos

| Arquivo | Camada | Responsabilidade |
| --- | --- | --- |
| `entity.go` | Entidade (modelo de banco) | Define a struct `OneByOne`, que espelha a tabela `tb_onebyone`. |
| `dto.go` | DTO (contrato HTTP) | Define os objetos de entrada e saída da API, isolando o banco da camada web. |
| `mapper.go` | Conversão | Converte a entidade `OneByOne` no DTO de resposta (`ParaRespostaDTO` e `ParaListaRespostaDTO`). |
| `repository.go` | Repositório (acesso a dados) | Interface `Repositorio` e implementação MySQL com as queries SQL (incluindo o `COALESCE` da herança de template). |
| `usecase.go` | Caso de uso (regras de negócio) | Interface `UseCase` e implementação com validações, geração de UUID, valores padrão e a regra de template. |
| `controller.go` | Controlador HTTP | Registra as rotas, lê o corpo da requisição, valida e delega ao `UseCase`. |

## Entidade e tabela no banco

A struct `OneByOne` (em `entity.go`) mapeia a tabela **`tb_onebyone`** no MySQL. Cada campo usa a tag `db:"..."` para indicar o nome da coluna correspondente.

| Campo (Go) | Coluna (`db`) | Tipo Go | Significado |
| --- | --- | --- | --- |
| `ID` | `id` | `string` | Identificador único da reunião, no formato UUID v4. |
| `UsuarioID` | `usuario_id` | `string` | UUID do líder que agendou a reunião. |
| `OrganizacaoID` | `organizacao_id` | `string` | UUID da organização que é o contexto da reunião. |
| `EquipeID` | `equipe_id` | `string` | UUID da equipe que é o contexto da reunião. |
| `ColaborID` | `colabor_id` | `string` | UUID do colaborador (liderado) participante. |
| `Recorrencia` | `recorrencia` | `string` | Frequência da reunião: `NENHUMA`, `MENSAL` ou `QUINZENAL`. |
| `Status` | `status` | `string` | Estado atual: `AGENDADO`, `REALIZADO` ou `PENDENTE`. |
| `DataAgendada` | `data_agendada` | `time.Time` | Data planejada para a realização da reunião. |
| `CriadoEm` | `criado_em` | `time.Time` | Data e hora de criação do registro. |
| `AlteradoEm` | `alterado_em` | `*time.Time` | Data e hora da última modificação. É um ponteiro: `nil` quando nunca foi alterado. |
| `DeletadoEm` | `deletado_em` | `*time.Time` | Data e hora da exclusão lógica. `nil` significa que o registro está **ativo**. |
| `DeletadoPor` | `deletado_por` | `*string` | UUID do usuário que realizou a exclusão lógica. `nil` quando não foi excluído. |

> Sobre os ponteiros (`*time.Time`, `*string`): em Go, um ponteiro pode ser `nil` (nulo). É assim que o código representa colunas que podem ser `NULL` no banco — equivalente a um `DateTime?` ou `string` anulável no C#.

Outras tabelas referenciadas nas queries do repositório (na regra de herança de template): `tb_template`, `tb_colaboradores`, `tb_equipes`, `tb_organizacoes`.

## Endpoints

Todas as rotas ficam sob o grupo `/onebyone`, e esse grupo é montado dentro do prefixo global `/api/v1` (definido em `cmd/api/rotas.go`). **Todas exigem JWT**: o grupo aplica o `authMiddleware` a todas as rotas, então é preciso enviar o cabeçalho `Authorization: Bearer <token>`.

| Método | Rota | Descrição | Autenticação |
| --- | --- | --- | --- |
| `POST` | `/api/v1/onebyone` | Agenda uma nova reunião one-on-one entre líder e colaborador. | JWT obrigatório |
| `POST` | `/api/v1/onebyone/encerrar` | Registra um 1:1 como **REALIZADO** (livro-razão). Corpo: `{ colabor_id }`. Idempotente por dia. Posse via colaborador (Cadeia B). | JWT + **ApenasLider** |
| `GET` | `/api/v1/onebyone` | Lista todas as reuniões ativas do líder autenticado, da mais recente à mais antiga. | JWT obrigatório |
| `GET` | `/api/v1/onebyone/{id}` | Retorna os dados de uma reunião ativa pelo UUID. | JWT obrigatório |
| `PUT` | `/api/v1/onebyone/{id}` | Atualiza status, recorrência ou data agendada de uma reunião pelo UUID. | JWT obrigatório |
| `DELETE` | `/api/v1/onebyone/{id}` | Realiza a exclusão lógica (soft delete) de uma reunião pelo UUID. | JWT obrigatório |

Detalhes do comportamento:

- **`POST /encerrar`** é a forma como o ritual de "Encerrar 1:1" (frontend) registra que o
  encontro aconteceu. Como o 1:1 ao vivo é por colaborador (não há reunião pré-criada), o
  endpoint **cria** uma linha já com `status=REALIZADO`, `data_agendada=hoje` e
  `realizado_em=NOW()`. É **idempotente por dia** (`BuscarRealizadoNoDia`) e resolve
  organização/equipe do liderado via `colaboradorUseCase`. Alimenta o módulo
  [`saude1a1`](../saude1a1/README.md) (cadência e streak). A coluna `realizado_em` veio na
  migration **015**.
- No `POST` e no `GET ""` (listar), o `usuario_id` do líder **não vem no corpo** — é extraído do token JWT (`middleware.ChaveUsuarioID`).
- No `DELETE`, o usuário do token é gravado como `deletado_por`.
- O `GET ""` retorna apenas as reuniões do **próprio** líder autenticado.

## DTOs

Os DTOs (`Data Transfer Objects`) ficam em `dto.go`. As validações de entrada usam as tags `binding:"..."` do Gin, que rejeitam a requisição com erro 400 quando uma regra não é atendida.

### Entrada — `CriarOneByOneDTO` (corpo do `POST`)

| Campo JSON | Tipo | Validação (`binding`) | Descrição |
| --- | --- | --- | --- |
| `organizacao_id` | `string` | `required` | UUID da organização. Obrigatório. |
| `equipe_id` | `string` | `required` | UUID da equipe. Obrigatório. |
| `colabor_id` | `string` | `required` | UUID do colaborador participante. Obrigatório. |
| `recorrencia` | `string` | `omitempty,oneof=NENHUMA MENSAL QUINZENAL` | Frequência. Opcional; se vier, deve ser um dos três valores. Padrão aplicado pelo caso de uso: `NENHUMA`. |
| `data_agendada` | `string` | `required` | Data planejada no formato `YYYY-MM-DD`. Obrigatório. |

### Entrada — `AtualizarOneByOneDTO` (corpo do `PUT`)

Todos os campos são opcionais; apenas os informados são alterados.

| Campo JSON | Tipo | Validação (`binding`) | Descrição |
| --- | --- | --- | --- |
| `status` | `string` | `omitempty,oneof=AGENDADO REALIZADO PENDENTE` | Novo status. Opcional. |
| `recorrencia` | `string` | `omitempty,oneof=NENHUMA MENSAL QUINZENAL` | Nova recorrência. Opcional. |
| `data_agendada` | `string` | `omitempty` | Nova data no formato `YYYY-MM-DD`. Opcional. |

### Saída — `OneByOneRespostaDTO` (retornado pela API)

| Campo JSON | Tipo | Descrição |
| --- | --- | --- |
| `id` | `string` | Identificador único da reunião. |
| `usuario_id` | `string` | UUID do líder que agendou. |
| `organizacao_id` | `string` | UUID da organização. |
| `equipe_id` | `string` | UUID da equipe. |
| `colabor_id` | `string` | UUID do colaborador. |
| `recorrencia` | `string` | Frequência da reunião. |
| `status` | `string` | Estado atual da reunião. |
| `data_agendada` | `time.Time` | Data planejada. |
| `criado_em` | `time.Time` | Data e hora de criação. |
| `alterado_em` | `*time.Time` | Data e hora da última modificação (pode ser nulo). |

> Repare que o DTO de saída **não** expõe os campos de exclusão lógica (`deletado_em`, `deletado_por`). Eles existem na entidade, mas são detalhe interno e ficam fora da resposta.

## Regras de negócio

Implementadas em `usecase.go`:

- **Geração do ID**: ao criar, o UUID v4 é gerado pelo servidor (`uuid.New().String()`); o cliente não envia o `id`.
- **Líder vem do token**: o `usuario_id` da reunião é o usuário autenticado (recebido do controller), nunca um valor do corpo da requisição.
- **Status inicial fixo**: toda reunião criada nasce com `Status = "AGENDADO"`.
- **Recorrência padrão**: se `recorrencia` não for informada no `POST`, o caso de uso assume `"NENHUMA"`.
- **Validação de data**: a `data_agendada` é convertida de string para `time.Time` usando o layout `2006-01-02` (ou seja, `YYYY-MM-DD`). Se o formato for inválido, retorna erro orientando o uso de `YYYY-MM-DD` — isso vale tanto na criação quanto na atualização.
- **Atualização parcial**: no `Atualizar`, primeiro a reunião é buscada; depois, apenas os campos não vazios do DTO (`status`, `recorrencia`, `data_agendada`) sobrescrevem os valores atuais. O repositório também grava `alterado_em` com a data/hora atual.
- **Exclusão lógica (soft delete)**: o `Deletar` confirma que a reunião existe e delega ao repositório, que preenche `deletado_em` e `deletado_por` em vez de remover a linha. Se nenhuma linha for afetada (não existe ou já estava excluída), retorna erro.
- **Filtro de registros ativos**: todas as queries de leitura e escrita usam `deletado_em IS NULL`, de forma que reuniões excluídas logicamente nunca aparecem nas buscas, listagens ou atualizações.
- **Herança de template** (`ResolverTemplate` → `Repositorio.ResolverTemplateID`): ao abrir uma reunião, o template do formulário é resolvido em uma única query SQL com `COALESCE`, seguindo a ordem de prioridade do mais específico ao mais genérico:
  1. `tb_colaboradores.template_id` — template exclusivo do colaborador (prioridade máxima);
  2. `tb_equipes.template_id` — template da equipe;
  3. `tb_organizacoes.template_id` — template da organização;
  4. **Template padrão do líder** — o template mais antigo criado por ele (`ORDER BY criado_em ASC LIMIT 1`), usado como último recurso.

  Se nenhum template for encontrado em nenhum nível, retorna erro orientando o líder a configurar pelo menos um template antes de abrir reuniões.

> Não há hash de senha neste módulo — essa responsabilidade pertence a outros módulos (ex.: usuário).

## Dependências

A montagem segue a injeção de dependências: cada construtor recebe a camada abaixo dela já pronta. Em `cmd/api/rotas.go` o encadeamento é `NovoRepositorio(db)` → `NovoUseCase(repo)` → `NovoController(uc)`.

| Construtor | Recebe | Por quê |
| --- | --- | --- |
| `NovoRepositorio(db *sqlx.DB)` | O pool de conexões MySQL (`*sqlx.DB`). | É a única camada que fala com o banco; precisa da conexão para executar as queries. |
| `NovoUseCase(repo Repositorio)` | O repositório (via interface `Repositorio`). | As regras de negócio delegam a persistência ao repositório, sem conhecer detalhes de SQL. |
| `NovoController(uc UseCase)` | O caso de uso (via interface `UseCase`). | O controller apenas valida a requisição HTTP e chama o caso de uso; nunca acessa o banco. |

Além disso, `RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc)` recebe o grupo de rotas (`/api/v1`) e o middleware de autenticação JWT, que é aplicado a todas as rotas do módulo.

> Diferente de outros módulos do projeto (como `usuario`, `organizacao`, `equipe` e `colaborador`, que recebem o serviço de S3 e/ou a configuração `cfg`), o `onebyone` **não depende de S3 nem de `cfg`** — ele só precisa do banco. O `UseCase` do `onebyone`, por sua vez, é reaproveitado pelo módulo `registroonebyone` para resolver o template pela regra de herança.
