# Pacote `middleware`

> Middlewares HTTP do Gin que cuidam de autenticação (JWT), autorização por papel e auditoria automática das requisições.

## O que faz

Este pacote concentra os "interceptadores" que rodam **antes e/ou depois** dos handlers de cada rota. Se você vem de C#/.NET, pense neles como os `HttpModule`/`IHttpHandler` do WebForms ou os filtros (`ActionFilter`) do ASP.NET: código que executa em volta da requisição sem precisar repetir lógica em cada controller. Aqui temos três responsabilidades: validar o token JWT (`AutenticarJWT`), restringir rotas só a líderes (`ApenasLider`) e registrar automaticamente em auditoria toda operação de escrita (`RegistrarAuditoria`).

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `auth.go` | Autenticação e autorização: valida o token JWT do cabeçalho `Authorization`, injeta `usuario_id` e `usuario_role` no contexto e oferece o middleware `ApenasLider`. Define também o tipo `ClaimsJWT`. |
| `auditoria.go` | Middleware que, após cada requisição de escrita bem-sucedida (POST/PUT/DELETE) e logins, grava um registro de auditoria. Faz isso via a interface mínima `AuditoriaUseCase`, sem importar diretamente o módulo de auditoria (evita importação circular). |

## API pública

Apenas os símbolos com letra inicial **maiúscula** são exportados (acessíveis de fora do pacote — equivalente ao `public` do C#).

| Símbolo | Assinatura | O que faz |
| --- | --- | --- |
| `ChaveUsuarioID` | `const ChaveUsuarioID = "usuario_id"` | Chave usada para gravar/recuperar o ID do usuário autenticado no contexto do Gin. |
| `ChaveUsuarioRole` | `const ChaveUsuarioRole = "usuario_role"` | Chave usada para gravar/recuperar o papel (role) do usuário no contexto do Gin. |
| `ClaimsJWT` | `type ClaimsJWT struct { UsuarioID string; Role string; jwt.RegisteredClaims }` | Representa o payload do token JWT. Carrega `UsuarioID` e `Role`, além dos campos padrão (`exp`, `iat`, etc.) via `RegisteredClaims`. |
| `AutenticarJWT` | `func AutenticarJWT(cfg *config.Config) gin.HandlerFunc` | Retorna um middleware que valida o `Bearer <token>`. Token ausente/mal formatado/inválido/expirado responde **401** e aborta a requisição. Se válido, injeta `usuario_id` e `usuario_role` no contexto. |
| `ApenasLider` | `func ApenasLider() gin.HandlerFunc` | Retorna um middleware de autorização que só deixa passar usuários com role `LIDER`. Caso contrário responde **403**. Deve ser usado **depois** de `AutenticarJWT`. |
| `AuditoriaUseCase` | `interface { Registrar(usuarioID *string, acao, entidade string, entidadeID *string, ip, userAgent string) }` | Interface mínima que o middleware de auditoria exige. Quem implementa é o módulo `internal/auditoria`. |
| `RegistrarAuditoria` | `func RegistrarAuditoria(uc AuditoriaUseCase) gin.HandlerFunc` | Retorna um middleware que, após o handler rodar, grava um registro de auditoria para operações POST/PUT/DELETE (e logins) bem-sucedidas. |

> As funções `extrairAcaoEntidade`, `extrairEntidadeID` (em `auditoria.go`) começam com letra minúscula, portanto são **privadas** do pacote e não fazem parte da API pública.

## Como é usado

Os middlewares são registrados na montagem das rotas, em `cmd/api/rotas.go`:

```go
router := gin.Default()
api := router.Group("/api/v1")

// Cria o middleware de autenticação a partir das configs (JWT_SECRET etc.)
authMiddleware := middleware.AutenticarJWT(cfg)

// Auditoria aplicada GLOBALMENTE ao grupo /api/v1 — grava toda operação de escrita
api.Use(middleware.RegistrarAuditoria(auditoriaUseCase))
```

O `authMiddleware` resultante é repassado para cada módulo no seu `RegistrarRotas`, que o aplica só nos grupos protegidos. Exemplo real em `internal/usuario/controller.go`:

```go
// Rotas públicas (sem token)
router.POST("/auth/login", c.Login)
router.POST("/auth/registrar", c.Registrar)

// Rotas protegidas (exigem Bearer token válido)
usuarios := router.Group("/usuarios")
usuarios.Use(authMiddleware)
```

Dentro dos controllers, o usuário autenticado é recuperado pela chave exportada:

```go
usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
```

O tipo `ClaimsJWT` também é reaproveitado na geração do token, em `internal/usuario/usecase.go`, durante o login:

```go
claims := middleware.ClaimsJWT{ UsuarioID: u.ID, Role: u.Role, ... }
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
```

## Detalhes importantes

- **Configuração lida (via `config.Config`):** `JWTSecret` (variável de ambiente `JWT_SECRET`) é a chave usada para validar a assinatura do token. A expiração (`JWTExpiracaoHoras` / `JWT_EXPIRACAO_HORAS`) é aplicada na **geração** do token no módulo `usuario`, não aqui — mas o middleware respeita o `exp` ao validar.
- **Validação de JWT:** o `AutenticarJWT` exige o formato `Bearer <token>` (case-insensitive no "bearer"). Ele só aceita tokens assinados com **HMAC** — se o algoritmo do token não for HMAC, é rejeitado com `jwt.ErrSignatureInvalid`. Isso protege contra o ataque de troca de algoritmo (`alg` confusion). Assinatura e expiração são checadas de uma vez no `jwt.ParseWithClaims`.
- **Respostas de erro padronizadas:** os erros usam o envelope do pacote `response` — `response.ErroNaoAutorizado` gera **HTTP 401** (token ausente/inválido/expirado) e `response.ErroProibido` gera **HTTP 403** (role diferente de `LIDER`). Em ambos os casos o middleware chama `ctx.Abort()` para impedir que o handler execute.
- **Papéis (roles):** o sistema trabalha com dois papéis — `LIDER` e `COLABORADOR`. O `ApenasLider` libera somente `LIDER`.
- **Ordem importa:** `ApenasLider` depende dos dados injetados por `AutenticarJWT`. Sempre aplique a autenticação antes da autorização.
- **Auditoria roda DEPOIS do handler:** `RegistrarAuditoria` chama `ctx.Next()` primeiro e só então decide se grava. Ela **ignora** requisições que não alteram estado (qualquer método diferente de POST/PUT/DELETE) e respostas com status **>= 400** (não audita tentativas que falharam).
- **Mapeamento automático de ação e entidade:** a partir do método HTTP e do path, o middleware deduz a ação (`POST → CRIAR`, `PUT → ATUALIZAR`, `DELETE → DELETAR`) e o nome da entidade. Há casos especiais: `/auth/login → (LOGIN, usuario)`, `/auth/registrar → (CRIAR, usuario)`, sufixo `/foto → UPLOAD_FOTO`, e rotas de `eventos` são ignoradas para evitar auditoria duplicada. Segmentos como `api`, `v1` e parâmetros (`:id`) são descartados na dedução.
- **Nomes de entidade normalizados:** o path é traduzido para nomes legíveis, por exemplo `organizacoes → organizacao`, `equipes → equipe`, `colaboradores → colaborador`, `templates → template`, `template-blocos → template_bloco`, `registros-onebyone → registro_onebyone`, `valores-registro → valor_registro`, `usuarios → usuario`, `onebyone → onebyone`.
- **Dados capturados na auditoria:** o `usuarioID` (quando há `usuario_id` no contexto), o `entidadeID` (o `:id` da rota, quando existir), o IP do cliente (`ctx.ClientIP()`) e o cabeçalho `User-Agent`.
- **Sem dependência circular:** `auditoria.go` define a interface local `AuditoriaUseCase` em vez de importar o pacote `internal/auditoria`, mantendo o `pkg/middleware` desacoplado dos módulos de negócio.
