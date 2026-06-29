# Módulo `equipe`

> Representa uma equipe (subgrupo de colaboradores) dentro de uma organização no OneByOne — é o agrupamento sob um líder/gestor que serve de contexto para as reuniões 1:1 entre gestor e liderados.

## O que faz

Este módulo cuida do cadastro e da manutenção das equipes. Cada equipe pertence a uma organização e tem um líder responsável (o usuário autenticado que a criou). O módulo permite criar, buscar, listar (por líder ou por organização), atualizar, fazer upload da foto da equipe e excluir logicamente (soft delete).

Para quem vem do .NET: pense neste módulo como um conjunto de classes que cumprem papéis bem separados, parecido com uma arquitetura em camadas. O `Controller` é como um ApiController (recebe o HTTP); o `UseCase` é a camada de serviço/regra de negócio; o `Repositorio` é o acesso a dados (como um Repository sobre o banco). Cada camada conversa com a próxima por meio de uma `interface` (equivalente a uma interface C#), o que facilita testar e trocar implementações.

## Arquivos

| Arquivo | Camada | Responsabilidade |
|---|---|---|
| `entity.go` | Entidade / Modelo de banco | Define a struct `Equipe`, que mapeia diretamente a tabela `tb_equipes`. |
| `dto.go` | DTOs (contratos HTTP) | Define os objetos de entrada e saída da API, isolando o modelo de banco da camada HTTP. |
| `mapper.go` | Conversor | Função `ParaRespostaDTO` que converte a entidade `Equipe` no DTO de resposta. |
| `repository.go` | Acesso a dados | Interface `Repositorio` e implementação MySQL (`repositorioMySQL`). Único lugar que toca o banco. |
| `usecase.go` | Regra de negócio | Interface `UseCase` e implementação (`useCaseImpl`). Validações, geração de UUID, orquestração do upload de foto. |
| `controller.go` | HTTP / Apresentação | Registra as rotas Gin, lê a requisição, chama o `UseCase` e devolve a resposta. Nunca acessa o banco diretamente. |

## Entidade e tabela no banco

A struct `Equipe` (em `entity.go`) mapeia a tabela **`tb_equipes`** no MySQL. As tags `db:"..."` indicam o nome da coluna correspondente (semelhante a atributos de mapeamento de um ORM no .NET).

| Campo Go | Tipo Go | Coluna | Significado |
|---|---|---|---|
| `ID` | `string` | `id` | Identificador único da equipe, no formato UUID v4. |
| `UsuarioID` | `string` | `usuario_id` | UUID do líder responsável pela equipe. |
| `OrganizacaoID` | `string` | `organizacao_id` | UUID da organização à qual a equipe pertence. |
| `TemplateID` | `*string` | `template_id` | UUID do template padrão da equipe. É um ponteiro (`*string`), ou seja, pode ser nulo. Quando preenchido, sobrescreve o template da organização. |
| `Nome` | `string` | `nome` | Nome da equipe. |
| `CriadoEm` | `time.Time` | `criado_em` | Data/hora de criação do registro. |
| `AlteradoEm` | `*time.Time` | `alterado_em` | Data/hora da última modificação. Nulo (`nil`) se nunca foi alterado. |
| `DeletadoEm` | `*time.Time` | `deletado_em` | Data/hora da exclusão lógica. Nulo significa que o registro está ativo. |
| `DeletadoPor` | `*string` | `deletado_por` | UUID do usuário que executou a exclusão lógica. Nulo se ativo. |
| `FotoKey` | `*string` | `foto_key` | Chave do objeto da foto no S3 (ex.: `equipes/uuid/foto.jpg`). Nulo quando a equipe não tem foto. |

Observação sobre ponteiros: em Go, um tipo como `*string` é usado para representar um valor que pode estar ausente (`nil`), papel parecido com o de um `string?` / `Nullable<T>` no C#.

## Endpoints

Todas as rotas ficam sob o prefixo **`/api/v1`** e exigem autenticação JWT (o grupo aplica o `authMiddleware` em todas as rotas registradas em `RegistrarRotas`).

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| POST | `/api/v1/equipes` | Cria uma nova equipe vinculada ao líder autenticado. | JWT obrigatório |
| GET | `/api/v1/equipes` | Lista todas as equipes ativas do líder autenticado. | JWT obrigatório |
| GET | `/api/v1/equipes/:id` | Busca uma equipe ativa pelo UUID. | JWT obrigatório |
| PUT | `/api/v1/equipes/:id` | Atualiza parcialmente uma equipe (nome e/ou template). | JWT obrigatório |
| DELETE | `/api/v1/equipes/:id` | Exclusão lógica (soft delete) da equipe. | JWT obrigatório |
| POST | `/api/v1/equipes/:id/foto` | Upload da foto da equipe (multipart/form-data). | JWT obrigatório |
| GET | `/api/v1/organizacoes/:id/equipes` | Lista todas as equipes ativas de uma organização específica. | JWT obrigatório |

Notas:

- O `:id` na rota aninhada `/organizacoes/:id/equipes` representa o UUID da **organização** (o grupo reaproveita o parâmetro `:id` por consistência com o grupo `/organizacoes` já existente). Nos comentários Swagger (`// @Router`) ele aparece documentado como `{organizacaoId}`, mas no código Gin o parâmetro lido é `ctx.Param("id")`.
- No upload de foto, o arquivo no corpo do formulário (multipart) deve usar o campo `foto`. Apenas os tipos `image/jpeg`, `image/png` e `image/webp` são aceitos (validados pelo `Content-Type` do arquivo, antes de chamar o `UseCase`). O tamanho máximo de **5 MB** (`5 << 20` bytes) é imposto por `http.MaxBytesReader`, que limita o corpo da requisição: o erro de tamanho excedido só ocorre durante a leitura do arquivo (no envio ao S3, dentro do `UseCase`), não em uma verificação prévia explícita.

## DTOs

Os DTOs ficam em `dto.go`. As tags `binding:"..."` são regras de validação aplicadas automaticamente pelo Gin ao desserializar o JSON (equivalente aos Data Annotations / validação de ModelState no ASP.NET).

### Entrada — `CriarEquipeDTO` (POST `/equipes`)

| Campo JSON | Tipo | Validação (`binding`) | Observação |
|---|---|---|---|
| `organizacao_id` | `string` | `required` | UUID da organização (obrigatório). |
| `nome` | `string` | `required,min=2,max=100` | Nome da equipe, de 2 a 100 caracteres. |
| `template_id` | `*string` | `omitempty` | UUID do template padrão (opcional; pode ser nulo). |

### Entrada — `AtualizarEquipeDTO` (PUT `/equipes/:id`)

Todos os campos são opcionais; apenas os informados são atualizados.

| Campo JSON | Tipo | Validação (`binding`) | Observação |
|---|---|---|---|
| `nome` | `string` | `omitempty,min=2,max=100` | Novo nome (opcional). Se vier vazio, o nome atual é preservado. |
| `template_id` | `*string` | `omitempty` | Novo UUID do template (opcional). Enviar `null` substitui pelo valor nulo. |

### Saída — `EquipeRespostaDTO`

Retornado por praticamente todos os endpoints (exceto o DELETE, que devolve uma string de confirmação).

| Campo JSON | Tipo | Observação |
|---|---|---|
| `id` | `string` | Identificador único da equipe. |
| `usuario_id` | `string` | UUID do líder responsável. |
| `organizacao_id` | `string` | UUID da organização. |
| `template_id` | `*string` | UUID do template padrão (`null` se não configurado). |
| `nome` | `string` | Nome da equipe. |
| `criado_em` | `time.Time` | Data/hora de criação. |
| `alterado_em` | `*time.Time` | Data/hora da última modificação (`null` se nunca alterado). |
| `foto_url` | `*string` | URL presignada temporária da foto (`null` se sem foto; expira em 2 horas). |

## Regras de negócio

As regras ficam em `usecase.go` (implementação `useCaseImpl`):

- **Geração do ID e vínculos na criação**: ao criar, o `UseCase` gera o UUID v4 (`uuid.New().String()`), define `CriadoEm` com a hora atual e vincula a equipe ao `usuario_id` do líder autenticado (recebido do JWT, não do corpo da requisição) e à `organizacao_id` informada no DTO.
- **Foto não é definida na criação**: ao criar a equipe, o DTO de resposta sai com `foto_url` igual a `nil` — a foto só é adicionada depois, pelo endpoint de upload.
- **Atualização parcial (preserva os demais campos)**: o `Atualizar` primeiro busca a equipe atual; só sobrescreve `Nome` se o DTO trouxe nome não vazio, e só sobrescreve `TemplateID` se ele não for `nil`. Os demais campos permanecem como estavam.
- **Verificação de existência antes de modificar/excluir**: `Atualizar`, `Deletar` e `UploadFoto` chamam `BuscarPorId` antes de agir; se a equipe não existir (ou estiver deletada), retornam erro de "equipe não encontrada".
- **Soft delete (exclusão lógica)**: `Deletar` não remove a linha fisicamente. Ele delega ao repositório (`DeletarSoft`), que preenche `deletado_em` e `deletado_por` (o UUID de quem excluiu, vindo do JWT). Todas as consultas de leitura filtram por `deletado_em IS NULL`, então registros excluídos somem das listagens e buscas.
- **Herança de template**: o campo `TemplateID` da equipe, quando preenchido, sobrescreve o template definido na organização (conforme documentado na entidade). Quando nulo, a equipe segue o template da organização.
- **Upload de foto para o S3**:
  - A extensão do arquivo é derivada do `Content-Type` por `extensaoPorTipo` (`image/jpeg` → `.jpg`, `image/png` → `.png`, `image/webp` → `.webp`; qualquer outro cai no padrão `.jpg`).
  - A chave do objeto é montada como `equipes/{id}/foto{ext}` e prefixada via `armazenamento.ChaveCompleta(...)`.
  - O arquivo é enviado ao S3 (`Upload`) e a chave resultante é persistida na coluna `foto_key` (`AtualizarFoto`).
  - A validação do tipo de imagem (via `Content-Type`) é feita antecipadamente, na camada do `Controller`, antes de chamar o `UseCase`. Já o limite de 5 MB não é uma checagem prévia: o `Controller` envolve o corpo da requisição com `http.MaxBytesReader`, de modo que o erro de tamanho só é disparado quando o arquivo é efetivamente lido (durante o `Upload` ao S3, dentro do `UseCase`).
- **URL da foto é presignada e temporária**: o `UseCase` nunca devolve a chave bruta do S3. Em toda resposta que envolve uma equipe com foto, ele chama `gerarFotoURL`, que gera uma URL presignada com validade de `storage.ExpiracaoURLFoto` (**2 horas**). Se a chave for nula, ou o serviço de armazenamento não estiver configurado, ou ocorrer erro ao gerar a URL, o campo `foto_url` volta como `nil` (a operação não falha por causa disso).
- **Ordenação das listagens**: tanto `ListarPorUsuario` quanto `ListarPorOrganizacao` retornam as equipes ordenadas por `nome` em ordem crescente (`ORDER BY nome ASC`), no repositório.

> Observação: este módulo não trata hash de senha — isso pertence a outros módulos (ex.: usuário). A única operação "sensível" aqui é a geração de URLs presignadas de foto.

## Dependências

Cada construtor recebe sua dependência por **injeção via interface**, montadas em `cmd/api/rotas.go`:

- **`NovoRepositorio(db *sqlx.DB) Repositorio`**: recebe o pool de conexões MySQL (`sqlx.DB`). É o único componente que executa SQL na tabela `tb_equipes`.
- **`NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento) UseCase`**: recebe:
  - o `Repositorio` (acesso a dados), para persistir e ler equipes;
  - o serviço `storage.Armazenamento` (S3), usado para enviar a foto (`Upload`), montar a chave completa (`ChaveCompleta`) e gerar a URL presignada de leitura (`GerarURLPresignada`). Receber via interface permite trocar a implementação (ex.: por um mock em testes) sem mexer na regra de negócio.
- **`NovoController(uc UseCase) *Controller`**: recebe apenas o `UseCase`. O controller não conhece o banco nem o S3 diretamente — toda a lógica passa pelo `UseCase`.

A ligação real (instanciação) acontece em `cmd/api/rotas.go`:

```
equipeRepo := equipe.NovoRepositorio(db)
equipeUseCase := equipe.NovoUseCase(equipeRepo, s3Svc)
equipeController := equipe.NovoController(equipeUseCase)
equipeController.RegistrarRotas(api, authMiddleware)
```

Aqui `db` é a conexão MySQL, `s3Svc` é o serviço de armazenamento S3, `api` é o grupo de rotas `/api/v1` e `authMiddleware` é o middleware de validação do JWT aplicado a todas as rotas do módulo.
