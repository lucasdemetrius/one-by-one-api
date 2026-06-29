# Módulo `registroonebyone`

> Representa o **formulário preenchido durante uma reunião 1:1** (one-on-one) entre gestor e liderado: cada vez que uma reunião é "aberta", nasce um registro que guarda qual template foi usado naquele momento.

## O que faz

Este módulo é responsável por **abrir, consultar e excluir registros de reuniões 1:1**. Pense em um registro como uma "instância" preenchida de uma reunião: ele liga uma reunião (`onebyone`) ao template (roteiro de perguntas/blocos) que deve ser usado naquele encontro.

O detalhe mais importante é que **o cliente nunca informa qual template usar**. Ao criar o registro, o próprio módulo pergunta ao módulo `onebyone` qual template aplicar, seguindo uma regra de herança (colaborador → equipe → organização → padrão do líder). Assim que o template é resolvido, o registro é persistido no banco.

Se você vem de C#/WebForms, a organização aqui lembra uma arquitetura em camadas (DAL / BLL / UI): o `repository.go` é o acesso a dados (como um `Repository`/ADO.NET), o `usecase.go` é a regra de negócio (como uma classe de serviço/BLL) e o `controller.go` é a camada web (como um Controller MVC). As `interface` em Go funcionam como contratos (parecido com `interface` em C#), e a "injeção de dependência" é feita manualmente passando as dependências no construtor.

## Arquivos

| Arquivo | Camada | Responsabilidade |
| --- | --- | --- |
| `entity.go` | Entidade / Modelo | Define a struct `RegistroOneByOne`, que espelha a tabela `tb_registros_onebyone` no MySQL. |
| `dto.go` | DTO (entrada/saída) | Define os objetos de transporte da API: `CriarRegistroOneByOneDTO` (entrada) e `RegistroOneByOneRespostaDTO` (saída). |
| `mapper.go` | Conversão | Converte a entidade do banco para o DTO de resposta (`ParaRespostaDTO` e `ParaListaRespostaDTO`), isolando o modelo de banco da camada HTTP. |
| `repository.go` | Acesso a dados (DAL) | Interface `Repositorio` e implementação MySQL (`repositorioMySQL`) com todo o SQL contra `tb_registros_onebyone`. |
| `usecase.go` | Regra de negócio (BLL) | Interface `UseCase` e implementação `useCaseImpl` com as regras: resolução de template, criação, busca, listagem e exclusão lógica. |
| `controller.go` | Web / HTTP | Struct `Controller`, registro de rotas (`RegistrarRotas`) e os handlers Gin que recebem as requisições HTTP. |

## Entidade e tabela no banco

A struct **`RegistroOneByOne`** (em `entity.go`) mapeia diretamente a tabela **`tb_registros_onebyone`** no MySQL (nome confirmado pelos comandos SQL em `repository.go`). As tags `db:"..."` indicam o nome da coluna correspondente no banco.

| Campo (Go) | Coluna (MySQL) | Tipo Go | Significado |
| --- | --- | --- | --- |
| `ID` | `id` | `string` | Identificador único do registro, no formato UUID v4. |
| `OneByOneID` | `oneaone_id` | `string` | UUID da reunião 1:1 à qual este registro pertence. |
| `TemplateID` | `template_id` | `string` | UUID do template (roteiro) usado para estruturar este registro. Preenchido automaticamente pela regra de herança. |
| `CriadoEm` | `criado_em` | `time.Time` | Data/hora de criação do registro. |
| `AlteradoEm` | `alterado_em` | `*time.Time` | Data/hora da última modificação. É um ponteiro (`*time.Time`) para permitir `NULL`: vale `nil` se nunca foi alterado. |
| `DeletadoEm` | `deletado_em` | `*time.Time` | Data/hora da exclusão lógica (soft delete). `nil` significa que o registro está **ativo**. |
| `DeletadoPor` | `deletado_por` | `*string` | ID do usuário que realizou a exclusão lógica. `nil` enquanto o registro estiver ativo. |

Observação importante para quem vem do .NET: em Go, tipos como `string` e `time.Time` **não aceitam `null`** por padrão. Por isso os campos que podem ser nulos no banco (`alterado_em`, `deletado_em`, `deletado_por`) são declarados como **ponteiros** (`*time.Time`, `*string`) — o ponteiro `nil` é o equivalente ao `null`/`Nullable<T>` do C#.

## Endpoints

Todas as rotas ficam sob o prefixo global **`/api/v1`** (definido em `cmd/api/rotas.go`). Todos os endpoints deste módulo passam pelo `authMiddleware`, ou seja, **exigem token JWT** (`Authorization: Bearer <token>`, documentado como `BearerAuth` no Swagger).

| Método | Rota | Descrição | Autenticação |
| --- | --- | --- | --- |
| `POST` | `/api/v1/registros-onebyone` | Abre um novo registro para uma reunião. O template é resolvido automaticamente pela regra de herança. | JWT obrigatório |
| `GET` | `/api/v1/registros-onebyone/:id` | Busca um registro **ativo** pelo UUID. | JWT obrigatório |
| `DELETE` | `/api/v1/registros-onebyone/:id` | Exclusão lógica (soft delete) do registro pelo UUID. | JWT obrigatório |
| `GET` | `/api/v1/onebyone/:id/registros` | Lista todos os registros ativos de uma reunião, do mais recente ao mais antigo. | JWT obrigatório |

Detalhes que ajudam a ler o código:

