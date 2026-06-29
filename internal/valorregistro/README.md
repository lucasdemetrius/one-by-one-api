# Módulo `valorregistro`

> Guarda as respostas preenchidas em cada bloco do formulário de uma reunião 1:1 entre gestor e liderado — é o conteúdo efetivo que cada registro de one-on-one armazena.

## O que faz

Quando um gestor e seu liderado realizam uma reunião 1:1, o registro daquela conversa é montado a partir de um template, que por sua vez é dividido em blocos (campos de texto, listas, imagens etc.). Este módulo é responsável por **salvar a resposta de cada bloco**: o que foi escrito num campo de texto, os itens de uma lista, a referência de uma imagem e assim por diante.

Cada resposta é um "valor de registro" e fica ligada a dois identificadores: o **registro** (a reunião) e o **bloco** (o campo daquele template). O módulo permite criar, buscar, listar (todas as respostas de um registro), atualizar e excluir essas respostas.

Se você vem de C# / .NET Framework, pense neste módulo como uma camada bem separada: o `Controller` é o equivalente a um `ApiController`, o `UseCase` seria a sua camada de serviço (regras de negócio), o `Repository` é o acesso a dados (como um DAL/Repository com ADO.NET ou Dapper) e a `Entity` é a classe que mapeia a tabela. A injeção de dependência é feita "na mão", passando uma interface para o construtor de cada camada.

## Arquivos

| Arquivo | Camada | Responsabilidade |
| --- | --- | --- |
| `controller.go` | Apresentação (HTTP) | Define os endpoints, lê o corpo/parâmetros da requisição, valida o JSON e delega ao `UseCase`. Nunca acessa o banco diretamente. |
| `usecase.go` | Negócio | Concentra as regras: validação de campos obrigatórios, serialização do JSON, geração de UUID, atualização parcial e verificação antes de deletar. |
| `repository.go` | Dados | Interface e implementação MySQL. Todo SQL contra a tabela `tb_valores_registro` vive aqui. |
| `entity.go` | Domínio | Define a struct `ValorRegistro`, que espelha as colunas da tabela. |
| `dto.go` | Contratos | Define os DTOs de entrada e de saída usados na API (o que entra e o que sai em JSON). |
| `mapper.go` | Conversão | Converte a entidade `ValorRegistro` para o DTO de resposta (`ValorRegistroRespostaDTO`), isolando o modelo de banco da camada HTTP. |

## Entidade e tabela no banco

A struct `ValorRegistro` (em `entity.go`) mapeia diretamente a tabela **`tb_valores_registro`** no MySQL. As tags `db:"..."` indicam o nome da coluna correspondente (usadas pela biblioteca `sqlx`).

| Campo (Go) | Coluna (MySQL) | Tipo Go | Significado |
| --- | --- | --- | --- |
| `ID` | `id` | `string` | Identificador único do valor, no formato UUID v4. |
| `RegistroID` | `registro_id` | `string` | UUID do registro de one-on-one (a reunião) ao qual esta resposta pertence. |
| `BlocoID` | `bloco_id` | `string` | UUID do bloco do template que foi respondido. |
| `ValorTexto` | `valor_texto` | `*string` (ponteiro) | Resposta em texto puro. Usado em blocos do tipo TEXT e HIGHLIGHT. É ponteiro porque pode ser nulo (`NULL` no banco). |
| `ValorJSON` | `valor_json` | `[]byte` | Resposta em formato JSON estruturado (bytes brutos). Usado em blocos do tipo LIST e IMAGE. |
| `CriadoEm` | `criado_em` | `time.Time` | Data/hora de criação do registro. |
| `AlteradoEm` | `alterado_em` | `*time.Time` (ponteiro) | Data/hora da última modificação. Fica `nil` (nulo) enquanto a resposta nunca foi alterada. |
| `DeletadoEm` | `deletado_em` | `*time.Time` (ponteiro) | Data/hora da exclusão lógica (soft delete). `nil` significa que o registro está **ativo**. |
| `DeletadoPor` | `deletado_por` | `*string` (ponteiro) | ID do usuário que realizou a exclusão lógica. |

> Observação para quem vem de C#: o asterisco (`*`) antes do tipo indica um **ponteiro**, que é a forma idiomática em Go de representar um valor que pode ser nulo (semelhante a `string?` / `Nullable<DateTime>` no .NET).

