# Módulo `auditoria`

> Registra a trilha de atividades dos usuários (gestores e liderados) que usam o OneByOne, guardando quem fez o quê, quando e de onde — tanto as operações de escrita no sistema quanto os eventos de navegação na tela.

## O que faz

Este módulo grava eventos de auditoria em uma tabela do MySQL e permite que cada usuário consulte sua própria linha do tempo de atividades. Ele atua de duas formas:

1. **Automaticamente**, por meio de um middleware global (`pkg/middleware/auditoria.go`) que intercepta toda requisição de escrita (POST, PUT, DELETE) bem-sucedida e grava um registro — por exemplo, ao criar uma organização ou atualizar um colaborador.
2. **Sob demanda do frontend**, recebendo eventos de interface (cliques, navegação, visualização de telas) por um endpoint próprio.

Para quem vem do .NET Framework: pense neste módulo como um `HttpModule`/`Action Filter` que loga as ações em uma tabela, mais um pequeno endpoint para o JavaScript do front mandar eventos de tela. A diferença importante de Go aqui é que a gravação acontece em uma **goroutine** (linha de execução leve em segundo plano), então o registro não atrasa a resposta HTTP ao usuário — é um "fire and forget".

A arquitetura segue o padrão em camadas usado em todo o projeto: **Controller** (HTTP) → **UseCase** (regras) → **Repositorio** (banco), conversando por **interfaces** (contratos), o que equivale a programar contra interfaces com injeção de dependência no .NET.

## Arquivos

| Arquivo | Camada | Responsabilidade |
| --- | --- | --- |
| `entity.go` | Entidade / Modelo de dados | Define a struct `Auditoria`, que espelha a tabela `tb_auditoria` do banco (tags `db`). |
| `dto.go` | DTOs (contratos da API) | Define `EventoDTO` (entrada vinda do frontend) e `AuditoriaRespostaDTO` (saída devolvida pela API). |
| `repository.go` | Repositório (acesso a dados) | Interface `Repositorio` e implementação MySQL (`sqlx`) com `Gravar` e `ListarPorUsuario`. |
| `usecase.go` | Caso de uso (regras de negócio) | Interface `UseCase` e implementação: gravação assíncrona, limite de paginação e conversão entidade → DTO. |
| `controller.go` | Controlador (HTTP / Gin) | Registra as rotas e trata as requisições `RegistrarEvento` e `MinhaTrilha`. |

## Entidade e tabela no banco

A struct `Auditoria` (em `entity.go`) representa uma linha da tabela **`tb_auditoria`** no MySQL (o nome da tabela aparece nas queries do `repository.go`).

Observação sobre os tipos: campos declarados como `*string` (ponteiro para string) podem ser **nulos** (`NULL` no banco). Isso é o equivalente em Go de um `string` que aceita `null` no C# — quando o ponteiro é `nil`, não há valor.

| Campo (Go) | Tipo | Coluna (`db`) | Significado |
| --- | --- | --- | --- |
| `ID` | `string` | `id` | Identificador único do registro (UUID gerado na aplicação). |
| `UsuarioID` | `*string` | `usuario_id` | UUID do usuário que originou o evento. Pode ser nulo (ex.: ação sem usuário autenticado identificado). |
| `Acao` | `string` | `acao` | O que aconteceu (ex.: `CRIAR`, `ATUALIZAR`, `DELETAR`, `LOGIN`, `UPLOAD_FOTO`, ou ações de UI como `VISUALIZAR`). |
| `Entidade` | `string` | `entidade` | Contexto/recurso afetado (ex.: `organizacao`, `equipe`, `colaborador`, ou no front `tela_organizacoes`). |
| `EntidadeID` | `*string` | `entidade_id` | UUID do recurso relacionado, quando existir. Pode ser nulo. |
| `IP` | `*string` | `ip` | Endereço IP do cliente. Pode ser nulo. |
| `UserAgent` | `*string` | `user_agent` | Cabeçalho `User-Agent` do navegador/cliente. Pode ser nulo. |
| `CriadoEm` | `time.Time` | `criado_em` | Data e hora em que o evento foi registrado. |

