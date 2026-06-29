# Módulo `colaborador`

> Representa um membro de equipe (o "liderado" ou eventualmente o gestor) dentro de uma organização no OneByOne, sobre quem acontecem as reuniões 1:1.

## O que faz

Este módulo é responsável pelo cadastro e gerenciamento dos colaboradores. Um colaborador pertence sempre a uma **organização** e a uma **equipe**, e pode opcionalmente estar vinculado a uma conta de sistema (`usuario_id`) — ou seja, nem todo colaborador tem login na plataforma. O módulo oferece operações de criar, buscar, listar (por equipe ou por organização), atualizar, excluir (exclusão lógica) e fazer upload da foto de perfil para o S3.

Se você vem do .NET Framework, pense neste módulo como um conjunto organizado em camadas bem separadas: o `Controller` é como um `ApiController` (recebe HTTP), o `UseCase` é a camada de regras de negócio (como uma classe de "Service"), o `Repositorio` é o acesso a dados (como um Repository/DAO com ADO.NET), e os `DTO`/`entity` separam o modelo HTTP do modelo de banco.

## Arquivos

| Arquivo | Camada | Responsabilidade |
| --- | --- | --- |
| `controller.go` | Apresentação (HTTP) | Registra as rotas, lê o corpo/parâmetros da requisição, valida o arquivo de foto e delega ao `UseCase`. Nunca acessa o banco. |
| `dto.go` | Apresentação (contratos) | Define os DTOs de entrada (`CriarColaboradorDTO`, `AtualizarColaboradorDTO`) e de saída (`ColaboradorRespostaDTO`), com as validações de `binding`. |
| `usecase.go` | Regras de negócio | Valida e converte dados, gera o UUID, monta a chave do S3, faz upload e gera a URL presignada da foto. Intermedia Controller e Repositório. |
| `entity.go` | Domínio / banco | Define a struct `Colaborador`, que mapeia diretamente as colunas da tabela `tb_colaboradores`. |
| `repository.go` | Acesso a dados | Define a interface `Repositorio` e sua implementação MySQL (`repositorioMySQL`). Toda query SQL fica aqui. |
| `mapper.go` | Conversão | Converte a entidade `Colaborador` no DTO de resposta `ColaboradorRespostaDTO`. |

## Entidade e tabela no banco

A struct `Colaborador` (em `entity.go`) mapeia a tabela **`tb_colaboradores`** no MySQL. As tags `db:"..."` indicam o nome da coluna correspondente (semelhante a um atributo `[Column]` no Entity Framework).

| Campo (Go) | Tipo Go | Coluna | Significado |
| --- | --- | --- | --- |
| `ID` | `string` | `id` | Identificador único do colaborador, no formato UUID v4. |
| `UsuarioID` | `*string` | `usuario_id` | UUID da conta de sistema do colaborador. É um ponteiro (`*string`), então pode ser **nulo** — colaborador sem login. |
| `OrganizacaoID` | `string` | `organizacao_id` | UUID da organização à qual o colaborador pertence. |
| `EquipeID` | `string` | `equipe_id` | UUID da equipe à qual o colaborador pertence. |
| `TemplateID` | `*string` | `template_id` | UUID do template exclusivo deste colaborador (prioridade máxima na herança de template). Pode ser nulo. |
| `Nome` | `string` | `nome` | Nome completo do colaborador. |
| `Email` | `string` | `email` | E-mail do colaborador (não necessariamente usado para login). |
| `Whatsapp` | `*string` | `whatsapp` | Número de WhatsApp com DDD. Pode ser nulo. |
| `DataNascimento` | `*time.Time` | `data_nascimento` | Data de nascimento. Pode ser nula. |
| `CriadoEm` | `time.Time` | `criado_em` | Timestamp de criação do registro. |
| `AlteradoEm` | `*time.Time` | `alterado_em` | Timestamp da última modificação (nulo se nunca alterado). |
| `DeletadoEm` | `*time.Time` | `deletado_em` | Timestamp da exclusão lógica. **Nulo = registro ativo.** |
| `DeletadoPor` | `*string` | `deletado_por` | ID do usuário que realizou a exclusão lógica. |
| `FotoKey` | `*string` | `foto_key` | Chave do objeto no S3 (ex.: `colaboradores/uuid/foto.jpg`). Nulo quando sem foto. |

> Observação sobre ponteiros: em Go, um `*string` (ponteiro para string) é a forma idiomática de representar um valor que pode ser **NULL** no banco — equivale aproximadamente a um `string?` / `Nullable` no C#. Quando o ponteiro é `nil`, o valor é nulo.

## Endpoints

Todas as rotas são registradas em `RegistrarRotas` (em `controller.go`) sob o prefixo base **`/api/v1`** (definido em `cmd/api/rotas.go`). **Todas exigem autenticação JWT** — o grupo aplica o `authMiddleware` (`BearerAuth`) antes dos handlers.

