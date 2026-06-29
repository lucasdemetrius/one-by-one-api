# Módulo `templatebloco`

> Representa um **bloco** (campo de formulário) dentro de um template de reunião 1:1: cada bloco é uma pergunta/campo que o gestor preenche durante a conversa com o liderado.

## O que faz

Este módulo gerencia os blocos que compõem um template de formulário. Um template é o "molde" de uma reunião 1:1; cada bloco é um item desse molde (um campo de texto, uma imagem, uma lista ou um destaque). O módulo permite criar, buscar, listar (por template), atualizar e excluir blocos. Cada bloco tem um `tipo`, uma `posicao` (ordem em que aparece no formulário) e um `rotulo` (o título que o usuário vê).

Assim como nos seus projetos C#/.NET, há uma separação clara de camadas: o **Controller** trata HTTP (equivalente a um `ApiController`), o **UseCase** contém as regras de negócio (a sua camada de "Service") e o **Repository** fala com o banco (o seu "DAL"/repositório). Em Go, cada camada é exposta por uma **interface**, e a implementação concreta é injetada no construtor — é injeção de dependência manual, sem container.

## Arquivos

| Arquivo | Camada | Responsabilidade |
| --- | --- | --- |
| `controller.go` | Apresentação (HTTP) | Registra as rotas, lê o JSON da requisição, valida o binding e delega ao UseCase. Nunca acessa o banco direto. |
| `dto.go` | Contrato (HTTP) | Define os objetos de entrada (`Criar`/`Atualizar`) e de saída (`Resposta`), desacoplando o banco da API. |
| `entity.go` | Domínio | Define a struct `TemplateBloco`, espelho da tabela `tb_template_blocos`. |
| `mapper.go` | Conversão | Converte a entidade (modelo de banco) para o DTO de resposta (modelo da API). |
| `repository.go` | Dados (DAL) | Interface `Repositorio` + implementação MySQL com as queries SQL sobre `tb_template_blocos`. |
| `usecase.go` | Negócio (Service) | Interface `UseCase` + implementação com as regras: gera UUID, atualização parcial, valida existência antes de deletar. |

## Entidade e tabela no banco

A struct `TemplateBloco` (em `entity.go`) mapeia diretamente a tabela **`tb_template_blocos`** no MySQL (nome confirmado nas queries do `repository.go`). As tags `db:"..."` indicam o nome da coluna correspondente — papel parecido com os atributos de mapeamento de um ORM no .NET.

| Campo (Go) | Coluna (MySQL) | Tipo Go | Significado |
| --- | --- | --- | --- |
| `ID` | `id` | `string` | Identificador único do bloco, no formato UUID v4. |
| `TemplateID` | `template_id` | `string` | UUID do template ao qual o bloco pertence. |
| `Tipo` | `tipo` | `string` | Tipo do conteúdo do bloco: `TEXT`, `IMAGE`, `LIST` ou `HIGHLIGHT`. |
| `Posicao` | `posicao` | `int` | Ordem de exibição do bloco no template (menor = aparece primeiro). |
| `Rotulo` | `rotulo` | `string` | Label/título exibido ao usuário ao preencher o campo. |
| `CriadoEm` | `criado_em` | `time.Time` | Data e hora de criação do registro. |
| `AlteradoEm` | `alterado_em` | `*time.Time` | Data e hora da última alteração; `nil` (NULL) se nunca foi alterado. |
| `DeletadoEm` | `deletado_em` | `*time.Time` | Data e hora da exclusão lógica; `nil` (NULL) significa registro **ativo**. |
| `DeletadoPor` | `deletado_por` | `*string` | ID do usuário que fez a exclusão lógica; `nil` se não deletado. |

> Observação sobre Go: tipos como `*time.Time` e `*string` são **ponteiros**. Em Go, um ponteiro pode ser `nil`, então é assim que o código representa um valor NULL do banco — equivale a um `DateTime?` / `string` nulo no C#.

## Endpoints

