# OneByOne API

API REST em Go para gerenciamento de reuniões **one-on-one** entre líderes e colaboradores.

## Tecnologias

- **Go 1.24** — linguagem principal
- **Gin** — framework HTTP
- **MySQL 8.0** — banco de dados (via Docker)
- **sqlx** — acesso ao banco com mapeamento de structs
- **JWT (golang-jwt/v5)** — autenticação Bearer Token
- **bcrypt** — hash de senhas (custo 12)
- **AWS S3** — armazenamento privado de fotos (URLs presignadas com validade de 2h)
- **Docker Compose** — banco de dados local para desenvolvimento

## Arquitetura

Clean Architecture em camadas:

```
Controller → UseCase → Repository → Entity
                ↓
            Mapper → DTO
```

Cada módulo é independente e toda dependência é injetada via interface.

## Módulos

| Módulo | Descrição |
|---|---|
| `usuario` | Cadastro, autenticação JWT e perfil |
| `organizacao` | Organização gerenciada pelo líder |
| `equipe` | Times dentro de uma organização |
| `colaborador` | Membros de uma equipe |
| `template` | Templates de agenda para reuniões |
| `templatebloco` | Blocos de perguntas dentro de um template |
| `oneaone` | Reunião one-on-one entre líder e colaborador |
| `registrooneaone` | Registros de uma reunião realizada |
| `valorregistro` | Respostas preenchidas em cada registro |

## Configuração

1. Copie `.env.example` para `.env` e preencha os valores:

```bash
cp .env.example .env
```

Variáveis obrigatórias:

```env
# Banco de dados
DB_HOST=localhost
DB_PORT=3306
DB_USER=oneaone_user
DB_PASSWORD=sua_senha
DB_NAME=oneaone

# JWT
JWT_SECRET=sua_chave_secreta_longa
JWT_EXPIRACAO_HORAS=24

# Servidor
PORTA_API=8080

# AWS S3 (fotos privadas via URL presignada)
AWS_ACCESS_KEY_ID=sua_access_key
AWS_SECRET_ACCESS_KEY=sua_secret_key
AWS_REGION=us-east-1
AWS_BUCKET=controleazul
AWS_PREFIXO=one-by-one
```

2. Suba o banco de dados com Docker:

```bash
docker compose up -d
```

As migrations são aplicadas automaticamente na primeira inicialização.

3. Execute a API:

```bash
go run ./cmd/api
```

A API estará disponível em `http://localhost:8080/api/v1`.

## Endpoints principais

### Autenticação
```
POST /api/v1/auth/login
```

### Recursos (todos requerem Bearer Token)
```
POST/GET/PUT/DELETE /api/v1/usuarios
POST/GET/PUT/DELETE /api/v1/organizacoes
POST/GET/PUT/DELETE /api/v1/equipes
POST/GET/PUT/DELETE /api/v1/colaboradores
POST/GET/PUT/DELETE /api/v1/templates
POST/GET/PUT/DELETE /api/v1/oneaones
POST/GET/PUT/DELETE /api/v1/registros
```

### Upload de foto (multipart/form-data, campo `foto`, máx 5MB)
```
POST /api/v1/usuarios/:id/foto
POST /api/v1/organizacoes/:id/foto
POST /api/v1/equipes/:id/foto
POST /api/v1/colaboradores/:id/foto
```

### Healthcheck
```
GET /api/v1/health
```

## Política IAM (AWS)

Crie um usuário IAM com a seguinte política para restringir acesso apenas à pasta do projeto dentro do bucket:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:PutObject",
      "s3:GetObject",
      "s3:DeleteObject"
    ],
    "Resource": "arn:aws:s3:::controleazul/one-by-one/*"
  }]
}
```

## Regras de negócio

### Herança de template
O template de uma reunião é resolvido por prioridade (COALESCE):

1. Template exclusivo do **colaborador**
2. Template da **equipe**
3. Template da **organização**
4. Primeiro template criado pelo **líder**

### Soft Delete
Todos os registros usam exclusão lógica (`deletado_em` + `deletado_por`). Nenhum dado é removido fisicamente do banco.

### Fotos privadas
Objetos no S3 nunca recebem ACL pública. O acesso é feito exclusivamente via **URL presignada** gerada sob demanda (validade de 2 horas). A chave S3 é armazenada no banco; a URL é gerada dinamicamente pela API.
