# Pacote `database`

> Responsável por abrir e validar a conexão com o banco de dados MySQL usada por toda a OneByOne API.

## O que faz

Este pacote concentra em um único lugar a criação da conexão com o MySQL. Ele monta a string de conexão (DSN) a partir das configurações da aplicação, abre o *pool* de conexões usando a biblioteca `sqlx` (uma extensão da `database/sql` padrão do Go) e confirma que o banco está realmente acessível antes de devolver a conexão pronta para uso. Pense nele como o equivalente, no .NET, a centralizar a montagem da `connectionString` e a abertura de um `SqlConnection` validado — mas aqui o objeto retornado é um *pool* reaproveitável durante toda a vida da aplicação.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `mysql.go` | Define a função `NovaConexao`, que monta o DSN, abre o *pool* de conexões MySQL e valida o acesso com um *ping*. |

## API pública

| Símbolo | Assinatura | O que faz |
| --- | --- | --- |
| `NovaConexao` | `NovaConexao(cfg *config.Config) (*sqlx.DB, error)` | Monta o DSN a partir das configurações, abre o *pool* de conexões MySQL, executa um `Ping` para confirmar que o banco responde e retorna o `*sqlx.DB` pronto. Em caso de falha, retorna um erro descritivo (com a causa original encapsulada via `%w`). |

Observação: o pacote não exporta tipos próprios. O retorno é o tipo `*sqlx.DB`, vindo da biblioteca externa `github.com/jmoiron/sqlx`.

## Como é usado

A conexão é criada uma única vez na inicialização da aplicação, no `main.go` (`cmd/api/main.go`), logo após o carregamento das configurações. O `*sqlx.DB` retornado é então repassado para a montagem das rotas e injetado nos repositórios/handlers que precisam acessar o banco.

```go
// cmd/api/main.go

// 1. Carrega as configurações (variáveis de ambiente / .env)
cfg, err := config.Carregar()
if err != nil {
    log.Fatalf("erro ao carregar configurações: %v", err)
}

// 2. Conecta ao banco de dados MySQL
db, err := database.NovaConexao(cfg)
if err != nil {
    log.Fatalf("erro ao conectar ao banco de dados: %v", err)
}
defer db.Close() // fecha o pool quando a aplicação encerra

// 3. O pool 'db' é passado adiante para montar as rotas e dependências
router := ConfigurarRotas(cfg, db, s3Svc)
```

O `defer db.Close()` garante que o *pool* seja encerrado quando a função `main` terminar — semelhante a um `using` no C#, porém com escopo de toda a aplicação.

## Detalhes importantes

- A conexão **não usa credenciais fixas no código**: todos os dados vêm do `*config.Config`, que por sua vez lê variáveis de ambiente. Os campos usados são:

  | Campo do `Config` | Variável de ambiente | Padrão |
  | --- | --- | --- |
  | `DBUser` | `DB_USER` | `root` |
  | `DBPassword` | `DB_PASSWORD` | (vazio) |
  | `DBHost` | `DB_HOST` | `localhost` |
  | `DBPort` | `DB_PORT` | `3306` |
  | `DBName` | `DB_NAME` | `onebyone` |

- O DSN gerado tem o formato:
  `usuario:senha@tcp(host:porta)/banco?charset=utf8mb4&parseTime=true&loc=Local`
- `charset=utf8mb4` garante suporte completo a UTF-8 (incluindo emojis e caracteres especiais).
- `parseTime=true` faz o driver converter automaticamente colunas `DATETIME`/`DATE` para o tipo `time.Time` do Go — sem isso, datas chegariam como bytes/`string`.
- `loc=Local` alinha o fuso horário ao do servidor onde a API está rodando.
- O driver MySQL (`github.com/go-sql-driver/mysql`) é importado apenas pelo seu efeito colateral (`_ "..."`): a importação registra o driver na `database/sql`, mas o pacote não chama nada dele diretamente. É por isso que `sqlx.Open("mysql", dsn)` consegue encontrar o driver pelo nome `"mysql"`.
- `sqlx.Open` **não abre uma conexão física imediatamente** — ele apenas prepara o *pool*. Por isso a função chama `db.Ping()` em seguida: é o `Ping` que efetivamente testa o acesso ao banco e detecta problemas de credenciais, rede ou banco indisponível antes de a aplicação começar a operar.
- Os erros são retornados com contexto (`fmt.Errorf("...: %w", err)`), preservando a causa original para diagnóstico — diferenciando claramente falha ao abrir o *pool* de falha no *ping*.