Todas as rotas são registradas dentro do grupo global **`/api/v1`** (definido em `cmd/api/rotas.go`) e o grupo deste módulo aplica `authMiddleware` a **todas** elas. Portanto, **todos os endpoints exigem um token JWT** no header `Authorization: Bearer <token>`.

| Método | Rota | Descrição | Autenticação |
| --- | --- | --- | --- |
| `POST` | `/api/v1/template-blocos` | Cria um novo bloco dentro de um template existente. | JWT obrigatório |
| `GET` | `/api/v1/template-blocos/:id` | Busca um bloco ativo pelo UUID. | JWT obrigatório |
| `PUT` | `/api/v1/template-blocos/:id` | Atualiza parcialmente um bloco pelo UUID. | JWT obrigatório |
| `DELETE` | `/api/v1/template-blocos/:id` | Exclusão lógica (soft delete) de um bloco pelo UUID. | JWT obrigatório |
| `GET` | `/api/v1/templates/:id/blocos` | Lista todos os blocos ativos de um template, ordenados por `posicao` (crescente). | JWT obrigatório |

> Detalhe de implementação: a rota de listagem usa o prefixo aninhado `/templates/:id/blocos` (registrado em um grupo separado) propositalmente, para evitar conflito de wildcard com a rota `GET /templates/:id` de outro módulo. O parâmetro de caminho é lido como `ctx.Param("id")`, ou seja, o `:id` aqui é o **UUID do template**.

## DTOs

Os DTOs vivem em `dto.go`. As tags `binding:"..."` são as **validações automáticas** do Gin (validador `go-playground/validator`), executadas no `ShouldBindJSON`. São o equivalente aos `DataAnnotations` (`[Required]`, `[MaxLength]`) do ASP.NET — se a validação falhar, o Controller responde `400` antes de chegar ao UseCase.

### Entrada — `CriarTemplateBlocoDTO`

| Campo JSON | Tipo | Validação (`binding`) | Observação |
| --- | --- | --- | --- |
| `template_id` | `string` | `required` | UUID do template de destino. Obrigatório. |
| `tipo` | `string` | `required,oneof=TEXT IMAGE LIST HIGHLIGHT` | Deve ser exatamente um dos quatro valores aceitos. |
| `posicao` | `int` | `min=0` | Ordem de exibição; não pode ser negativa. |
| `rotulo` | `string` | `required,min=1,max=150` | Label do campo; entre 1 e 150 caracteres. |

### Entrada — `AtualizarTemplateBlocoDTO`

Atualização **parcial**: todos os campos são opcionais; apenas os informados são aplicados.

| Campo JSON | Tipo | Validação (`binding`) | Observação |
| --- | --- | --- | --- |
| `tipo` | `string` | `omitempty,oneof=TEXT IMAGE LIST HIGHLIGHT` | Se enviado, deve ser um dos quatro valores válidos. |
| `posicao` | `*int` (ponteiro) | `omitempty,min=0` | Ponteiro para distinguir "não enviado" (`nil`) de "enviado como 0". |
| `rotulo` | `string` | `omitempty,min=1,max=150` | Se enviado, entre 1 e 150 caracteres. |

> Por que `posicao` é `*int` (ponteiro) aqui e `int` no de criar? Porque `0` é um valor válido de posição. Se fosse `int` comum, o código não conseguiria diferenciar "o cliente quer posição 0" de "o cliente não mandou esse campo". Com ponteiro, `nil` = não mandou; um valor = mandou (mesmo que seja 0).

### Saída — `TemplateBlocoRespostaDTO`

| Campo JSON | Tipo | Observação |
| --- | --- | --- |
| `id` | `string` | UUID do bloco. |
| `template_id` | `string` | UUID do template ao qual o bloco pertence. |
| `tipo` | `string` | `TEXT`, `IMAGE`, `LIST` ou `HIGHLIGHT`. |
| `posicao` | `int` | Ordem de exibição. |
| `rotulo` | `string` | Label exibido ao usuário. |
| `criado_em` | `time.Time` | Data/hora de criação. |
| `alterado_em` | `*time.Time` | Data/hora da última alteração; `null` no JSON se nunca alterado. |

