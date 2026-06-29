# ═══════════════════════════════════════════════════════
# Dockerfile — OneByOne API
# Build multi-stage: compila em golang:alpine e roda em
# alpine mínimo para reduzir o tamanho da imagem final.
# ═══════════════════════════════════════════════════════

# ───────────────────────────────────────────────
# Estágio 1: build — compila o binário Go
# ───────────────────────────────────────────────
FROM golang:1.24-alpine AS build

# Instala git (necessário para alguns módulos Go)
RUN apk add --no-cache git

WORKDIR /app

# Copia os arquivos de dependência primeiro para aproveitar o cache de camadas:
# enquanto go.mod/go.sum não mudarem, o download de dependências fica em cache.
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código-fonte
COPY . .

# Compila o binário com CGO desativado para gerar um executável estático
# que funciona em qualquer imagem Linux sem dependências externas.
# -ldflags="-w -s" remove informações de debug e reduz o tamanho do binário.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o onebyone-api ./cmd/api

# ───────────────────────────────────────────────
# Estágio 2: produção — imagem mínima sem ferramentas de build
# ───────────────────────────────────────────────
FROM alpine:3.19

# Instala certificados TLS (necessários para chamadas HTTPS, ex.: S3)
# e tzdata para configuração de fuso horário
RUN apk --no-cache add ca-certificates tzdata

# Define o fuso horário padrão para o horário de Brasília
ENV TZ=America/Sao_Paulo

WORKDIR /app

# Cria um usuário não-root por segurança — evita que o processo rode como root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copia apenas o binário compilado do estágio de build
COPY --from=build /app/onebyone-api .

# Passa a propriedade do binário ao usuário não-root e ativa esse usuário
RUN chown appuser:appgroup /app/onebyone-api
USER appuser

# Expõe a porta HTTP da API (padrão definido por PORTA_API)
EXPOSE 8090

# Inicia o servidor HTTP.
# As variáveis de ambiente (banco, JWT, S3) vêm do docker-compose / ambiente,
# não de um arquivo .env embutido na imagem.
CMD ["./onebyone-api"]
