# Pacote `response`

> Centraliza o formato (envelope) das respostas JSON da API, garantindo que todo endpoint responda de forma padronizada, tanto em sucesso quanto em erro.

## O que faz

Este pacote define um "envelope" único de resposta JSON e um conjunto de funções auxiliares para escrever esse envelope na resposta HTTP. Em vez de cada controller montar manualmente o JSON e o código de status, ele chama uma função como `response.Sucesso(...)` ou `response.ErroNaoEncontrado(...)`. O resultado é que toda a API fala a mesma "língua": o cliente sempre recebe um objeto com o campo `sucesso` indicando se deu certo, mais os dados (em caso de sucesso) ou a mensagem de erro (em caso de falha). Para quem vem do .NET, pense nisso como uma classe utilitária estática (ex.: `ApiResponse.Ok(...)` / `ApiResponse.NotFound(...)`) que substitui o ato de montar o `HttpResponseMessage` na mão.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `response.go` | Define os tipos do envelope (`RespostaPadrao` e `ErroPadrao`) e todas as funções auxiliares que escrevem respostas de sucesso e de erro no `gin.Context`. |

## API pública

Símbolos exportados (começam com letra maiúscula, portanto visíveis para outros pacotes).

### Tipos

| Símbolo | Campos | O que faz |
| --- | --- | --- |
| `RespostaPadrao` | `Sucesso bool` (`json:"sucesso"`), `Dados interface{}` (`json:"dados,omitempty"`), `Erro string` (`json:"erro,omitempty"`) | Envelope JSON usado em todas as respostas. Em sucesso vem `{ "sucesso": true, "dados": {...} }`; em erro vem `{ "sucesso": false, "erro": "mensagem" }`. As tags `omitempty` fazem o campo `dados` sumir nos erros e o campo `erro` sumir nos sucessos. |
| `ErroPadrao` | `Sucesso bool` (`json:"sucesso"`, exemplo `false`), `Erro string` (`json:"erro"`, exemplo `"mensagem de erro legível"`) | Estrutura usada apenas nas anotações do Swagger para documentar o formato dos erros. Não é escrita diretamente na resposta; serve de modelo na documentação. |

### Funções

Todas recebem `*gin.Context` como primeiro parâmetro (o contexto da requisição do framework Gin) e escrevem o JSON com o código de status correspondente.

| Símbolo | Assinatura | O que faz |
| --- | --- | --- |
| `Sucesso` | `Sucesso(ctx *gin.Context, dados interface{})` | Responde **HTTP 200 OK** com `sucesso: true` e o payload informado em `dados`. |
| `Criado` | `Criado(ctx *gin.Context, dados interface{})` | Responde **HTTP 201 Created** com `sucesso: true` e os `dados`. Usado quando um novo recurso foi criado. |
| `Erro` | `Erro(ctx *gin.Context, status int, mensagem string)` | Responde com o código de status HTTP informado, `sucesso: false` e a `mensagem` no campo `erro`. É a função base que as demais funções de erro chamam internamente. |
| `ErroInterno` | `ErroInterno(ctx *gin.Context, mensagem string)` | Atalho para `Erro` com **HTTP 500 Internal Server Error**. Indica falha interna do servidor. |
| `ErroNaoEncontrado` | `ErroNaoEncontrado(ctx *gin.Context, mensagem string)` | Atalho para `Erro` com **HTTP 404 Not Found**. Recurso solicitado não existe. |
| `ErroRequisicao` | `ErroRequisicao(ctx *gin.Context, mensagem string)` | Atalho para `Erro` com **HTTP 400 Bad Request**. Dados da requisição inválidos. |
| `ErroNaoAutorizado` | `ErroNaoAutorizado(ctx *gin.Context, mensagem string)` | Atalho para `Erro` com **HTTP 401 Unauthorized**. Token ausente ou inválido. |
| `ErroProibido` | `ErroProibido(ctx *gin.Context, mensagem string)` | Atalho para `Erro` com **HTTP 403 Forbidden**. Usuário autenticado, mas sem permissão para o recurso. |
| `ErroConflito` | `ErroConflito(ctx *gin.Context, mensagem string)` | Atalho para `Erro` com **HTTP 409 Conflict**. Conflito com dados já existentes (ex.: e-mail já cadastrado). |