As consultas SQL reais usadas:

- **Inserção** (`Gravar`): `INSERT INTO tb_auditoria (id, usuario_id, acao, entidade, entidade_id, ip, user_agent, criado_em) VALUES (...)`.
- **Listagem** (`ListarPorUsuario`): `SELECT ... FROM tb_auditoria WHERE usuario_id = ? ORDER BY criado_em DESC LIMIT ?` — ou seja, sempre filtra pelo usuário e devolve os mais recentes primeiro.

## Endpoints

As rotas são registradas em `controller.go`, na função `RegistrarRotas`, sob o grupo `/auditoria`. Esse grupo é montado dentro do prefixo global `/api/v1` (definido em `cmd/api/rotas.go`), resultando nos caminhos completos abaixo. O grupo aplica `authMiddleware`, então **ambas as rotas exigem JWT** (cabeçalho `Authorization: Bearer <token>`).

| Método | Rota | Descrição | Autenticação |
| --- | --- | --- | --- |
| POST | `/api/v1/auditoria/eventos` | O frontend envia um evento de UI (navegação, clique, visualização) para compor a trilha de atividade. Responde `200` com a mensagem `"evento registrado"`. | JWT obrigatório |
| GET | `/api/v1/auditoria/minha` | Retorna os últimos eventos do usuário autenticado (mais recentes primeiro). Aceita o parâmetro de query `limite` (padrão 50, máximo 200). | JWT obrigatório |

Detalhes úteis:

- Em `/auditoria/eventos`, o `usuario_id`, o `IP` (`ctx.ClientIP()`) e o `User-Agent` são preenchidos pelo servidor a partir do contexto e dos cabeçalhos — o frontend não precisa (nem deve) enviá-los.
- Em `/auditoria/minha`, o parâmetro `limite` é lido da query string; se não vier ou for inválido, assume-se 50 (validação adicional no UseCase, ver abaixo).
- O usuário autenticado é obtido do contexto pela chave `middleware.ChaveUsuarioID` (constante `"usuario_id"`), populada pelo middleware de autenticação.

## DTOs

### Entrada — `EventoDTO`

Payload JSON enviado pelo frontend. As tags `binding` são validações automáticas do Gin (equivalente a Data Annotations / `[Required]` no .NET); se falharem, a API responde `400`.

| Campo JSON | Tipo | Obrigatório | Validações (`binding`) | Descrição |
| --- | --- | --- | --- | --- |
| `acao` | `string` | Sim | `required`, `max=50` | O que aconteceu (ex.: `VISUALIZAR`, `CLICAR`, `NAVEGAR`). |
| `entidade` | `string` | Sim | `required`, `max=100` | Contexto do evento (ex.: `tela_organizacoes`, `btn_criar_equipe`). |
| `entidade_id` | `*string` | Não | `omitempty` | UUID do recurso relacionado (opcional). |

### Saída — `AuditoriaRespostaDTO`

Objeto devolvido pela API em `/auditoria/minha`. Espelha a entidade, mas exposto com nomes de campo em JSON.

| Campo JSON | Tipo | Descrição |
| --- | --- | --- |
| `id` | `string` | Identificador do registro. |
| `usuario_id` | `string` (anulável) | UUID do usuário. |
| `acao` | `string` | Ação registrada. |
| `entidade` | `string` | Entidade/contexto. |
| `entidade_id` | `string` (anulável) | UUID do recurso relacionado. |
| `ip` | `string` (anulável) | IP do cliente. |
| `user_agent` | `string` (anulável) | User-Agent do cliente. |
| `criado_em` | `string` (data/hora) | Momento do registro. |

## Regras de negócio

Implementadas em `usecase.go`:

- **Gravação assíncrona (não bloqueia a resposta):** o método `Registrar` monta a entidade e dispara a gravação em uma **goroutine** (`go func() { _ = uc.repo.Gravar(a) }()`). A requisição HTTP é respondida sem esperar o banco. Como consequência intencional, **um eventual erro de gravação é ignorado** (não retorna ao cliente nem interrompe a operação principal) — auditoria nunca deve quebrar o fluxo do usuário.
- **Geração do ID:** todo registro recebe um `id` UUID novo (`uuid.New().String()`) gerado na aplicação, não no banco.
- **Carimbo de tempo:** o `CriadoEm` é preenchido com `time.Now()` no momento do registro.
- **Normalização de strings vazias para nulo:** a função auxiliar `strPtr` converte `IP` e `User-Agent` vazios (`""`) em `nil`, gravando `NULL` no banco em vez de string vazia.
- **Limite de paginação seguro:** em `ListarPorUsuario`, se o `limite` for menor ou igual a 0 **ou** maior que 200, ele é forçado para 50. Isso evita consultas sem teto e protege o banco de pedidos abusivos.
- **Conversão entidade → DTO:** ao listar, cada `Auditoria` (modelo de banco) é mapeada manualmente para um `AuditoriaRespostaDTO` (contrato da API), mantendo separadas a camada de dados e a camada de exposição.
- **Evento de UI sempre amarrado ao usuário:** `RegistrarEvento` recebe o `usuarioID` como `string` e o transforma em ponteiro para reaproveitar o `Registrar`, garantindo que o evento de tela fique vinculado a quem está logado.

Regras complementares no **middleware global** (`pkg/middleware/auditoria.go`), que usa este UseCase:

- Só audita automaticamente requisições **POST, PUT e DELETE** (escrita); leituras `GET` não são auditadas pelo middleware.
- **Ignora respostas com erro** (status HTTP >= 400) — só registra operações bem-sucedidas.
- Traduz método HTTP em ação legível: POST → `CRIAR`, PUT → `ATUALIZAR`, DELETE → `DELETAR`; e casos especiais como `login` → `LOGIN`, `registrar` → `CRIAR` (usuário), `foto` → `UPLOAD_FOTO`.
- **Evita auditoria dupla:** o caminho `eventos` é explicitamente ignorado pelo middleware, porque esses eventos já são registrados pelo próprio endpoint `/auditoria/eventos`.
- Deriva o nome da `entidade` a partir do path da rota (ex.: `organizacoes` → `organizacao`, `equipes` → `equipe`) e tenta extrair o `entidade_id` do parâmetro `:id` da URL.

## Dependências

O módulo é montado por injeção de dependência em `cmd/api/rotas.go`, encadeando os três construtores:

| Construtor | Recebe | Por quê |
| --- | --- | --- |
| `NovoRepositorio(db *sqlx.DB)` | A conexão com o banco MySQL (`*sqlx.DB`). | É a camada que efetivamente executa os SQLs de `INSERT` e `SELECT` na tabela `tb_auditoria`. |
| `NovoUseCase(repo Repositorio)` | O repositório (via interface `Repositorio`). | O caso de uso aplica as regras (gravação assíncrona, limites, conversão para DTO) e delega a persistência ao repositório, sem conhecer detalhes do MySQL. |
| `NovoController(uc UseCase)` | O caso de uso (via interface `UseCase`). | O controlador apenas traduz HTTP ↔ DTO e chama o caso de uso; não acessa banco diretamente. |

Cada camada depende de uma **interface** da camada de baixo, não da implementação concreta — o mesmo princípio de "depender de abstrações" do .NET, o que facilita testes com mocks.

Além disso, o `UseCase` deste módulo é compartilhado com o **middleware global de auditoria**: em `cmd/api/rotas.go`, o mesmo `auditoriaUseCase` é passado para `middleware.RegistrarAuditoria(...)`. O middleware depende apenas da interface mínima `AuditoriaUseCase` (com o método `Registrar`), definida no próprio pacote de middleware para evitar importação circular entre os pacotes.
