# CLAUDE.md — OneByOne

> Guia de orientação para o Claude Code e para desenvolvedores neste repositório.
> Tudo aqui é escrito em **português**, assim como o código (funções, variáveis,
> parâmetros e comentários). Mantenha esse padrão.

---

## 1. Visão geral do produto

**OneByOne** é um sistema para gerenciar **reuniões one-on-one (1:1)** entre
**líderes** e **colaboradores**. O líder organiza sua estrutura
(organização → equipe → colaborador), define **templates de pauta** e registra
o que foi conversado em cada reunião, com **respostas** estruturadas por bloco.

O produto é dividido em dois repositórios, ambos dentro de
`\\wsl.localhost\Ubuntu-20.04\home\ubuntu\onebyone`:

| Pasta            | O que é                         | Status        |
|------------------|---------------------------------|---------------|
| `onebyone-api`   | Backend REST em **Go**          | Em uso (este) |
| `onebyone-app`   | Frontend em **React JS**        | Futuro        |

> **Frontend (futuro):** será em React. O usuário **não quer cara de "app de IA"**
> — o objetivo é algo **inovador** no visual e na experiência. Isso será detalhado
> quando o `onebyone-app` for criado.

---

## 2. Stack do backend

- **Go 1.24** — linguagem
- **Gin** — framework HTTP (router + middlewares)
- **MySQL 8.0** — banco de dados
- **sqlx** — acesso ao banco mapeando structs (não é ORM; SQL é escrito à mão)
- **JWT (golang-jwt/v5)** — autenticação via Bearer Token
- **bcrypt** — hash de senha (custo 12)
- **AWS S3** — fotos privadas, servidas por URL presignada (validade ~2h)
- **Swagger / swaggo** — documentação interativa em `/swagger/index.html`
- **Docker + Docker Compose** — **tudo roda em container** (ver seção 8)

---

## 3. Como rodar (Docker para tudo)

> A regra do projeto é **Docker para tudo**: você não precisa instalar Go nem
> MySQL na máquina. Só Docker.

```bash
# Na pasta onebyone-api/
docker compose up --build
```

Isso sobe **dois containers**:

1. `onebyone-mysql` — banco MySQL 8.0, com as migrations aplicadas automaticamente
2. `onebyone-api`   — a API Go compilada

Endereços depois de subir:

- API ............ http://localhost:8090/api/v1
- Healthcheck .... http://localhost:8090/api/v1/health
- Swagger UI ..... http://localhost:8090/swagger/index.html

> Portas escolhidas para não conflitar com outros projetos: API **8090**,
> App **3100**, Vite dev **5273**, MySQL **3307**.

Para parar: `docker compose down` (os dados do banco ficam no volume `mysql_data`).
Para zerar o banco do zero: `docker compose down -v`.

### Rodando só o banco (desenvolvimento com `go run`)

Se quiser editar o Go e rodar fora do container (iteração mais rápida):

```bash
docker compose up -d mysql        # sobe só o MySQL
go run ./cmd/api                  # roda a API local, lê o .env
```

Nesse modo o `.env` usa `DB_HOST=localhost`. Dentro do container a API usa
`DB_HOST=mysql` (nome do serviço na rede do Compose). Veja a seção 8.

---

## 4. Mapa de pastas

```
onebyone-api/
├── cmd/api/main.go          ← ponto de entrada (ver seção 5)
├── internal/                ← os módulos de negócio (um por subpasta)
│   ├── usuario/
│   ├── organizacao/
│   ├── equipe/
│   ├── colaborador/
│   ├── template/
│   ├── templatebloco/
│   ├── onebyone/
│   ├── registroonebyone/
│   ├── valorregistro/
│   └── auditoria/
├── pkg/                     ← código compartilhado, sem regra de negócio
│   ├── config/              ← carrega variáveis de ambiente (.env)
│   ├── database/            ← abre a conexão com o MySQL
│   ├── middleware/          ← auth JWT e auditoria
│   ├── response/            ← helpers de resposta HTTP padronizada
│   └── storage/             ← integração com o S3
├── migrations/              ← scripts SQL (aplicados pelo MySQL na 1ª subida)
├── docs/                    ← arquivos gerados pelo Swagger (não editar à mão)
├── docker-compose.yml       ← orquestra MySQL + API
├── Dockerfile               ← build multi-stage da API
└── .env                     ← segredos locais (NÃO versionar)
```

