# Módulo `organizacao`

> Representa a organização (empresa/área) que um líder cadastra no OneByOne para agrupar suas equipes e colaboradores antes de conduzir as reuniões 1:1.

## O que faz

Este módulo cuida do cadastro de organizações. Cada organização pertence a um líder (o usuário autenticado) e serve como o "contêiner" sob o qual ficam as equipes e os colaboradores. Ele oferece o CRUD completo (criar, listar, buscar por ID, atualizar e excluir), além do upload de uma foto que fica guardada no S3.

Se você vem do .NET, pense no módulo como um conjunto de camadas bem separadas, parecido com Controller / Service / Repository do ASP.NET MVC: o **Controller** trata o HTTP, o **UseCase** (equivalente ao Service) contém as regras de negócio, e o **Repository** conversa com o banco. Cada camada é uma `interface` Go (semelhante a uma interface C#) com uma implementação concreta, o que facilita testes e injeção de dependência.

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---------|--------|------------------|
| `controller.go` | Apresentação (HTTP) | Registra as rotas, lê o corpo/parâmetros da requisição, valida o upload de foto e delega ao UseCase. Nunca toca no banco. |
| `usecase.go` | Negócio | Contém as regras de negócio: gera UUID, vincula a organização ao líder, aplica atualização parcial, valida existência antes de deletar e orquestra o upload da foto para o S3. |
| `repository.go` | Dados | Interface e implementação MySQL. Toda query SQL contra a tabela `tb_organizacoes` mora aqui. |
| `entity.go` | Domínio | Define a struct `Organizacao`, que espelha as colunas da tabela no banco. |
| `dto.go` | Contratos | Define os DTOs de entrada (criar/atualizar) e de saída (resposta), desacoplando o modelo de banco da API HTTP. |
| `mapper.go` | Conversão | Função que converte a entidade `Organizacao` no DTO de resposta `OrganizacaoRespostaDTO`. |

## Entidade e tabela no banco

A struct `Organizacao` (em `entity.go`) mapeia diretamente a tabela **`tb_organizacoes`** do MySQL. As tags `db:"..."` indicam o nome da coluna correspondente (semelhante a um atributo `[Column]` no Entity Framework).

Em Go, um tipo prefixado com `*` (ex.: `*string`, `*time.Time`) é um **ponteiro** e pode ser `nil` — é o equivalente a um valor anulável (`string?`, `DateTime?` no C#). Quando a coluna do banco aceita `NULL`, o campo é um ponteiro.

| Campo | Tipo | Coluna | Significado |
|-------|------|--------|-------------|
| `ID` | `string` | `id` | Identificador único da organização, no formato UUID v4. |
| `UsuarioID` | `string` | `usuario_id` | UUID do líder proprietário desta organização. |
| `TemplateID` | `*string` (anulável) | `template_id` | UUID do template padrão da organização; pode ser nulo quando nenhum template foi configurado. |
| `Nome` | `string` | `nome` | Nome da organização. |
| `CriadoEm` | `time.Time` | `criado_em` | Data/hora de criação do registro. |
| `AlteradoEm` | `*time.Time` (anulável) | `alterado_em` | Data/hora da última modificação; nulo se nunca foi alterado. |
| `DeletadoEm` | `*time.Time` (anulável) | `deletado_em` | Data/hora da exclusão lógica (soft delete); nulo significa registro ativo. |
| `DeletadoPor` | `*string` (anulável) | `deletado_por` | ID do usuário responsável pela exclusão lógica. |
| `FotoKey` | `*string` (anulável) | `foto_key` | Chave do objeto da foto no S3 (ex.: `organizacoes/<uuid>/foto.jpg`); nulo quando a organização não tem foto. |

Observação: os campos `DeletadoEm` e `DeletadoPor` nunca são expostos diretamente na API; eles existem apenas para o controle de soft delete dentro do banco.

## Endpoints

Todas as rotas ficam sob o prefixo **`/api/v1`** e abaixo do grupo `/organizacoes`. **Todas exigem autenticação JWT** — no `controller.go`, o grupo aplica `orgs.Use(authMiddleware)`, então o middleware de autenticação roda antes de qualquer handler. O ID do líder é lido do token (via `middleware.ChaveUsuarioID`), não do corpo da requisição.

| Método | Rota | Descrição | Autenticação |
|--------|------|-----------|--------------|
| POST | `/api/v1/organizacoes` | Cria uma nova organização vinculada ao líder autenticado. | JWT obrigatório |
| GET | `/api/v1/organizacoes` | Lista todas as organizações ativas do líder autenticado (ordenadas por nome). | JWT obrigatório |
| GET | `/api/v1/organizacoes/:id` | Retorna os dados de uma organização ativa pelo UUID. | JWT obrigatório |
| PUT | `/api/v1/organizacoes/:id` | Atualiza parcialmente os dados de uma organização pelo UUID. | JWT obrigatório |
| DELETE | `/api/v1/organizacoes/:id` | Realiza o soft delete (exclusão lógica) de uma organização. | JWT obrigatório |
| POST | `/api/v1/organizacoes/:id/foto` | Faz upload da foto da organização (multipart/form-data, campo `foto`) e retorna a organização com a URL presignada. | JWT obrigatório |

## DTOs

DTOs (Data Transfer Objects) são structs usadas para entrada e saída na borda HTTP, separando o que trafega pela API do modelo de banco. A tag `binding:"..."` é a validação automática do Gin (equivalente a `[Required]`, `[StringLength]`, etc. dos DataAnnotations no .NET); ela roda quando o controller chama `ShouldBindJSON`.

### Entrada

**`CriarOrganizacaoDTO`** — corpo do POST de criação:

| Campo | Tipo JSON | Validação (`binding`) | Observação |
|-------|-----------|------------------------|------------|
| `nome` | `string` | `required, min=2, max=100` | Obrigatório; entre 2 e 100 caracteres. |
| `template_id` | `string` (anulável) | `omitempty` | Opcional; UUID do template padrão. |

**`AtualizarOrganizacaoDTO`** — corpo do PUT. Todos os campos são opcionais; apenas os informados são alterados:

| Campo | Tipo JSON | Validação (`binding`) | Observação |
|-------|-----------|------------------------|------------|
| `nome` | `string` | `omitempty, min=2, max=100` | Quando informado, deve ter de 2 a 100 caracteres. |
| `template_id` | `string` (anulável) | `omitempty` | Novo UUID do template. Só é alterado quando um valor é enviado; enviar `null` ou omitir o campo preserva o valor atual (não há como remover o template por este endpoint). |

### Saída

**`OrganizacaoRespostaDTO`** — formato retornado por todos os endpoints (exceto o DELETE, que devolve uma mensagem de texto):

| Campo | Tipo JSON | Significado |
|-------|-----------|-------------|
| `id` | `string` | Identificador único da organização. |
| `usuario_id` | `string` | UUID do líder proprietário. |
| `template_id` | `string` (anulável) | UUID do template padrão; `null` se não configurado. |
| `nome` | `string` | Nome da organização. |
| `criado_em` | `time.Time` | Data/hora de criação. |
| `alterado_em` | `time.Time` (anulável) | Data/hora da última modificação; `null` se nunca alterado. |
| `foto_url` | `string` (anulável) | URL presignada temporária da foto; `null` se sem foto. Expira em ~2h (`storage.ExpiracaoURLFoto`). |

Note que o DTO de resposta **não** expõe `foto_key`, `deletado_em` nem `deletado_por` — esses campos da entidade ficam só no banco.

## Regras de negócio

As regras estão concentradas em `usecase.go`:

- **Geração de identidade**: no `Criar`, o UUID v4 é gerado no servidor (`uuid.New()`); o cliente não envia o `id`.
- **Vínculo ao líder**: a organização é amarrada ao `usuarioID` extraído do token JWT — o controller pega esse valor do contexto autenticado, não do corpo da requisição. Garante que cada líder só cria organizações para si.
- **`criado_em` no servidor**: o timestamp de criação é definido com `time.Now()` no momento da criação.
- **Atualização parcial (PATCH-like)**: embora o verbo HTTP seja PUT, o `Atualizar` busca o registro atual e só substitui os campos enviados — `Nome` é trocado apenas se não vier vazio (`""`), e `TemplateID` é trocado apenas se o ponteiro não for `nil`. Como consequência, enviar `template_id: null` (que desserializa para ponteiro `nil`) **preserva** o valor atual em vez de removê-lo; não há como limpar o `template_id` por este endpoint. Os demais campos são preservados.
- **Verificação de existência antes de mutar**: `Atualizar`, `Deletar` e `UploadFoto` chamam `BuscarPorId` primeiro; se a organização não existir (ou estiver deletada), retornam erro de "não encontrada" antes de qualquer escrita.
- **Soft delete (exclusão lógica)**: `Deletar` nunca remove a linha do banco. Ele delega ao repositório (`DeletarSoft`), que preenche `deletado_em` e `deletado_por`. Todas as consultas (`BuscarPorId`, `ListarPorUsuario`, e os próprios UPDATEs) filtram por `deletado_em IS NULL`, então registros excluídos ficam invisíveis para a aplicação.
- **`alterado_em` automático**: o repositório define `alterado_em = time.Now()` em toda atualização (incluindo a de foto), sem o UseCase precisar gerenciar isso.
- **Upload de foto para o S3**:
  - O controller limita o arquivo a **5 MB** (`maxTamanho = 5 << 20`) e aceita apenas os tipos `image/jpeg`, `image/png` e `image/webp` (mapa `tiposImagemPermitidos`); qualquer outro tipo é rejeitado com erro de requisição.
  - A extensão do arquivo é derivada do `Content-Type` (`extensaoPorTipo`): `.jpg`, `.png` ou `.webp` (padrão `.jpg`).
  - A chave no S3 segue o padrão `organizacoes/<id>/foto<ext>`, ajustada pelo prefixo do bucket via `ChaveCompleta`.
  - O fluxo é: enviar ao S3 → persistir a chave em `foto_key` → recarregar a organização → devolver o DTO já com a `foto_url` presignada.
- **URL presignada temporária**: a foto nunca é exposta como link público. O `gerarFotoURL` produz, sob demanda, uma URL assinada e temporária (`GerarURLPresignada` com `storage.ExpiracaoURLFoto`, ~2h). Se não houver chave de foto, ou o serviço de armazenamento estiver indisponível, o campo volta como `null` (a falha ao gerar a URL não derruba a resposta).

> Não há regras de hash de senha neste módulo — esse tipo de tratamento pertence a outros módulos (ex.: usuário/autenticação).

## Dependências

Em Go, a injeção de dependência é feita explicitamente passando as dependências para funções construtoras (as `Novo...`), em vez de um contêiner de DI automático como no ASP.NET. A montagem das camadas acontece em `cmd/api/rotas.go`:

```
organizacaoRepo       := organizacao.NovoRepositorio(db)
organizacaoUseCase    := organizacao.NovoUseCase(organizacaoRepo, s3Svc)
organizacaoController := organizacao.NovoController(organizacaoUseCase)
organizacaoController.RegistrarRotas(api, authMiddleware)
```

| Construtor | Recebe | Para quê |
|------------|--------|----------|
| `NovoRepositorio(db *sqlx.DB)` | O pool de conexões MySQL (`*sqlx.DB`). | Executar as queries SQL contra `tb_organizacoes`. |
| `NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento)` | O repositório e o serviço de armazenamento (S3). | Aplicar as regras de negócio e fazer upload/geração de URLs de fotos. |
| `NovoController(uc UseCase)` | O UseCase. | Tratar o HTTP e delegar a lógica. |
| `RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc)` | O grupo de rotas `/api/v1` e o middleware de autenticação JWT. | Registrar os endpoints sob `/organizacoes`, todos protegidos por JWT. |

Cada dependência é recebida via **interface** (`Repositorio`, `UseCase`, `storage.Armazenamento`), e não pela implementação concreta. Isso é o equivalente Go de programar contra interfaces no C#: facilita substituir a implementação por um mock em testes e mantém as camadas desacopladas.
