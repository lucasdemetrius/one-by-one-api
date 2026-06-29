# Módulo `usuario`

> Representa as pessoas que usam o OneByOne — tanto o gestor (LIDER) quanto o liderado (COLABORADOR) que participam das reuniões 1:1. É o módulo de cadastro, autenticação e foto de perfil.

## O que faz

Este módulo cuida de tudo relacionado a contas de usuário: criar (cadastro próprio ou por um usuário autenticado), listar, buscar, atualizar, excluir (de forma lógica) e fazer login com geração de token JWT. Também permite enviar uma foto de perfil, que é armazenada no S3 e devolvida como uma URL temporária.

Se você vem do .NET WebForms, pense assim: o `controller.go` é como o code-behind / a camada de API que recebe o request HTTP; o `usecase.go` é a sua camada de regras de negócio (Business Layer); o `repository.go` é o acesso a dados (como um DAL / ADO.NET fortemente tipado); e os DTOs são os contratos de entrada e saída — equivalentes às suas classes de ViewModel/Model. Cada camada só conhece a interface da camada de baixo, nunca a implementação concreta, o que facilita testes e substituição.

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---------|--------|------------------|
| `entity.go` | Entidade / Domínio | Define a struct `Usuario`, espelho fiel da tabela `tb_usuarios` no MySQL (tags `db`). |
| `dto.go` | Contrato da API | DTOs de entrada (`CriarUsuarioDTO`, `AtualizarUsuarioDTO`, `LoginDTO`) e de saída (`UsuarioRespostaDTO`, `LoginRespostaDTO`) com regras de validação. |
| `mapper.go` | Conversão | Função `ParaRespostaDTO` que converte a entidade `Usuario` em `UsuarioRespostaDTO`, omitindo senha e dados de exclusão. |
| `repository.go` | Acesso a dados | Interface `Repositorio` e implementação `repositorioMySQL` com todas as queries SQL sobre `tb_usuarios`. |
| `usecase.go` | Regras de negócio | Interface `UseCase` e implementação com validações, hash de senha, geração de JWT e upload de foto. |
| `controller.go` | HTTP / Entrada | Recebe requisições Gin, valida o corpo, chama o `UseCase` e formata a resposta. Registra as rotas. |

## Entidade e tabela no banco

A struct `Usuario` (em `entity.go`) mapeia diretamente a tabela **`tb_usuarios`** do MySQL. Cada campo usa a tag `db` para indicar a coluna correspondente.

| Campo (Go) | Coluna (MySQL) | Tipo Go | Significado |
|------------|----------------|---------|-------------|
| `ID` | `id` | `string` | Identificador único do usuário, no formato UUID v4. |
| `Nome` | `nome` | `string` | Nome completo do usuário. |
| `Email` | `email` | `string` | E-mail único, usado como login. |
| `Password` | `password` | `string` | Senha armazenada como **hash bcrypt** (nunca em texto puro). |
| `Role` | `role` | `string` | Papel do usuário: `LIDER` ou `COLABORADOR`. |
| `CriadoEm` | `criado_em` | `time.Time` | Data/hora de criação do registro. |
| `AlteradoEm` | `alterado_em` | `*time.Time` | Data/hora da última alteração; `nil` se nunca foi alterado. |
| `DeletadoEm` | `deletado_em` | `*time.Time` | Data/hora da exclusão lógica; `nil` significa registro ativo. |
| `DeletadoPor` | `deletado_por` | `*string` | ID do usuário que executou a exclusão lógica. |
| `FotoKey` | `foto_key` | `*string` | Chave do objeto da foto no S3 (ex.: `usuarios/{id}/foto.jpg`); `nil` quando não há foto. |

Observação sobre o `*` (ponteiro): campos como `*time.Time` e `*string` usam ponteiro justamente para poderem ser `nil`. Em C# seria o equivalente a um `DateTime?` ou `string` que aceita `null`. Aqui o `nil` representa "sem valor" / coluna `NULL` no banco.

## Endpoints

Todas as rotas abaixo ficam sob o prefixo **`/api/v1`** (definido no wireup em `cmd/api/rotas.go`). As rotas de `/auth` são públicas; as de `/usuarios` passam pelo `authMiddleware` e exigem um token JWT no cabeçalho `Authorization: Bearer <token>`.