`internal/` é uma convenção do Go: só pode ser importado por código **deste**
módulo (`onebyone-api`). É onde mora a regra de negócio.
`pkg/` é infraestrutura reutilizável, sem regra de negócio.

---

## 5. `main.go` e `rotas.go` — onde mora a "fiação" (leia isto)

O ponto de entrada está dividido em **dois arquivos**, ambos no `package main`:

- **`cmd/api/main.go`** — só o **ciclo de vida**: carrega config → conecta no
  banco → inicializa o S3 → chama `ConfigurarRotas(...)` → sobe o servidor. ~70 linhas.
- **`cmd/api/rotas.go`** — toda a **fiação de injeção de dependências (DI)** e o
  registro das rotas. É aqui que cada módulo é montado. Para adicionar um módulo
  novo, você mexe **só aqui**.

Vindo do mundo **WebForms / .NET 4.8**, essa fiação parece "código demais", mas
ela **não tem lógica de negócio nenhuma**. Faz só uma coisa: **monta o grafo de
injeção de dependências (DI) na mão** e liga as rotas.

Em .NET você teria um container de DI (Autofac, o `Startup.cs`, etc.) fazendo
isso "por mágica" nos bastidores. Em Go a convenção é **fazer isso explícito**,
escrito à mão, para ficar tudo visível em um lugar só. Cada módulo segue
sempre o mesmo trio de 3 linhas:

```go
// Exemplo: módulo usuario
usuarioRepo       := usuario.NovoRepositorio(db)              // 1. cria o Repository (fala com o banco)
usuarioUseCase    := usuario.NovoUseCase(usuarioRepo, cfg, s3Svc) // 2. cria o UseCase (regra de negócio)
usuarioController := usuario.NovoController(usuarioUseCase)   // 3. cria o Controller (HTTP)
usuarioController.RegistrarRotas(api, authMiddleware)         // 4. liga as rotas no /api/v1
```

Repare na ordem: cada camada **recebe a de baixo** no construtor. É isso que
"injeção de dependência" significa aqui — nada de framework, só passar o objeto
pronto adiante. Por isso o `rotas.go` cresce 4 linhas a cada módulo novo; é
linear e previsível, não complexo.

A sequência completa do boot:

1. `config.Carregar()` — lê o `.env` / variáveis de ambiente *(main.go)*
2. `database.NovaConexao(cfg)` — abre o pool de conexões MySQL *(main.go)*
3. `storage.NovoArmazenamentoS3(cfg)` — inicializa o cliente S3 *(main.go)*
4. `ConfigurarRotas(cfg, db, s3Svc)` — em `rotas.go`: cria o router Gin e o
   middleware de auth JWT, registra o middleware global de **auditoria**, monta
   cada módulo (o trio acima) e registra suas rotas
5. `router.Run(":8080")` — sobe o servidor HTTP *(main.go)*

> **Atenção a uma dependência entre módulos:** `registroonebyone` recebe o
> `onebyoneUseCase` no construtor porque precisa resolver o template pela regra
> de herança **colaborador → equipe → organização → padrão do líder**. Por isso
> a ordem de criação no `rotas.go` importa: `onebyone` é montado **antes** de
> `registroonebyone`.

---

## 6. Anatomia de um módulo (Clean Architecture)

Todo módulo em `internal/<nome>/` segue **o mesmo padrão de 6 arquivos**. Aprenda
um e você conhece todos. Fluxo de uma requisição:

```
HTTP → Controller → UseCase → Repository → MySQL
                       │
                     Mapper  (Entity → DTO)
```

