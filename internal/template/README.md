# Módulo `template`

> Representa um modelo de formulário que o líder cria para padronizar como serão estruturadas as reuniões 1:1 (one-on-one) com seus liderados.

## O que faz

Este módulo gerencia os **templates** do líder. Um template é, basicamente, um "molde" de formulário com um nome (ex.: "Reunião Mensal", "Feedback Trimestral"). Cada template pertence a um líder (o usuário autenticado) e serve como base para organizar os registros de one-on-one. O módulo oferece o CRUD completo (criar, buscar, listar, atualizar e excluir), sempre respeitando que **só o dono do template pode alterá-lo ou excluí-lo**, e a exclusão é lógica (o registro continua no banco, apenas marcado como deletado).

Se você vem do .NET Framework / WebForms, pense neste módulo como um conjunto de classes bem separadas por responsabilidade: o `Controller` é como o code-behind/Controller que recebe o request HTTP, o `UseCase` é a camada de regras de negócio (Service), o `Repository` é o acesso a dados (como um DAL/Repository com ADO.NET), e os DTOs/Entity separam o que trafega na API do que está no banco.

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---|---|---|
| `entity.go` | Domínio / Entidade | Define a struct `Template`, que espelha a tabela `tb_template` do MySQL. |
| `dto.go` | Apresentação (HTTP) | Define os DTOs de entrada e saída, isolando o modelo do banco da API. |
| `mapper.go` | Conversão | Converte a entidade `Template` para o DTO de resposta (individual e em lista). |
| `repository.go` | Acesso a dados | Interface `Repositorio` e implementação MySQL com as queries SQL da tabela. |
| `usecase.go` | Regras de negócio | Interface `UseCase` e implementação com validações, geração de UUID e checagem de dono. |
| `controller.go` | Apresentação (HTTP) | Registra as rotas Gin, lê o request, delega ao `UseCase` e devolve a resposta. |

## Entidade e tabela no banco

A struct `Template` (em `entity.go`) mapeia a tabela **`tb_template`** no MySQL. As tags `db:"..."` indicam o nome da coluna correspondente.

| Campo (Go) | Coluna (MySQL) | Tipo Go | Significado |
|---|---|---|---|
| `ID` | `id` | `string` | Identificador único no formato UUID v4. |
| `UsuarioID` | `usuario_id` | `string` | UUID do líder proprietário do template. |
| `Nome` | `nome` | `string` | Nome descritivo do template (ex.: "Reunião Mensal"). |
| `CriadoEm` | `criado_em` | `time.Time` | Data/hora de criação do registro. |
| `AlteradoEm` | `alterado_em` | `*time.Time` | Data/hora da última modificação. É um ponteiro porque pode ser `nil` (nulo no banco) quando o registro nunca foi alterado. |
| `DeletadoEm` | `deletado_em` | `*time.Time` | Data/hora da exclusão lógica. `nil` significa que o registro está **ativo**. |
| `DeletadoPor` | `deletado_por` | `*string` | UUID do usuário que realizou a exclusão lógica. `nil` enquanto o registro está ativo. |

Observação para quem vem do C#: os campos com `*` (ponteiro) são o equivalente aos tipos anuláveis (`DateTime?`, `string?`). Um ponteiro `nil` em Go corresponde a um `null` no banco.

## Endpoints

Todas as rotas ficam sob o prefixo **`/api/v1`** e estão dentro do grupo `/templates`. **Todas exigem autenticação JWT** (o grupo aplica `authMiddleware` via `templates.Use(authMiddleware)`), portanto é obrigatório enviar o header `Authorization: Bearer <token>`.

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| POST | `/api/v1/templates` | Cria um novo template vinculado ao líder autenticado. | JWT obrigatório |
| GET | `/api/v1/templates` | Lista todos os templates ativos do líder autenticado, do mais antigo ao mais novo. | JWT obrigatório |
| GET | `/api/v1/templates/{id}` | Busca um template ativo pelo UUID. | JWT obrigatório |
| PUT | `/api/v1/templates/{id}` | Atualiza (renomeia) um template. Apenas o dono pode alterar. | JWT obrigatório |
| DELETE | `/api/v1/templates/{id}` | Exclusão lógica (soft delete) do template. Apenas o dono pode excluir. | JWT obrigatório |

O `id` do usuário não vem no corpo da requisição: ele é extraído do token JWT pelo middleware e lido no controller via `ctx.Get(middleware.ChaveUsuarioID)`.

## DTOs

Os DTOs ficam em `dto.go`. As tags `binding:"..."` são as validações automáticas do Gin (equivalente aos Data Annotations / validações do `ModelState` no ASP.NET). Se a validação falhar, o controller responde `400 - dados inválidos`.

### Entrada

**`CriarTemplateDTO`** (corpo do POST):

| Campo | Tipo | JSON | Validação (`binding`) |
|---|---|---|---|
| `Nome` | `string` | `nome` | `required`, `min=2`, `max=100` (obrigatório, de 2 a 100 caracteres). |

**`AtualizarTemplateDTO`** (corpo do PUT):