| Método | Rota | Descrição | Autenticação |
| --- | --- | --- | --- |
| `POST` | `/api/v1/colaboradores` | Cria um novo colaborador dentro de uma equipe e organização. | JWT + ApenasLider |
| `POST` | `/api/v1/importar-liderados` | **Import em lote (CSV)**: cria vários liderados numa equipe. Corpo: `{ organizacao_id, equipe_id, itens: [{nome,email}] }` (máx. 500). Valida linha a linha e retorna `{ criados, erros }` — uma linha ruim não derruba o lote. Reusa `Criar` (posse + e-mail único + anti-gestor). Rota **top-level** de propósito (evita colisão estático×`:id` no Gin). | JWT + ApenasLider |
| `GET` | `/api/v1/colaboradores/:id` | Retorna os dados de um colaborador ativo pelo UUID. | JWT |
| `PUT` | `/api/v1/colaboradores/:id` | Atualiza parcialmente os dados de um colaborador. | JWT |
| `DELETE` | `/api/v1/colaboradores/:id` | Realiza a exclusão lógica (soft delete) do colaborador. | JWT |
| `POST` | `/api/v1/colaboradores/:id/foto` | Faz upload da foto (multipart/form-data, campo `foto`) e retorna a URL presignada. | JWT |
| `GET` | `/api/v1/equipes/:id/colaboradores` | Lista todos os colaboradores ativos de uma equipe (rota aninhada). | JWT |
| `GET` | `/api/v1/organizacoes/:id/colaboradores` | Lista todos os colaboradores ativos de uma organização (rota aninhada). | JWT |

> Detalhe das rotas aninhadas: tanto `/equipes/:id/colaboradores` quanto `/organizacoes/:id/colaboradores` usam o parâmetro de rota chamado `:id` (e não `:equipeId`/`:organizacaoId`). No handler, o valor é lido com `ctx.Param("id")`. Os comentários `// @Router` do Swagger usam `{equipeId}`/`{organizacaoId}` apenas a título de documentação, mas o nome real do parâmetro registrado é `id`.

## DTOs

Os DTOs ficam em `dto.go`. As validações usam as tags `binding` do Gin (similar a Data Annotations no .NET, como `[Required]`, `[StringLength]`, `[EmailAddress]`).

### Entrada: `CriarColaboradorDTO` (corpo do `POST /colaboradores`)

| Campo JSON | Tipo | Validação (`binding`) | Observação |
| --- | --- | --- | --- |
| `organizacao_id` | `string` | `required` | Obrigatório. |
| `equipe_id` | `string` | `required` | Obrigatório. |
| `nome` | `string` | `required,min=2,max=100` | Obrigatório, entre 2 e 100 caracteres. |
| `email` | `string` | `required,email,max=150` | Obrigatório, formato de e-mail, máx. 150 caracteres. |
| `usuario_id` | `*string` | `omitempty` | Opcional. |
| `template_id` | `*string` | `omitempty` | Opcional (template exclusivo, máxima prioridade). |
| `whatsapp` | `*string` | `omitempty,max=20` | Opcional, máx. 20 caracteres. |
| `data_nascimento` | `string` | `omitempty` | Opcional, formato `YYYY-MM-DD` (validado no UseCase). |

### Entrada: `AtualizarColaboradorDTO` (corpo do `PUT /colaboradores/:id`)

Todos os campos são opcionais; apenas os informados são atualizados (atualização parcial).

| Campo JSON | Tipo | Validação (`binding`) | Observação |
| --- | --- | --- | --- |
| `nome` | `string` | `omitempty,min=2,max=100` | Novo nome. |
| `email` | `string` | `omitempty,email,max=150` | Novo e-mail. |
| `equipe_id` | `string` | `omitempty` | UUID da nova equipe (transferência). |
| `usuario_id` | `*string` | `omitempty` | Conta de sistema (envie `null` para desvincular). |
| `template_id` | `*string` | `omitempty` | Novo template exclusivo (envie `null` para remover). |
| `whatsapp` | `*string` | `omitempty,max=20` | Novo WhatsApp. |
| `data_nascimento` | `string` | `omitempty` | Nova data, formato `YYYY-MM-DD`. |

### Saída: `ColaboradorRespostaDTO`

Retornado por todas as operações (montado pelo `mapper.ParaRespostaDTO`).

| Campo JSON | Tipo | Observação |
| --- | --- | --- |
| `id` | `string` | Identificador do colaborador. |
| `usuario_id` | `*string` | Conta de sistema (`null` se não vinculado). |
| `organizacao_id` | `string` | UUID da organização. |
| `equipe_id` | `string` | UUID da equipe. |
| `template_id` | `*string` | Template exclusivo (`null` se não configurado). |
| `nome` | `string` | Nome completo. |
| `email` | `string` | E-mail. |
| `whatsapp` | `*string` | WhatsApp (`null` se não informado). |
| `data_nascimento` | `*time.Time` | Data de nascimento (`null` se não informada). |
| `criado_em` | `time.Time` | Data/hora de criação. |
| `alterado_em` | `*time.Time` | Data/hora da última modificação. |
| `foto_url` | `*string` | URL presignada **temporária** da foto (`null` se sem foto; expira em 2h). Note que a resposta expõe a URL pronta para uso, e **não** a chave bruta do S3 (`foto_key`). |