| Arquivo          | Camada      | Responsabilidade                                              | Pode acessar |
|------------------|-------------|---------------------------------------------------------------|--------------|
| `controller.go`  | HTTP        | Recebe a requisição, valida o JSON, chama o UseCase. **Nunca toca no banco.** | UseCase |
| `usecase.go`     | Negócio     | Regras de negócio, validações, orquestração. **Não conhece HTTP.** | Repository, Mapper |
| `repository.go`  | Dados       | SQL puro com sqlx. **Só I/O de banco.** | MySQL |
| `entity.go`      | Domínio     | A struct que espelha a tabela do banco | — |
| `dto.go`         | Contrato    | Structs de entrada/saída da API (o que o cliente envia/recebe) | — |
| `mapper.go`      | Tradução    | Converte `Entity` ⇄ `DTO` (esconde campos internos do cliente) | — |

Regras de ouro do fluxo:

- **Controller** não chama Repository direto — sempre via UseCase.
- **UseCase** não importa nada de `gin` (não sabe que existe HTTP).
- **Repository** não tem `if` de regra de negócio — só SQL.
- O cliente nunca vê a `Entity` crua — sempre o `DTO` (via Mapper). É assim que
  o hash da senha, por exemplo, nunca vaza na resposta.

> O módulo `auditoria` é a exceção: não tem `mapper.go` porque é só leitura de
> trilha e gravação automática via middleware.

### Cada camada é uma `interface`

`UseCase` e `Repository` são declarados como **interface** dentro do próprio
módulo. O Controller depende da **interface** `UseCase`, não da struct concreta.
Isso é o que torna o código testável (dá pra trocar por um mock) e é o motivo
de existirem os construtores `NovoRepositorio` / `NovoUseCase` / `NovoController`.

---

## 6.1 Documentação detalhada por módulo

Este CLAUDE.md é a **visão geral**. Cada módulo tem um **`README.md` próprio**
dentro da sua pasta, com endpoints (rotas reais), entidade/tabela, DTOs, regras
de negócio e dependências. Comece pelo módulo que vai mexer:

> 📚 **O catálogo completo está em [docs/CATALOGO.md](docs/CATALOGO.md)** — todas as
> funcionalidades, regras de negócio e validações de cada módulo, geradas a partir do
> código. Use-o como referência única; os READMEs abaixo trazem o mesmo, focado em um módulo só.

**Módulos de negócio (`internal/`):**

| Módulo | Doc | O que faz |
|---|---|---|
| usuario | [README](internal/usuario/README.md) | Cadastro, login JWT, perfil e foto |
| organizacao | [README](internal/organizacao/README.md) | Organização do líder |
| equipe | [README](internal/equipe/README.md) | Times da organização |
| colaborador | [README](internal/colaborador/README.md) | Membros de uma equipe |
| convite | [README](internal/convite/README.md) | Convite de liderado (link + código) |
| blocotema | [README](internal/blocotema/README.md) | Conteúdo rico dos temas (texto/link/imagem/marco) |
| classificacao | [README](internal/classificacao/README.md) | Matriz 9-box (desempenho × potencial) |
| pdi | [README](internal/pdi/README.md) | Plano de Desenvolvimento Individual (objetivos + prazos) |
| acompanhamento | [README](internal/acompanhamento/README.md) | Sentimento, entregas, feedbacks e estudos do liderado |
| aovivo | [README](internal/aovivo/README.md) | 1:1 ao vivo (WebSocket: presença, cursores, board) |
| tabuleiro | [README](internal/tabuleiro/README.md) | Estado persistido do tabuleiro da pauta (1:1 ao vivo) |
| agendamento | [README](internal/agendamento/README.md) | Agenda de 1:1 com recorrência + lembretes por e-mail |
| template | [README](internal/template/README.md) | Templates de pauta |
| templatebloco | [README](internal/templatebloco/README.md) | Blocos de um template |
| onebyone | [README](internal/onebyone/README.md) | A reunião 1:1 (inclui encerrar → REALIZADO) |
| saude1a1 | [README](internal/saude1a1/README.md) | Saúde do 1:1: cadência, atrasados e streak 🔥 |
| registroonebyone | [README](internal/registroonebyone/README.md) | Registros de uma reunião |
| valorregistro | [README](internal/valorregistro/README.md) | Respostas preenchidas |
| notificacao | [README](internal/notificacao/README.md) | Sino in-app + cron de avisos da agenda + preferências |
| ia | [README](internal/ia/README.md) | IA plugável BYOK (Claude/OpenAI/DeepSeek/Grok) — chave AES-GCM |
| auditoria | [README](internal/auditoria/README.md) | Trilha de atividades |