| Campo | Tipo | JSON | Validação (`binding`) |
|---|---|---|---|
| `Nome` | `string` | `nome` | `required`, `min=2`, `max=100` (obrigatório na atualização, de 2 a 100 caracteres). |

### Saída

**`TemplateRespostaDTO`** (retornado por todos os endpoints que devolvem um template):

| Campo | Tipo | JSON | Significado |
|---|---|---|---|
| `ID` | `string` | `id` | UUID do template. |
| `UsuarioID` | `string` | `usuario_id` | UUID do líder proprietário. |
| `Nome` | `string` | `nome` | Nome do template. |
| `CriadoEm` | `time.Time` | `criado_em` | Data/hora de criação. |
| `AlteradoEm` | `*time.Time` | `alterado_em` | Data/hora da última alteração (`null` se nunca alterado). |

Note que o `TemplateRespostaDTO` **não** expõe os campos `deletado_em` e `deletado_por` da entidade — eles são detalhes internos do soft delete e ficam fora da resposta da API.

## Regras de negócio

Implementadas em `usecase.go` (e algumas garantidas pelo `repository.go`):

- **Geração do UUID no servidor:** ao criar, o `UseCase` gera o `ID` com `uuid.New().String()`. O cliente não envia o id.
- **Vínculo automático com o líder:** o `UsuarioID` do novo template é o usuário autenticado (vindo do JWT), não um valor enviado no corpo.
- **`CriadoEm` definido no servidor:** o `UseCase` preenche `CriadoEm` com `time.Now()` na criação.
- **Validação de propriedade na atualização:** antes de renomear, o `UseCase` busca o template e compara `atual.UsuarioID` com o usuário autenticado. Se forem diferentes, retorna o erro `"você não tem permissão para alterar este template"`, que o controller traduz em **403 (Proibido)**.
- **Validação de propriedade na exclusão:** mesma checagem de dono antes de deletar; se falhar, retorna `"você não tem permissão para excluir este template"` → **403 (Proibido)**.
- **`AlteradoEm` atualizado automaticamente:** o método `Atualizar` do repositório define `alterado_em = time.Now()` a cada update, sem depender do cliente.
- **Soft delete (exclusão lógica):** a exclusão não remove a linha. O repositório preenche `deletado_em` e `deletado_por` via UPDATE. Se nenhuma linha for afetada (registro inexistente ou já deletado), retorna `"template não encontrado ou já deletado"`.
- **Filtro de registros ativos:** todas as queries de leitura e escrita usam `WHERE ... deletado_em IS NULL`, então templates excluídos logicamente nunca aparecem em buscas, listagens, atualizações ou novas exclusões.
- **Ordenação intencional na listagem:** `ListarPorUsuario` ordena por `criado_em ASC` (do mais antigo ao mais novo). Isso é proposital: o **primeiro template criado** é considerado o padrão do líder, usado pela regra de **herança de template** do módulo `onebyone`.
- **Mapeamento de erros para status HTTP (depende do endpoint):** o tratamento de erro é feito por endpoint no `controller.go`, e não há um mapeamento global uniforme. Em detalhe:
  - **POST `/templates` (Criar):** qualquer erro do `UseCase` vira **500** (`ErroInterno`); falha de validação do corpo vira **400** (`ErroRequisicao`).
  - **GET `/templates/{id}` (BuscarPorId):** qualquer erro vira **404** (`ErroNaoEncontrado`) — inclusive um eventual erro interno de banco.
  - **GET `/templates` (Listar):** qualquer erro vira **500** (`ErroInterno`).
  - **PUT `/templates/{id}` (Atualizar):** erro de permissão (`"você não tem permissão para alterar este template"`) vira **403**; **qualquer outro erro — inclusive "template não encontrado" — vira 500** (não 404); falha de validação do corpo vira **400**.
  - **DELETE `/templates/{id}` (Deletar):** erro de permissão (`"você não tem permissão para excluir este template"`) vira **403**; **qualquer outro erro vira 404** (`ErroNaoEncontrado`), inclusive um eventual erro interno.

## Dependências

O módulo segue injeção de dependência por interface (cada camada recebe a de baixo no construtor). Isso facilita testes e mantém o acoplamento baixo.

| Construtor | Recebe | Por quê |
|---|---|---|
| `NovoRepositorio(db *sqlx.DB)` | O pool de conexões MySQL (`*sqlx.DB` da lib `jmoiron/sqlx`). | É a única camada que conhece o banco; usa `db` para executar as queries na `tb_template`. |
| `NovoUseCase(repo Repositorio)` | A interface `Repositorio`. | As regras de negócio delegam a persistência ao repositório sem conhecer detalhes de SQL/MySQL. |
| `NovoController(uc UseCase)` | A interface `UseCase`. | O controller só sabe lidar com HTTP; toda lógica é delegada ao `UseCase`. Nunca acessa o banco diretamente. |

Além disso, o `RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc)` recebe o grupo de rotas do Gin e o middleware de autenticação JWT, aplicado a todas as rotas do módulo. O controller também usa `pkg/middleware` (para a chave `ChaveUsuarioID` que identifica o usuário do token) e `pkg/response` (para padronizar as respostas de sucesso e erro).