| Método | Rota | Descrição | Autenticação |
|--------|------|-----------|--------------|
| `POST` | `/api/v1/auth/login` | Autentica com e-mail e senha e retorna um token JWT. | Pública |
| `POST` | `/api/v1/auth/registrar` | Auto-cadastro público. Cria usuário (role padrão `COLABORADOR`). | Pública |
| `POST` | `/api/v1/usuarios` | Cria um novo usuário (permite definir `role`, inclusive `LIDER`). | JWT |
| `GET` | `/api/v1/usuarios` | Lista todos os usuários ativos, ordenados por nome. | JWT |
| `GET` | `/api/v1/usuarios/:id` | Busca um usuário ativo pelo UUID. | JWT |
| `PUT` | `/api/v1/usuarios/:id` | Atualiza parcialmente um usuário (só os campos enviados). | JWT |
| `DELETE` | `/api/v1/usuarios/:id` | Exclusão lógica (soft delete); registra quem deletou. | JWT |
| `POST` | `/api/v1/usuarios/:id/foto` | Envia a foto de perfil (multipart/form-data, campo `foto`). | JWT |

Detalhe importante: a diferença entre `POST /auth/registrar` e `POST /usuarios` é que ambas chamam a mesma regra de criação (`UseCase.Criar`), mas a primeira é pública e pensada para auto-cadastro, enquanto a segunda exige autenticação. Tecnicamente, ambas aceitam o campo `role` no corpo; a documentação Swagger indica que para criar um `LIDER` deve-se usar a rota autenticada `POST /usuarios`.

## DTOs

### Entrada

**`CriarUsuarioDTO`** (usado em `POST /auth/registrar` e `POST /usuarios`)

| Campo | Tipo | Validação (`binding`) |
|-------|------|-----------------------|
| `nome` | `string` | `required`, `min=2`, `max=100` |
| `email` | `string` | `required`, `email`, `max=150` |
| `password` | `string` | `required`, `min=6`, `max=100` |
| `role` | `string` | `omitempty`, `oneof=LIDER COLABORADOR` (opcional; padrão `COLABORADOR`) |

**`AtualizarUsuarioDTO`** (usado em `PUT /usuarios/:id`) — todos os campos são opcionais; só os informados são alterados.

| Campo | Tipo | Validação (`binding`) |
|-------|------|-----------------------|
| `nome` | `string` | `omitempty`, `min=2`, `max=100` |
| `email` | `string` | `omitempty`, `email`, `max=150` |
| `role` | `string` | `omitempty`, `oneof=LIDER COLABORADOR` |

**`LoginDTO`** (usado em `POST /auth/login`)

| Campo | Tipo | Validação (`binding`) |
|-------|------|-----------------------|
| `email` | `string` | `required`, `email` |
| `password` | `string` | `required` |

> As tags `binding` são validadas automaticamente pelo Gin no `ShouldBindJSON`. Se a regra falhar, o controller responde `400 Dados inválidos` sem nem chegar à camada de negócio. É parecido com Data Annotations + ModelState.IsValid no ASP.NET.

### Saída

**`UsuarioRespostaDTO`** — o que a API devolve ao cliente. **Não** expõe senha nem campos de soft delete.

| Campo | Tipo | Observação |
|-------|------|------------|
| `id` | `string` | UUID do usuário. |
| `nome` | `string` | Nome completo. |
| `email` | `string` | E-mail. |
| `role` | `string` | `LIDER` ou `COLABORADOR`. |
| `criado_em` | `time.Time` | Data de criação. |
| `alterado_em` | `*time.Time` | `null` se nunca alterado. |
| `foto_url` | `*string` | URL presignada temporária da foto (expira em ~2h); `null` se sem foto. |

**`LoginRespostaDTO`** — retornado no login bem-sucedido.

| Campo | Tipo | Observação |
|-------|------|------------|
| `token` | `string` | JWT assinado, usado nas rotas protegidas. |
| `usuario` | `UsuarioRespostaDTO` | Dados do usuário autenticado. |

## Regras de negócio

Implementadas em `usecase.go`:

- **E-mail único na criação**: antes de criar, `Criar` chama `BuscarPorEmail`. Se já existir um usuário ativo com aquele e-mail, retorna o erro `"já existe um usuário com este e-mail"` (o controller traduz para HTTP `409 Conflito`).
- **Hash de senha com bcrypt**: a senha em texto puro nunca é salva. Em `Criar`, é gerado um hash com `bcrypt.GenerateFromPassword(..., 12)` (custo 12). No login, a comparação usa `bcrypt.CompareHashAndPassword`.
- **Role padrão**: se o cliente não informar `role` na criação, o sistema assume `"COLABORADOR"`.
- **UUID gerado no servidor**: o ID é criado com `uuid.New().String()` dentro do `UseCase`, não vem do cliente.
- **Atualização parcial**: em `Atualizar`, o usuário atual é carregado do banco e só os campos não-vazios do DTO sobrescrevem os valores existentes (`nome`, `email`, `role`). A senha **não** é alterável por esta rota.
- **E-mail único também na atualização**: se o `email` mudar, verifica-se se já pertence a outro usuário ativo; em caso positivo retorna `"este e-mail já está em uso por outro usuário"` (HTTP `409`).
- **Soft delete (exclusão lógica)**: `Deletar` confirma que o usuário existe e delega ao repositório, que preenche `deletado_em` e `deletado_por` em vez de apagar a linha. Todas as queries de leitura filtram por `deletado_em IS NULL`, então registros deletados ficam invisíveis. Como o filtro de e-mail também respeita isso, um e-mail de usuário deletado pode ser reutilizado em um novo cadastro.
- **Quem deletou é o usuário autenticado**: o controller pega o ID do token JWT (`middleware.ChaveUsuarioID` no contexto Gin) e passa como `deletadoPor`.
- **Login com mensagem genérica**: tanto e-mail inexistente quanto senha errada retornam o mesmo erro `"credenciais inválidas"`, para não revelar se um e-mail está ou não cadastrado. O controller responde `401`.
- **Geração do JWT**: no login, monta `middleware.ClaimsJWT` com `UsuarioID` e `Role`, define expiração (`cfg.JWTExpiracaoHoras` horas) e assina com `HS256` usando `cfg.JWTSecret`.
- **Upload de foto**: `UploadFoto` valida que o usuário existe, deriva a extensão a partir do `Content-Type` (`extensaoPorTipo`: jpeg→`.jpg`, png→`.png`, webp→`.webp`, padrão `.jpg`), monta a chave `usuarios/{id}/foto.{ext}`, envia ao S3 e salva a chave no banco via `AtualizarFoto`. O controller limita o arquivo a **5MB** e só aceita os tipos `image/jpeg`, `image/png` e `image/webp`.
- **URL presignada da foto**: `gerarFotoURL` produz uma URL temporária a partir da `foto_key` usando `storage.ExpiracaoURLFoto`. Em caso de erro (ou sem foto / sem serviço de storage), retorna `nil` silenciosamente para não quebrar listagens.
- **Mapper protege dados sensíveis**: `ParaRespostaDTO` (em `mapper.go`) é o único caminho de saída da entidade e propositalmente omite `password`, `deletado_em` e `deletado_por`.

## Dependências

O módulo segue injeção de dependências por interface. Cada construtor recebe o que precisa pronto e configurado:

- **`NovoRepositorio(db *sqlx.DB)`** — recebe o pool de conexões MySQL (`sqlx`). É a única peça que conhece SQL e a tabela `tb_usuarios`.
- **`NovoUseCase(repo Repositorio, cfg *config.Config, armazenamento storage.Armazenamento)`** — recebe:
  - `repo`: o repositório, para acesso a dados (via interface, não a implementação concreta).
  - `cfg`: as configurações da aplicação, usadas para gerar o JWT (`JWTSecret`, `JWTExpiracaoHoras`).
  - `armazenamento`: o serviço de storage S3, usado para upload de fotos e geração de URLs presignadas.
- **`NovoController(uc UseCase)`** — recebe apenas o `UseCase`. O controller não conhece banco nem S3 diretamente; só orquestra HTTP.

A montagem (wireup) acontece em `cmd/api/rotas.go`, na ordem repositório → usecase → controller, e então `RegistrarRotas(api, authMiddleware)` pendura as rotas no grupo `/api/v1`, passando o middleware de autenticação JWT para as rotas protegidas.