**Pacotes de infraestrutura (`pkg/`):**

| Pacote | Doc | O que faz |
|---|---|---|
| config | [README](pkg/config/README.md) | Carrega variáveis de ambiente |
| cripto | §12.21 | Cifra/decifra a chave de API de IA (AES-GCM) |
| database | [README](pkg/database/README.md) | Conexão com o MySQL |
| email | §12.21 | Envio SMTP (dormente sem config) + templates HTML |
| middleware | [README](pkg/middleware/README.md) | Auth JWT e auditoria |
| response | [README](pkg/response/README.md) | Envelope padrão de resposta |
| storage | [README](pkg/storage/README.md) | Integração com o S3 |

> Ao alterar um módulo (rotas, DTOs, regras), **atualize o README dele** junto.

---

## 7. Convenções do código (siga sempre)

- **Idioma:** tudo em português — nomes de função, variável, parâmetro, comentário,
  mensagem de erro e log. Ex.: `NovoUseCase`, `BuscarPorId`, `cfg`, `deletadoPor`.
- **Construtores:** sempre `Novo<Coisa>(...)` retornando ponteiro. Ex.: `NovoController`.
- **Cabeçalho de arquivo:** todo arquivo começa com o bloco de comentário
  `Pacote / Arquivo / Descrição / Autor / Criado em`. Mantenha ao criar arquivos novos.
- **IDs:** são **UUID** (string), gerados na aplicação (`google/uuid`), não auto-incremento.
- **Soft delete:** registros não são apagados — preenchem `deletado_em` e
  `deletado_por`. Toda query de leitura filtra `deletado_em IS NULL`.
- **Respostas HTTP:** sempre pelos helpers de `pkg/response` (`Sucesso`, `Criado`,
  `ErroRequisicao`, `ErroNaoEncontrado`, `ErroConflito`, `ErroInterno`). Nunca
  monte `ctx.JSON` na mão no controller — mantém o envelope padronizado.
- **Swagger:** os comentários `// @Summary`, `// @Router` etc. acima de cada
  handler **geram** a doc. Se você mexer numa rota, atualize o comentário e
  rode `swag init` (ver seção 9).
- **Erros de negócio conhecidos** viajam como `error` com mensagem fixa, e o
  Controller compara a string pra escolher o status HTTP (ex.: e-mail duplicado → 409).

---

## 7.1 Segurança e autorização (POSSE) — leia antes de criar/editar rotas

> **Princípio:** o `authMiddleware` (JWT) prova **quem você é**; ele **não** prova
> que o recurso é **seu**. Toda rota que recebe um `:id` de recurso PRECISA checar
> a **posse** no UseCase — senão vira IDOR (qualquer usuário logado acessa dado
> de outro trocando o UUID). Isto foi auditado e está sendo corrigido módulo a
> módulo (ver memória `remediacao-seguranca-idor`).

**As 3 regras de ouro:**

1. **A identidade viaja do Controller até a decisão.** O controller lê
   `usuarioID := ctx.GetString(middleware.ChaveUsuarioID)` e **repassa ao UseCase**.
   Nunca confie em id de dono vindo do corpo/DTO.