## Regras de negócio

Implementadas em `usecase.go`:

- **Geração de identidade**: ao criar, o UUID v4 do colaborador é gerado no servidor (`uuid.New().String()`), nunca enviado pelo cliente. O campo `criado_em` é preenchido com `time.Now()`.
- **Validação da data de nascimento**: tanto em `Criar` quanto em `Atualizar`, se `data_nascimento` for informada, ela é convertida de string para data usando o layout `"2006-01-02"` (formato `YYYY-MM-DD`). Se o formato for inválido, retorna erro `"data de nascimento inválida — use o formato YYYY-MM-DD"`.
- **Atualização parcial (patch)**: `Atualizar` carrega primeiro o registro atual e só sobrescreve os campos enviados. Para os campos de string simples (`nome`, `email`, `equipe_id`, `data_nascimento`), a regra é "atualiza se diferente de vazio" (`!= ""`). Para os campos ponteiro (`usuario_id`, `template_id`, `whatsapp`), a regra é "atualiza se `!= nil`", o que permite enviar explicitamente `null` no JSON para limpar/desvincular o valor.
- **Soft delete (exclusão lógica)**: `Deletar` primeiro confirma que o colaborador existe (`BuscarPorId`) e então chama `DeletarSoft`, que preenche `deletado_em` e `deletado_por` em vez de remover a linha. O `deletado_por` vem do ID do usuário autenticado, extraído do contexto JWT (`middleware.ChaveUsuarioID`) no controller. Todas as queries de leitura filtram por `deletado_em IS NULL`, então registros deletados ficam invisíveis para a API.
- **Upload de foto**: validado em duas etapas. No `controller.go`, o tamanho é limitado a **5MB** (`http.MaxBytesReader`) e o `Content-Type` deve estar na lista `tiposImagemPermitidos` (`image/jpeg`, `image/png`, `image/webp`). No `usecase.go`, antes de subir o arquivo, confirma que o colaborador existe; a extensão do arquivo é derivada do tipo de conteúdo (`extensaoPorTipo`: `.jpg`/`.png`/`.webp`, com fallback `.jpg`); o caminho `colaboradores/{id}/foto{ext}` é então transformado na chave completa do S3 por `armazenamento.ChaveCompleta(...)` (que prefixa o caminho com o prefixo do projeto, ex.: `one-by-one/colaboradores/{id}/foto{ext}`), e é essa chave já prefixada que é usada no `Upload` e salva na coluna `foto_key`.
- **URL presignada da foto**: o campo `foto_url` da resposta é gerado sob demanda por `gerarFotoURL`. Se `foto_key` for nulo ou o serviço de armazenamento não estiver configurado, retorna `nil`. A URL presignada expira em **2 horas** (`storage.ExpiracaoURLFoto`). Em `Criar`, a foto ainda não existe, então a resposta sai com `foto_url = null`.
- **Ordenação das listagens**: `ListarPorEquipe` e `ListarPorOrganizacao` retornam os colaboradores ativos ordenados por `nome ASC`.
- **Não há hash de senha neste módulo**: o colaborador não gerencia credenciais; o vínculo com login é apenas a referência `usuario_id` para o módulo de usuários.

## Dependências

A injeção de dependências segue o padrão de construtores Go (funções `Novo...`), montados em `cmd/api/rotas.go`:

- **`NovoRepositorio(db *sqlx.DB) Repositorio`** — recebe o pool de conexões MySQL (`sqlx.DB`). É a única camada que conhece SQL e a tabela `tb_colaboradores`.
- **`NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento) UseCase`** — recebe:
  - `repo`: o repositório (via interface), para persistência.
  - `armazenamento`: o serviço de storage S3 (`storage.Armazenamento`, o `s3Svc`), usado para fazer upload da foto (`Upload`), montar a chave completa (`ChaveCompleta`) e gerar URLs presignadas (`GerarURLPresignada`). Receber a interface (e não uma implementação concreta) facilita testes e troca de provedor.
- **`NovoController(uc UseCase) *Controller`** — recebe apenas o `UseCase` (via interface). O controller não conhece banco nem S3 diretamente; ele apenas traduz HTTP para chamadas de negócio.

Ordem de montagem (em `cmd/api/rotas.go`): `colaboradorRepo := colaborador.NovoRepositorio(db)` → `colaboradorUseCase := colaborador.NovoUseCase(colaboradorRepo, s3Svc)` → `colaboradorController := colaborador.NovoController(colaboradorUseCase)` → `colaboradorController.RegistrarRotas(api, authMiddleware)`.
