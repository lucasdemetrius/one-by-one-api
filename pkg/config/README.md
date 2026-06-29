# Pacote `config`

> Centraliza a leitura das variáveis de ambiente da OneByOne API e as expõe em uma única struct tipada (`Config`).

## O que faz

Este pacote é o ponto único onde a aplicação lê suas configurações externas (banco de dados, JWT, porta HTTP e AWS S3). Na inicialização, ele tenta carregar um arquivo `.env` (útil em desenvolvimento, via biblioteca `godotenv`) e, em seguida, lê cada valor das variáveis de ambiente do sistema operacional. Toda variável ausente recebe um valor padrão razoável, de forma que a aplicação consiga subir mesmo sem um `.env` completo.

Pensando em quem vem do .NET Framework 4.8: ele cumpre o papel do `Web.config`/`<appSettings>` + `ConfigurationManager.AppSettings["..."]`, mas em vez de chaves de configuração espalhadas em XML, tudo vira campos fortemente tipados de uma struct preenchida uma única vez na partida do programa.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `config.go` | Define a struct `Config`, a função `Carregar()` (lê `.env` + variáveis de ambiente) e o auxiliar interno `getEnv` para aplicar valores padrão. |

## API pública

São exportados (começam com letra maiúscula) a struct `Config`, seus campos e a função `Carregar`. O auxiliar `getEnv` é minúsculo, portanto privado ao pacote.

| Símbolo | Assinatura | O que faz |
| --- | --- | --- |
| `Config` | `type Config struct { ... }` | Struct que guarda todas as configurações da aplicação já carregadas. É passada por ponteiro para os demais pacotes que precisam dela. |
| `Carregar` | `func Carregar() (*Config, error)` | Carrega o `.env` (se existir), lê as variáveis de ambiente e devolve um `*Config` preenchido. O `error` faz parte da assinatura para uso futuro, mas hoje sempre retorna `nil`. |

### Campos exportados de `Config`

| Campo | Tipo | Variável de ambiente | Padrão |
| --- | --- | --- | --- |
| `DBHost` | `string` | `DB_HOST` | `localhost` |
| `DBPort` | `string` | `DB_PORT` | `3306` |
| `DBUser` | `string` | `DB_USER` | `root` |
| `DBPassword` | `string` | `DB_PASSWORD` | `""` (vazio) |
| `DBName` | `string` | `DB_NAME` | `onebyone` |
| `JWTSecret` | `string` | `JWT_SECRET` | `""` (vazio) |
| `JWTExpiracaoHoras` | `int` | `JWT_EXPIRACAO_HORAS` | `24` |
| `PortaAPI` | `string` | `PORTA_API` | `8080` |
| `AWSAccessKeyID` | `string` | `AWS_ACCESS_KEY_ID` | `""` (vazio) |
| `AWSSecretAccessKey` | `string` | `AWS_SECRET_ACCESS_KEY` | `""` (vazio) |
| `AWSRegion` | `string` | `AWS_REGION` | `us-east-1` |
| `AWSBucket` | `string` | `AWS_BUCKET` | `controleazul` |
| `AWSPrefixo` | `string` | `AWS_PREFIXO` | `one-by-one` |

## Como é usado

O fluxo é sempre o mesmo: `Carregar()` é chamado uma vez no `main`, e o `*Config` resultante é injetado nos pacotes que dele dependem (injeção de dependência manual, sem container). Quem está acostumado com .NET pode pensar nesse `*Config` como uma instância única, parecida com um objeto registrado como singleton, repassada para quem precisa.

Locais reais que consomem este pacote:

| Local | Como usa |
| --- | --- |
| `cmd/api/main.go` | Ponto de entrada: chama `config.Carregar()` e, se der erro, encerra com `log.Fatalf`. Usa `cfg.PortaAPI` para montar o endereço de escuta. |
| `cmd/api/rotas.go` | `ConfigurarRotas(cfg *config.Config, ...)` recebe o `cfg` e o distribui para middlewares e casos de uso. |
| `pkg/database/mysql.go` | `NovaConexao(cfg *config.Config)` monta o DSN do MySQL a partir de `DBUser`, `DBPassword`, `DBHost`, `DBPort` e `DBName`. |
| `pkg/storage/s3.go` | `NovoArmazenamentoS3(cfg)` usa os campos `AWS*` para configurar o cliente do bucket S3. |
| `pkg/middleware/auth.go` | `AutenticarJWT(cfg *config.Config)` usa `cfg.JWTSecret` para validar a assinatura dos tokens. |
| `internal/usuario/usecase.go` | `NovoUseCase(repo, cfg, armazenamento)` guarda o `cfg` para gerar/validar tokens JWT. |

Exemplo curto (extraído de `cmd/api/main.go`):

```go
// 1. Carrega configurações (variáveis de ambiente / .env)
cfg, err := config.Carregar()
if err != nil {
    log.Fatalf("erro ao carregar configurações: %v", err)
}

// 2. Conecta ao banco usando o cfg
db, err := database.NovaConexao(cfg)

// 5. Sobe o servidor na porta configurada
endereco := fmt.Sprintf(":%s", cfg.PortaAPI)
router.Run(endereco)
```

## Detalhes importantes

- **`.env` é opcional.** `Carregar()` executa `_ = godotenv.Load()` e ignora o erro de propósito: em produção as variáveis já chegam pelo ambiente (ou pelo `docker-compose`), então a ausência do arquivo `.env` não é um problema.
- **Precedência.** Se uma variável já existir no ambiente do sistema, ela vale; senão, o `getEnv` aplica o valor padrão da tabela acima. Variáveis vazias (`""`) são tratadas como inexistentes e também caem no padrão.
- **`JWTExpiracaoHoras` é o único campo numérico.** A variável `JWT_EXPIRACAO_HORAS` é convertida com `strconv.Atoi`. Se a conversão falhar (valor não numérico), o pacote aplica o padrão seguro de `24` horas em vez de quebrar a inicialização.
- **Segredos vêm vazios por padrão.** `JWTSecret`, `DBPassword`, `AWSAccessKeyID` e `AWSSecretAccessKey` têm padrão `""`. Isso é intencional: nenhum segredo fica escrito no código-fonte; eles devem ser fornecidos pelo ambiente. Vale lembrar que um `JWTSecret` vazio deixa a validação de token insegura, então deve ser definido em qualquer ambiente real.
- **Sem rotas HTTP.** Este é um pacote de infraestrutura (`pkg/`); ele não registra endpoints nem expõe handlers — apenas fornece dados de configuração para os demais pacotes.
- **`Carregar()` nunca retorna erro hoje.** A assinatura inclui `error` por convenção e para permitir evolução futura (ex.: validar campos obrigatórios), mas a implementação atual sempre devolve `nil` no segundo retorno.
