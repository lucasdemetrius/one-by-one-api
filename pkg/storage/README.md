# Pacote `storage`

> Serviço de armazenamento de arquivos no AWS S3, com objetos privados acessados por URLs temporárias (presignadas).

## O que faz

Este pacote concentra toda a comunicação com o AWS S3 para guardar arquivos (na prática, fotos de usuários, colaboradores, equipes e organizações). Os arquivos são gravados como objetos **privados** — ninguém acessa pela URL pública do bucket. Quando o frontend precisa exibir uma foto, a API gera sob demanda uma **URL presignada**: um link temporário e assinado criptograficamente que expira depois de um tempo. Todos os arquivos deste projeto ficam isolados dentro de uma "pasta" (prefixo) do bucket, definida no `.env`, para conviver com outros projetos no mesmo bucket.

Para quem vem de C#/.NET: pense no `Armazenamento` como uma `interface` (igual em C#) que abstrai o S3. Em vez de injetar a dependência via construtor de uma classe, aqui usamos uma função `NovoArmazenamentoS3(...)` que devolve a implementação concreta. A URL presignada é equivalente a gerar um link temporário com token de acesso em vez de deixar o arquivo público.

## Arquivos

| Arquivo | Responsabilidade |
|---------|------------------|
| `s3.go` | Define a interface `Armazenamento`, a implementação concreta sobre o AWS S3 (`s3Armazenamento`), o construtor `NovoArmazenamentoS3` e a constante de expiração das URLs de foto. |

## API pública

Símbolos exportados (começam com letra maiúscula, ou seja, são visíveis fora do pacote):

| Símbolo | Assinatura | O que faz |
|---------|------------|-----------|
| `ExpiracaoURLFoto` | `const ExpiracaoURLFoto = 2 * time.Hour` | Tempo de validade padrão das URLs presignadas de foto. Após esse prazo, o frontend precisa pedir uma nova URL à API. |
| `Armazenamento` | `interface { ... }` | Contrato (interface) que descreve as operações de armazenamento. Quem consome o pacote depende dessa interface, não da implementação. |
| `NovoArmazenamentoS3` | `func NovoArmazenamentoS3(cfg *config.Config) (Armazenamento, error)` | Cria e devolve uma instância pronta do serviço S3 a partir das configurações da aplicação. Carrega as credenciais AWS e monta os clientes S3 e de presign. |

Métodos da interface `Armazenamento`:

| Método | Assinatura | O que faz |
|--------|------------|-----------|
| `Upload` | `Upload(chave string, arquivo io.Reader, tamanho int64, tipoConteudo string) error` | Envia um arquivo para o S3 na `chave` informada. Não aplica ACL pública — o objeto fica privado. |
| `GerarURLPresignada` | `GerarURLPresignada(chave string, expiracao time.Duration) (string, error)` | Gera uma URL temporária e assinada para acessar (GET) um objeto privado. A assinatura é local: não faz chamada de rede ao S3. |
| `Deletar` | `Deletar(chave string) error` | Remove permanentemente o objeto do S3 pela `chave`. |
| `ChaveCompleta` | `ChaveCompleta(caminho string) string` | Prefixa o caminho com o prefixo do projeto. Ex.: `usuarios/abc-123/foto.jpg` → `one-by-one/usuarios/abc-123/foto.jpg`. Se o prefixo estiver vazio, devolve o caminho sem alteração. |

> O tipo `s3Armazenamento` (minúsculo) é a implementação concreta e **não** é exportado — só é alcançável pela interface `Armazenamento`.

## Como é usado

A instância é criada uma única vez na inicialização da aplicação, em `cmd/api/main.go`, e depois injetada nas rotas e nos casos de uso (use cases) que precisam mexer com fotos.

Criação no `main.go`:

```go
// cmd/api/main.go
s3Svc, err := storage.NovoArmazenamentoS3(cfg)
if err != nil {
    log.Fatalf("erro ao inicializar serviço S3: %v", err)
}
router := ConfigurarRotas(cfg, db, s3Svc)
```

Injeção nas rotas (`cmd/api/rotas.go`):

```go
func ConfigurarRotas(cfg *config.Config, db *sqlx.DB, s3Svc storage.Armazenamento) *gin.Engine { ... }
```

Uso típico dentro de um use case — fluxo completo de upload de foto (`internal/usuario/usecase.go`, e de forma análoga em `colaborador`, `equipe` e `organizacao`):

```go
// monta o caminho e aplica o prefixo do projeto
caminho := fmt.Sprintf("usuarios/%s/foto%s", id, ext)
chave := uc.armazenamento.ChaveCompleta(caminho)

// envia o arquivo (privado)
if err := uc.armazenamento.Upload(chave, arquivo, tamanho, tipoConteudo); err != nil { ... }

// persiste a chave no banco e gera a URL temporária para o frontend
url, err := uc.armazenamento.GerarURLPresignada(chave, storage.ExpiracaoURLFoto)
```

Os use cases que dependem deste pacote recebem `storage.Armazenamento` no construtor `NovoUseCase(...)` e chamam `GerarURLPresignada` sempre que montam o DTO de resposta com a foto.

## Detalhes importantes

- **Objetos sempre privados**: o `Upload` (via `PutObject`) não define ACL pública. O acesso só acontece por URL presignada.
- **URLs temporárias**: `GerarURLPresignada` usa `PresignGetObject` com `s3.WithPresignExpires`. A assinatura é feita localmente com as credenciais IAM — não há chamada de rede ao S3 ao gerar a URL. A validade padrão para fotos é `ExpiracaoURLFoto` (2 horas); passado esse prazo, o frontend precisa pedir uma URL nova à API.
- **Prefixo do projeto**: `ChaveCompleta` isola os arquivos deste projeto dentro do bucket compartilhado, prefixando com `AWSPrefixo` (padrão `one-by-one`). Se o prefixo for vazio, o caminho fica inalterado.
- **Credenciais estáticas**: `NovoArmazenamentoS3` usa `credentials.NewStaticCredentialsProvider` com `AWSAccessKeyID` e `AWSSecretAccessKey`. O *session token* é vazio, pois são credenciais de usuário IAM permanente.
- **Variáveis de ambiente lidas** (via `pkg/config`):

  | Variável de ambiente | Campo em `config.Config` | Padrão |
  |----------------------|--------------------------|--------|
  | `AWS_ACCESS_KEY_ID` | `AWSAccessKeyID` | (vazio) |
  | `AWS_SECRET_ACCESS_KEY` | `AWSSecretAccessKey` | (vazio) |
  | `AWS_REGION` | `AWSRegion` | `us-east-1` |
  | `AWS_BUCKET` | `AWSBucket` | `controleazul` |
  | `AWS_PREFIXO` | `AWSPrefixo` | `one-by-one` |

- **Tratamento de erros**: todas as operações embrulham o erro original com `fmt.Errorf("...: %w", err)` incluindo a chave envolvida, preservando a cadeia de erros para diagnóstico (equivalente a um `InnerException` no .NET).
- **Contexto**: as chamadas ao SDK usam `context.Background()` (sem timeout/cancelamento configurado neste pacote).