2. **A posse é checada no UseCase; o SQL é a defesa em profundidade.** Em
   escrita/exclusão, reforce o `WHERE` com o dono e cheque `RowsAffected`.
3. **Recurso alheio → 404, não 403** (não revele que o id existe). Use o
   sentinela `ErrAcessoNegado` do módulo e mapeie para `response.ErroNaoEncontrado`.

**Quem é o dono?** Sempre o **LÍDER** (`usuario_id` com role `LIDER`), por 2 cadeias:

- **Cadeia A (dono direto):** `organizacao` / `equipe` / `onebyone` / `agendamento`
  / `template` têm `usuario_id == JWT`.
- **Cadeia B (indireto):** `colaborador → equipe.usuario_id` (ou `organizacao.usuario_id`);
  e `blocotema` / `classificacao` → `colaborador_id` → Cadeia B; `registroonebyone`
  → `onebyone`; `valorregistro` → `registro` → `onebyone`; `templatebloco` → `template`.

> ⚠️ **Armadilha:** `tb_colaboradores.usuario_id` é a conta do **liderado** (a pessoa),
> NÃO o líder dono. Nunca use como prova de posse do gestor.

**Peça central (reutilize, não reescreva):** o módulo `colaborador` expõe os
helpers de posse da Cadeia B:
`PertenceAoLider(colaboradorID, usuarioID)` (só o líder dono),
`PodeAcessar(colaboradorID, usuarioID)` (líder dono **OU** o próprio liderado — para
o conteúdo colaborativo do 1:1 ao vivo) e `OrganizacaoPertenceAoLider(...)`.
Os módulos `blocotema`, `classificacao`, `agendamento` e `convite` chamam esses
helpers em vez de duplicar SQL de posse.

**`ApenasLider()`** (`pkg/middleware`) é **defesa em profundidade** nas rotas de
**gestão** (criar/editar/excluir/ listar PII) — barra contas `COLABORADOR` de
imediato. **NÃO substitui** a checagem de posse (um líder ainda veria o recurso de
outro líder com só `ApenasLider`). E **não** aplique em rotas onde o liderado
acessa o próprio dado (ex.: `GET /meu-colaborador`, ver/editar seu 1:1).

**Vínculo de conta (anti-sequestro):** o `usuario_id` do colaborador só é definido
pelo fluxo de **aceite de convite** (`colaborador.VincularConta`). O `PUT
/colaboradores/:id` **ignora** `usuario_id` de propósito.