> Note que o DTO de resposta **não** expõe os campos de exclusão lógica (`deletado_em`, `deletado_por`): eles existem na entidade/banco, mas o `mapper.go` propositalmente não os copia para a saída da API.

## Regras de negócio

Regras implementadas em `usecase.go` (e apoiadas pelo `repository.go`):

- **Criação gera o UUID no servidor**: o `Criar` chama `uuid.New().String()` para o `ID` e define `CriadoEm = time.Now()`. O cliente não envia o ID; envia apenas `template_id`, `tipo`, `posicao` e `rotulo`.
- **Atualização é parcial e preserva o estado atual**: o `Atualizar` primeiro faz `BuscarPorId` para carregar o bloco existente; depois sobrescreve **apenas** os campos informados (`tipo` se não vazio, `rotulo` se não vazio, `posicao` se o ponteiro não for `nil`). Os demais campos permanecem como estavam.
- **`alterado_em` é definido automaticamente**: no repositório, o `Atualizar` força `alterado_em = time.Now()` em toda atualização — o cliente não controla esse campo.
- **Deletar valida existência antes de excluir**: o `Deletar` chama `BuscarPorId` primeiro; se o bloco não existe (ou já está deletado), retorna erro de "não encontrado" e não tenta excluir.
- **Soft delete (exclusão lógica)**: nada é removido fisicamente. O `DeletarSoft` apenas preenche `deletado_em` (com `time.Now()`) e `deletado_por` (com o ID do usuário autenticado). O `deletado_por` vem do contexto da requisição via `middleware.ChaveUsuarioID`, ou seja, é extraído do JWT — não é informado no body.
- **Todas as leituras ignoram registros deletados**: as queries de `BuscarPorId`, `ListarPorTemplate` e os `UPDATE` sempre incluem `WHERE ... deletado_em IS NULL`. Um bloco deletado some das consultas e não pode mais ser atualizado nem re-deletado.
- **Listagem sempre ordenada por `posicao`**: o `ListarPorTemplate` usa `ORDER BY posicao ASC`, garantindo que os blocos venham na ordem em que devem aparecer no formulário.
- **`DeletarSoft` confirma o efeito**: o repositório checa `RowsAffected()`; se nenhuma linha foi afetada (bloco inexistente ou já deletado), retorna o erro "bloco de template não encontrado ou já deletado".

> Este módulo **não** lida com hash de senha nem com upload para S3 — essas preocupações pertencem a outros módulos do projeto.

## Dependências

A injeção de dependência é manual e feita em cadeia no arquivo `cmd/api/rotas.go`, do mais baixo (banco) para o mais alto (HTTP):

| Construtor | Recebe | Para quê |
| --- | --- | --- |
| `NovoRepositorio(db *sqlx.DB)` | O pool de conexões MySQL (`*sqlx.DB`) | Executar as queries SQL sobre `tb_template_blocos`. É a única camada que conhece o banco. |
| `NovoUseCase(repo Repositorio)` | A **interface** `Repositorio` | Aplicar as regras de negócio sem saber qual é a implementação concreta de banco (facilita testes com mock). |
| `NovoController(uc UseCase)` | A **interface** `UseCase` | Tratar HTTP delegando toda a lógica ao UseCase. |
| `RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc)` | O grupo de rotas `/api/v1` e o middleware de autenticação JWT | Pendurar as rotas do módulo no roteador e proteger todas com JWT. |

Ordem real de inicialização (em `cmd/api/rotas.go`):

```
templateBlocoRepo       := templatebloco.NovoRepositorio(db)
templateBlocoUseCase    := templatebloco.NovoUseCase(templateBlocoRepo)
templateBlocoController := templatebloco.NovoController(templateBlocoUseCase)
templateBlocoController.RegistrarRotas(api, authMiddleware)
```

> Em Go, depender de **interfaces** (e não de classes concretas) é o padrão idiomático para inversão de dependência — semelhante a registrar `IServico` no DI do .NET, mas aqui o "wiring" é escrito à mão, sem framework de container.