## Como é usado

Este pacote é importado como `onebyone-api/pkg/response` e é a "porta de saída" de quase toda a API. Ele é chamado principalmente em dois lugares:

1. **Nos controllers** (`internal/.../controller.go`) — depois de processar a requisição, o controller usa as funções para devolver a resposta. Pacotes que o importam: `equipe`, `templatebloco`, `valorregistro`, `organizacao`, `onebyone`, `usuario`, `registroonebyone`, `colaborador`, `template` e `auditoria`.
2. **No middleware de autenticação** (`pkg/middleware/auth.go`) — para barrar requisições sem token válido ou sem permissão antes mesmo de chegar ao controller.

Exemplo retirado do controller de usuário (`internal/usuario/controller.go`):

```go
func (c *Controller) Login(ctx *gin.Context) {
    var dto LoginDTO
    if err := ctx.ShouldBindJSON(&dto); err != nil {
        response.ErroRequisicao(ctx, "dados inválidos: "+err.Error())
        return
    }

    resultado, err := c.uc.Login(dto)
    if err != nil {
        response.Erro(ctx, http.StatusUnauthorized, err.Error())
        return
    }

    response.Sucesso(ctx, resultado)
}
```

Exemplo no middleware de autenticação (`pkg/middleware/auth.go`):

```go
if cabecalho == "" {
    response.ErroNaoAutorizado(ctx, "cabeçalho Authorization ausente")
    ctx.Abort()
    return
}
// ...
if !existe || role != "LIDER" {
    response.ErroProibido(ctx, "acesso restrito a líderes")
    ctx.Abort()
    return
}
```

As anotações do Swagger nos controllers também referenciam os tipos deste pacote para documentar o formato das respostas, por exemplo:

```go
// @Success 200 {object} response.RespostaPadrao{dados=LoginRespostaDTO} "Login realizado com sucesso"
// @Failure 400 {object} response.ErroPadrao                              "Dados inválidos"
```

## Detalhes importantes

- **Dependência externa:** usa o framework web **Gin** (`github.com/gin-gonic/gin`). O método `ctx.JSON(status, objeto)` serializa o objeto em JSON, define o cabeçalho `Content-Type: application/json` e escreve o código de status — tudo em uma chamada.
- **Envelope único em toda a API:** o campo booleano `sucesso` permite ao cliente saber rapidamente se a operação deu certo sem precisar interpretar apenas o código HTTP.
- **`omitempty` evita campos vazios:** graças às tags `json:"dados,omitempty"` e `json:"erro,omitempty"`, uma resposta de sucesso não traz o campo `erro` e uma resposta de erro não traz o campo `dados`. O JSON sai mais enxuto.
- **`interface{}` aceita qualquer payload:** o parâmetro `dados` é do tipo `interface{}` (equivalente ao `object` do C#), então `Sucesso` e `Criado` aceitam qualquer struct, slice ou mapa como conteúdo.
- **Hierarquia de funções:** `ErroInterno`, `ErroNaoEncontrado`, `ErroRequisicao`, `ErroNaoAutorizado`, `ErroProibido` e `ErroConflito` são apenas atalhos legíveis que chamam `Erro` com o status já preenchido. Para um status fora dessa lista, use `Erro` diretamente passando o `status int` desejado.
- **`ErroPadrao` é só documentação:** essa struct existe para o gerador de documentação Swagger; ela não é instanciada nem enviada em tempo de execução. O que de fato vai na resposta de erro é uma `RespostaPadrao` com `Sucesso: false`.
- **Sem efeitos colaterais ocultos:** o pacote não lê variáveis de ambiente, não acessa banco de dados nem valida tokens. É puramente formatação e escrita de resposta HTTP. A validação de JWT e as regras de permissão ficam no pacote `pkg/middleware`, que apenas reaproveita estas funções para reportar os erros.
- **Lembrete de uso:** assim como no exemplo, sempre coloque um `return` logo após chamar uma função de resposta dentro de um `if`, para não continuar executando o restante do handler depois de já ter respondido ao cliente.
