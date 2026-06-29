# Módulo `convite`

> Convite de liderado: o gestor gera um link (UUID) + código (contra-senha); o
> liderado abre o link, informa o código, define a senha e ganha acesso —
> vinculando sua conta de usuário ao colaborador. Cobre troca de gestor/empresa
> (mesma pessoa, novo convite).

## O que faz

O liderado **nunca se cadastra sozinho** — ele entra por convite. Este módulo
gera o convite (protegido, para o gestor) e processa a visualização e o aceite
(públicos, para o liderado). No aceite, **reaproveita** o módulo `usuario`
(criar conta + login/JWT) e o módulo `colaborador` (vincular `usuario_id`).

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---|---|---|
| `entity.go` | Domínio | Struct `Convite` (tabela `tb_convites`) + constantes de status |
| `dto.go` | Contrato | `ConviteGeradoDTO`, `ConvitePublicoDTO`, `AceitarConviteDTO` |
| `repository.go` | Dados | SQL sobre `tb_convites` (criar, buscar por token, marcar aceito, cancelar pendentes) |
| `usecase.go` | Negócio | Gera código, valida, e orquestra o aceite (cria/usa conta + vincula) |
| `controller.go` | HTTP | Rotas de gerar (protegida) e ver/aceitar (públicas) |

## Entidade e tabela no banco

Tabela **`tb_convites`** (migration `004`):

| Campo | Coluna | Tipo | Significado |
|---|---|---|---|
| `ID` | `id` | string (UUID) | Token do link `/convite/{id}` |
| `ColaboradorID` | `colaborador_id` | string | Liderado convidado |
| `CodigoHash` | `codigo_hash` | string | Hash **bcrypt** do código (contra-senha) |
| `Status` | `status` | string | `PENDENTE`, `ACEITO` ou `CANCELADO` |
| `ExpiraEm` | `expira_em` | datetime | Validade (7 dias) |
| `CriadoEm` / `AceitoEm` | `criado_em` / `aceito_em` | datetime | Datas |

## Endpoints

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| `POST` | `/api/v1/colaboradores/{id}/convite` | Gera o convite (devolve token + **código em texto puro, só aqui**) | JWT (gestor) |
| `GET` | `/api/v1/convites/{token}` | Dados públicos do convite (nome do liderado, validade) | Pública |
| `POST` | `/api/v1/convites/{token}/aceitar` | Valida código + senha, cria/usa a conta e devolve o login (JWT) | Pública |

## DTOs

- **`ConviteGeradoDTO`** (saída do gerar): `token`, `codigo` (uma vez), `link`, `expira_em`.
- **`ConvitePublicoDTO`** (saída do ver): `token`, `valido`, `colaborador_nome`, `email`.
- **`AceitarConviteDTO`** (entrada do aceitar): `codigo` (required), `senha` (required, min 6).

## Regras de negócio

- **Código** de 6 caracteres (alfabeto sem `O/0/I/1`), gerado com `crypto/rand` e
  guardado como **hash bcrypt** — nunca em texto puro no banco.
- Ao gerar, **cancela convites pendentes anteriores** do mesmo colaborador.
- No **aceite**: valida status `PENDENTE` + não expirado + código correto. Então:
  - se já existe usuário com aquele e-mail e a senha confere → usa (caso de troca
    de gestor/empresa);
  - senão → cria uma conta nova (role `COLABORADOR`) com a senha informada.
  - vincula `colaborador.usuario_id` à conta e marca o convite como `ACEITO`.
  - devolve o **login (token JWT)** para acesso imediato.

## Dependências

`NovoUseCase(repo, usuarioUC, colaboradorUC)` — recebe o repositório de convites,
o **UseCase de usuario** (criar conta + login) e o **UseCase de colaborador**
(buscar + vincular `usuario_id`). Montado em `cmd/api/rotas.go` após os módulos
`usuario` e `colaborador`.