> **Isolamento entre empresas (uma conta ↔ um colaborador atual):** o e-mail de liderado é
> único **por líder/empresa**, então empresas diferentes podem ter o mesmo `liderado@x.com`;
> a **conta de login** (`tb_usuarios`) é única global e é **reusada** no aceite (caso "troca de
> empresa"). Para a pessoa não manter acesso ao 1:1 da empresa anterior, `VincularConta` chama
> `repo.DesvincularOutrasContas(usuarioID, exceto)` — zera o `usuario_id` dos registros antigos
> dessa conta. Assim, ao entrar numa nova empresa, o liderado **perde o acesso** ao 1:1 da
> anterior (o histórico/registro fica com o gestor de lá, só sem o vínculo de login). Validado
> por smoke ponta-a-ponta (board antigo → 404 após a troca).

**Regras de e-mail (liderado e equipe) — validadas no UseCase, mapeadas para 409:**

> **E-mails são normalizados (canônicos): `texto.NormalizarEmail` = `LOWER(TRIM())`.**
> Aplicado ao gravar E ao comparar em `usuario` (Criar/Login/Atualizar) e `colaborador`
> (Criar/Atualizar/ImportarLote). Os dados antigos foram normalizados na migration `016`.
> Assim "Joao@X.com" e "joao@x.com " são o mesmo e-mail — login funciona em qualquer
> caixa e as regras 1 e 2 abaixo não podem ser burladas por diferença de maiúsculas.
> *(Dívida restante, não fechada: a unicidade de e-mail de liderado é só aplicacional —
> sem índice único no banco, há janela TOCTOU em escritas concorrentes do mesmo líder.)*

1. **E-mail de liderado é único por líder.** Um gestor não pode ter dois liderados
   ativos com o mesmo e-mail. Checado em `colaborador.Criar` e `colaborador.Atualizar`
   via `repo.ExisteEmailNoLider(email, usuarioLiderID, excetoID)` (mesma Cadeia B de
   posse: `equipe.usuario_id` OU `organizacao.usuario_id`, com `deletado_em IS NULL` —
   então e-mail de liderado removido pode ser reutilizado). Erro: `ErrEmailDuplicado`.
2. **O e-mail do liderado NÃO pode ser o da conta do próprio gestor.** Senão, no aceite
   de convite, o liderado assumiria a conta do líder (sequestro). Checado via
   `repo.EmailEhDoLider(email, usuarioLiderID)` (consulta `tb_usuarios`). Erro:
   `ErrEmailDoGestor`. Vale no `Criar` e no `Atualizar`.
3. **Nome de equipe é único por líder** (`usuario_id`), case-insensitive (normalizado
   com `LOWER(TRIM())`/`strings.ToLower(TrimSpace())`), ignorando equipes deletadas.
   Checado em `equipe.Criar`/`equipe.Atualizar` via `repo.ExistePorNome`. Erro fixo
   `"já existe uma equipe com este nome"` → 409. O nome é trimado ao salvar.

> Ambos os módulos seguem o padrão de e-mail duplicado do `usuario` (mensagem fixa
> comparada no controller para escolher 409). Ao adicionar novas regras de unicidade,
> sempre filtre `deletado_em IS NULL` e escope pelo **líder dono**.

**Status atual da remediação:** ✅ **todos os módulos com IDOR foram corrigidos** —
`colaborador`, `blocotema`, `classificacao`, `agendamento`, `convite`, `onebyone`,
`registroonebyone`, `valorregistro`, `template`, `templatebloco`. Cadeias usadas:
`registroonebyone`/`valorregistro` herdam posse via `onebyone.PertenceAoUsuario`;
`templatebloco` via `template.PertenceAoUsuario`. 🔜 hardening restante (não-IDOR):
`blocotema` delete físico → soft (precisa migration `deletado_em`); reforço de
`WHERE usuario_id` nos UPDATE/DELETE (defesa em profundidade); rate-limit no convite.

---

## 8. Configuração e variáveis de ambiente

Lidas em `pkg/config/config.go`. **Segredos nunca são versionados** — só os `*.example`.

### Perfis de ambiente (dev/prod) — no estilo Spring `application-{perfil}.yml`

`AMBIENTE` é o **seletor de perfil** (como o `spring.profiles.active`). Vem do **SO / docker-compose**
(nunca do próprio arquivo) e o `config.Carregar()` escolhe qual arquivo de variáveis carregar:

| Perfil | `AMBIENTE` | Arquivo | Equivale a | Onde é usado |
|---|---|---|---|---|
| Desenvolvimento | `desenvolvimento` (padrão) | **`.env.dev`** | `application-dev.yml` | `go run` local + `docker-compose.yml` |
| Produção | `producao` | **`.env`** | `application.yml` | `docker-compose.prod.yml` (já seta `AMBIENTE=producao`) |

- Variáveis já no ambiente (container/compose) **sempre vencem** o arquivo. Se o arquivo do
  perfil não existir, cai no `.env` genérico (retrocompatível).
- Templates **versionados**: `.env.dev.example` (dev) e `../.env.example` (prod). Para começar:
  `cp .env.dev.example .env.dev` e preencha.
- `AMBIENTE=producao` liga as travas de segurança (JWT ≥32, `APP_URL` https, origem do WS
  restrita, Gin release, rejeição do JWT de exemplo).

| Variável               | Para quê                                  | Local (`go run`) | Container |
|------------------------|-------------------------------------------|------------------|-----------|
| `DB_HOST`              | host do MySQL                             | `localhost`      | `mysql`   |
| `DB_PORT`              | porta do MySQL                            | `3306`           | `3306`    |
| `DB_USER` / `DB_PASSWORD` | credenciais do banco                   | (ver `.env`)     | idem      |
| `DB_NAME`              | schema                                    | `onebyone`       | `onebyone`|
| `JWT_SECRET`           | chave de assinatura do token              | obrigatório      | obrigatório |
| `JWT_EXPIRACAO_HORAS`  | validade do token (padrão 24)             | opcional         | opcional  |
| `PORTA_API`            | porta HTTP (projeto usa 8090)             | opcional         | opcional  |
| `AWS_ACCESS_KEY_ID` …  | credenciais e bucket S3 das fotos         | obrigatório p/ fotos | idem  |
| `AMBIENTE`             | perfil (`desenvolvimento`/`producao`)     | `desenvolvimento`| `producao` |
| `RECAPTCHA_SITE_KEY` / `RECAPTCHA_SECRET` | reCAPTCHA anti-bot — **vazio = desligado** | opcional | opcional |

> No container, o Compose sobrescreve `DB_HOST` (`localhost`→`mysql`) e define o `AMBIENTE`
> (perfil). O restante vem do `.env.dev` (dev) ou `.env` (prod). O **reCAPTCHA** fica dormente
> até você preencher as duas chaves do Google — então ele exige a verificação no **login**,
> no **cadastro** e na recuperação de senha, sem precisar rebuildar o front.

---

## 9. Tarefas comuns

```bash
# Subir tudo (API + banco) em container
docker compose up --build

# Subir só o banco e rodar a API localmente
docker compose up -d mysql && go run ./cmd/api

# Rodar os testes (quando existirem)
go test ./...

# Formatar e checar o código
go fmt ./... && go vet ./...

# Baixar/limpar dependências
go mod tidy

# Regenerar a documentação Swagger após mexer nos comentários // @...
swag init -g cmd/api/main.go -o docs
```

### Para criar um módulo novo (passo a passo)

1. Crie `internal/<nome>/` com os 6 arquivos do padrão (seção 6).
2. Escreva a migration em `migrations/00X_*.sql`.
3. No `rotas.go`, adicione o trio (`Novo...`) + `RegistrarRotas`, na ordem certa
   se houver dependência entre módulos.
4. Anote os comentários Swagger nos handlers e rode `swag init`.

---

## 10. Estado atual e próximos passos

- **Backend:** funcional e completo para o escopo atual (CRUDs dos 10 módulos,
  auth JWT, upload de foto no S3, auditoria, Swagger). Tudo containerizado.
- **Próximo:** criar o `onebyone-app` (React) — frontend **inovador**, sem cara
  de "app de IA". Será documentado em CLAUDE.md próprio quando começar.

---

## 11. Lembretes para o Claude Code

- Responda e comente **em português**. Funções, variáveis e parâmetros em português.
- Documente bem e comente o código — o dono do projeto vem de **WebForms/.NET 4.8**
  e está aprendendo Go; prefira clareza a esperteza.
- **Docker para tudo.** Ao propor como rodar/testar, use o Compose.
- Não invente: confira o código antes de afirmar. Mantenha os padrões das seções 6 e 7.
- Ao mexer em rotas, atualize os comentários Swagger.

## 12. Catálogo completo — funcionalidades, regras e validações

O catálogo completo (todas as rotas, regras de negócio e validações de cada módulo,
**gerado a partir do código**) vive em arquivo próprio, para manter este guia leve:

➡️ **[docs/CATALOGO.md](docs/CATALOGO.md)**

**Camadas que se completam:** este **CLAUDE.md** (visão geral, sempre em contexto) →
**[docs/CATALOGO.md](docs/CATALOGO.md)** (índice de tudo) → **`internal/<modulo>/README.md`**
(detalhe de um módulo). Ao mudar regra/rota/validação, atualize o README do módulo e o catálogo.