- As três primeiras rotas são registradas no grupo `/registros-onebyone` dentro de `RegistrarRotas` (controller.go), e a quarta é uma **rota aninhada** sob o grupo `/onebyone/:id/registros`.
- Na rota aninhada, o parâmetro de caminho é lido como `ctx.Param("id")` no handler `ListarPorOneByOne` (esse `:id` representa o UUID da reunião). Nos comentários `// @Router` do Swagger ele aparece documentado como `{oneaoneId}`, mas o nome real do parâmetro na rota Gin é `:id`.
- No `DELETE`, o ID do usuário que está excluindo é obtido do contexto da requisição via `middleware.ChaveUsuarioID` (preenchido pelo JWT) e gravado em `deletado_por`.

## DTOs

Os DTOs estão em `dto.go`. As tags `binding:"required"` são as validações automáticas do Gin (equivalente, em espírito, a `[Required]` do Data Annotations no .NET): se o campo obrigatório faltar, o `ShouldBindJSON` devolve erro e a API responde **400 Bad Request**.

### Entrada — `CriarRegistroOneByOneDTO`

| Campo (Go) | JSON | Tipo | Validação | Observação |
| --- | --- | --- | --- | --- |
| `OneByOneID` | `onebyone_id` | `string` | `required` (obrigatório) | UUID da reunião 1:1 a ser registrada. **Não existe campo de template aqui** — ele é resolvido automaticamente. |

### Saída — `RegistroOneByOneRespostaDTO`

| Campo (Go) | JSON | Tipo | Observação |
| --- | --- | --- | --- |
| `ID` | `id` | `string` | Identificador único do registro. |
| `OneByOneID` | `onebyone_id` | `string` | UUID da reunião 1:1. |
| `TemplateID` | `template_id` | `string` | UUID do template que foi aplicado automaticamente pela herança. |
| `CriadoEm` | `criado_em` | `time.Time` | Data/hora de criação. |
| `AlteradoEm` | `alterado_em` | `*time.Time` | Data/hora da última modificação (`null` no JSON se nunca alterado). |

Repare que o DTO de saída **não expõe** os campos `deletado_em` e `deletado_por` da entidade — eles existem apenas no banco/entidade, e o `mapper.go` decide o que vai para a API.

## Regras de negócio

As regras estão em `usecase.go` (`useCaseImpl`). Pontos importantes:

- **Resolução automática de template (herança):** ao criar (`Criar`), o módulo **não** aceita um template do cliente. Ele chama `onebyoneUC.ResolverTemplate(dto.OneByOneID)`, que determina o template seguindo a prioridade **colaborador → equipe → organização → padrão do líder** (implementada via `COALESCE` no repositório do módulo `onebyone`). Se nenhum template puder ser resolvido, a criação falha com `"não foi possível abrir a reunião"`.
- **Geração de identidade no servidor:** o `ID` do novo registro é gerado pela aplicação com `uuid.New().String()` (UUID v4), e o `CriadoEm` é preenchido com `time.Now()`. O cliente não controla nem o ID nem a data.
- **Persistência e releitura:** após o `INSERT`, o repositório (`Criar` em repository.go) chama `BuscarPorId` para devolver o registro completo já persistido (garantindo que a resposta reflita o estado real no banco).
- **Consultas só retornam registros ativos:** tanto `BuscarPorId` quanto `ListarPorOneByOne` filtram por `deletado_em IS NULL` no SQL. Registros com soft delete são tratados como inexistentes (resultam em **404**).
- **Ordenação na listagem:** `ListarPorOneByOne` retorna os registros ordenados por `criado_em DESC` (do mais recente para o mais antigo).
- **Exclusão lógica (soft delete):** o `Deletar` do UseCase primeiro confirma que o registro existe e está ativo (chama `BuscarPorId`); se não existir, retorna `"registro não encontrado"`. Em seguida delega a `DeletarSoft`, que faz um `UPDATE` preenchendo `deletado_em = agora` e `deletado_por = <usuário>`, **sem remover a linha fisicamente**. Se o `UPDATE` não afetar nenhuma linha (registro inexistente ou já deletado), retorna `"registro não encontrado ou já deletado"`.
- **Não há hash de senha, nem upload S3, nem template informado manualmente** neste módulo — a única "inteligência" é a delegação da resolução de template ao módulo `onebyone`.

## Dependências

A montagem (wiring) acontece em `cmd/api/rotas.go`, na ordem repositório → usecase → controller:

- **`NovoRepositorio(db *sqlx.DB) Repositorio`** — recebe o pool de conexões MySQL (`*sqlx.DB`, da biblioteca `jmoiron/sqlx`). É a única dependência da camada de dados, pois todo o SQL é executado contra `tb_registros_onebyone`.
- **`NovoUseCase(repo Repositorio, onebyoneUC onebyone.UseCase) UseCase`** — recebe duas dependências:
  - `repo Repositorio` — para ler/gravar registros no banco.
  - `onebyoneUC onebyone.UseCase` — o UseCase do módulo `onebyone`, usado **exclusivamente** para chamar `ResolverTemplate` e descobrir qual template aplicar na criação. É essa dependência que implementa a regra de herança; sem ela, o módulo não saberia qual template usar.
- **`NovoController(uc UseCase) *Controller`** — recebe apenas o `UseCase`. O controller não acessa banco diretamente; ele apenas valida a entrada HTTP, chama o UseCase e formata a resposta usando o pacote `pkg/response`.

Como tudo é recebido via **interface** (`Repositorio`, `UseCase`, `onebyone.UseCase`), cada camada depende de um contrato e não de uma implementação concreta — o que facilita testes (é possível injetar um mock) e troca de implementação, de forma parecida com programar contra interfaces no .NET.