As colunas `deletado_em` e `deletado_por` implementam **exclusão lógica** (soft delete): nenhum registro é apagado fisicamente; apenas marca-se a data e o autor da exclusão. Todas as consultas de leitura filtram por `deletado_em IS NULL` para ignorar os registros já excluídos.

## Endpoints

Todas as rotas exigem **autenticação JWT** — o grupo de rotas aplica o `authMiddleware` (`valores.Use(authMiddleware)` e `aninhado.Use(authMiddleware)` em `controller.go`). O prefixo base da API é `/api/v1` (registrado em `cmd/api/rotas.go`).

| Método | Rota | Descrição | Autenticação |
| --- | --- | --- | --- |
| `POST` | `/api/v1/valores-registro` | Cria/salva a resposta de um bloco dentro de um registro de one-on-one. | JWT obrigatório |
| `GET` | `/api/v1/valores-registro/:id` | Busca uma resposta ativa pelo seu UUID. | JWT obrigatório |
| `PUT` | `/api/v1/valores-registro/:id` | Atualiza o conteúdo (texto ou JSON) de uma resposta existente. | JWT obrigatório |
| `DELETE` | `/api/v1/valores-registro/:id` | Realiza a exclusão lógica (soft delete) da resposta. | JWT obrigatório |
| `GET` | `/api/v1/registros-onebyone/:id/valores` | Lista todas as respostas de um registro específico (rota aninhada). | JWT obrigatório |

> Detalhe da rota aninhada: o parâmetro de URL chama-se `:id`, e no handler `ListarPorRegistro` ele é lido como `ctx.Param("id")`, sendo tratado como o **UUID do registro** (não do valor). Nos comentários Swagger ele aparece documentado como `registroId`.

## DTOs

Os DTOs (Data Transfer Objects) ficam em `dto.go`. São as classes de transporte da API — o que o cliente envia e o que a API devolve. As tags `binding:"..."` definem as validações aplicadas automaticamente pelo Gin ao fazer o bind do JSON (equivalente, em espírito, aos `DataAnnotations` do .NET, como `[Required]`).

### Entrada — `CriarValorRegistroDTO`

Usado no `POST`.

| Campo (JSON) | Tipo | Validação (`binding`) | Descrição |
| --- | --- | --- | --- |
| `registro_id` | `string` | `required` (obrigatório) | UUID do registro de one-on-one ao qual o valor pertence. |
| `bloco_id` | `string` | `required` (obrigatório) | UUID do bloco do template que está sendo respondido. |
| `valor_texto` | `*string` | `omitempty` (opcional) | Resposta em texto puro (blocos TEXT e HIGHLIGHT). |
| `valor_json` | `interface{}` | `omitempty` (opcional) | Resposta em JSON estruturado (blocos LIST e IMAGE). |

### Entrada — `AtualizarValorRegistroDTO`

Usado no `PUT`. Contém apenas os campos alteráveis.

| Campo (JSON) | Tipo | Validação (`binding`) | Descrição |
| --- | --- | --- | --- |
| `valor_texto` | `*string` | `omitempty` (opcional) | Novo conteúdo textual. |
| `valor_json` | `interface{}` | `omitempty` (opcional) | Novo conteúdo JSON estruturado. |

### Saída — `ValorRegistroRespostaDTO`

Retornado pela API em todas as respostas de sucesso (e em listas). Note que **não** expõe os campos de soft delete (`deletado_em` / `deletado_por`).

| Campo (JSON) | Tipo | Descrição |
| --- | --- | --- |
| `id` | `string` | Identificador único do valor. |
| `registro_id` | `string` | UUID do registro de one-on-one. |
| `bloco_id` | `string` | UUID do bloco respondido. |
| `valor_texto` | `*string` | Conteúdo textual (`null` para blocos JSON). |
| `valor_json` | `interface{}` | Conteúdo JSON já desserializado (`null` para blocos de texto). |
| `criado_em` | `time.Time` | Data/hora de criação. |
| `alterado_em` | `*time.Time` | Data/hora da última modificação (`null` se nunca alterado). |

## Regras de negócio

As regras estão concentradas em `usecase.go`. Pontos importantes do código:

- **Pelo menos um valor é obrigatório na criação.** Em `Criar`, se `valor_texto` e `valor_json` vierem ambos nulos, a operação falha com a mensagem `é necessário informar valor_texto ou valor_json`. Ou seja, não dá para salvar uma resposta vazia.
- **Geração de UUID no servidor.** O `ID` do novo valor não vem do cliente; é gerado pelo próprio backend com `uuid.New().String()`. O cliente informa apenas `registro_id` e `bloco_id`.
- **Serialização do JSON.** O campo `valor_json` chega como objeto livre (`interface{}`) e é convertido para bytes (`json.Marshal`) antes de ir ao banco, sendo armazenado na coluna `valor_json`. Se o conteúdo não puder ser serializado, retorna `valor_json inválido`.
- **Desserialização na saída.** No `mapper.go`, ao montar a resposta, os bytes de `valor_json` são lidos com `json.Unmarshal` de volta para objeto. Se a coluna estiver vazia/`NULL` (tamanho zero), o campo de saída fica `null` — evitando erro ao desserializar bytes nulos.
- **`CriadoEm` definido no servidor.** A data de criação é `time.Now()`, gerada no `UseCase`, não enviada pelo cliente.
- **Atualização parcial (patch).** Em `Atualizar`, o registro atual é carregado primeiro via `BuscarPorId`. Só os campos enviados no DTO são sobrescritos: se `valor_texto` vier `nil`, o texto anterior é preservado; o mesmo vale para `valor_json`. Você não precisa reenviar todos os campos.
- **`AlteradoEm` automático.** O timestamp `alterado_em` é definido no repositório (`repository.go`, método `Atualizar`) com `time.Now()` a cada atualização — o cliente não controla esse campo.
- **Verificação de existência antes de deletar.** Em `Deletar`, o `UseCase` primeiro chama `BuscarPorId`; se a resposta não existir (ou já estiver deletada), retorna erro antes de tentar excluir.
- **Exclusão lógica (soft delete).** Deletar **não remove** a linha do banco. O método `DeletarSoft` apenas preenche `deletado_em` e `deletado_por` (o ID do usuário autenticado, extraído do token via `middleware.ChaveUsuarioID` no `controller.go`). Se nenhuma linha for afetada (registro inexistente ou já deletado), retorna `valor de registro não encontrado ou já deletado`.
- **Leituras ignoram registros deletados.** Todas as consultas (`BuscarPorId`, `ListarPorRegistro`, e o `UPDATE` de atualização/exclusão) incluem a condição `deletado_em IS NULL`.
- **Ordenação da listagem.** `ListarPorRegistro` devolve os valores ordenados por `criado_em ASC` (do mais antigo para o mais novo).

> Não há hashing de senha nem herança de template implementados neste módulo — essas responsabilidades pertencem a outros módulos da API.

## Dependências

A montagem das camadas (injeção de dependência manual) acontece em `cmd/api/rotas.go`, encadeando os construtores:

| Construtor | Recebe | Por quê |
| --- | --- | --- |
| `NovoRepositorio(db *sqlx.DB)` | O pool de conexões MySQL (`*sqlx.DB`). | É a única camada que fala com o banco; precisa da conexão para executar as queries em `tb_valores_registro`. Retorna a interface `Repositorio`. |
| `NovoUseCase(repo Repositorio)` | A **interface** `Repositorio`. | As regras de negócio dependem do acesso a dados, mas só conhecem o contrato (interface), não a implementação concreta — o que facilita testes com mocks. Retorna a interface `UseCase`. |
| `NovoController(uc UseCase)` | A **interface** `UseCase`. | O controller HTTP delega tudo ao caso de uso; não conhece banco nem SQL. Retorna `*Controller`. |
| `RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc)` | O grupo de rotas do Gin e o middleware de autenticação JWT. | Para pendurar as rotas sob `/api/v1` e proteger todas elas com autenticação. |

Sequência real em `cmd/api/rotas.go`:

```
valorRepo := valorregistro.NovoRepositorio(db)
valorUseCase := valorregistro.NovoUseCase(valorRepo)
valorController := valorregistro.NovoController(valorUseCase)
valorController.RegistrarRotas(api, authMiddleware)
```

Bibliotecas externas usadas pelo módulo:

- `github.com/gin-gonic/gin` — framework HTTP (roteamento, bind de JSON).
- `github.com/jmoiron/sqlx` — extensão do `database/sql` que mapeia colunas para structs via tags `db`.
- `github.com/google/uuid` — geração dos UUIDs dos registros.
- `onebyone-api/pkg/response` — helpers de resposta HTTP padronizada (`Sucesso`, `Criado`, `ErroRequisicao`, `ErroNaoEncontrado`, `ErroInterno`).
- `onebyone-api/pkg/middleware` — fornece `ChaveUsuarioID`, usada para identificar o autor da exclusão a partir do token JWT.
