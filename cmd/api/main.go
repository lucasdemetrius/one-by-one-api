// Pacote: cmd/api
// Arquivo: main.go
// Descrição: Ponto de entrada da aplicação OneByOne API. Cuida apenas do ciclo
//            de vida: carrega configurações, conecta ao banco MySQL, inicializa
//            o S3, monta as rotas (em rotas.go) e sobe o servidor HTTP.
//            Toda a fiação de módulos e rotas fica em rotas.go, mantendo este
//            arquivo enxuto.
// Autor: OneByOne API
// Criado em: 2025

// @title           OneByOne API
// @version         1.0
// @description     API para gerenciamento de reuniões one-on-one entre líderes e colaboradores.
// @termsOfService  http://swagger.io/terms/

// @contact.name   OneByOne Suporte
// @contact.email  suporte@oneaone.com.br

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8090
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Informe o token JWT no formato: Bearer {seu_token}

package main

import (
	"fmt"
	"log"

	migracoes "onebyone-api/migrations"
	"onebyone-api/pkg/config"
	"onebyone-api/pkg/database"
	"onebyone-api/pkg/storage"
)

func main() {
	// ─── 1. Carrega configurações (variáveis de ambiente / .env) ─────────────────
	cfg, err := config.Carregar()
	if err != nil {
		log.Fatalf("erro ao carregar configurações: %v", err)
	}

	// ─── 2. Conecta ao banco de dados MySQL ──────────────────────────────────────
	db, err := database.NovaConexao(cfg)
	if err != nil {
		log.Fatalf("erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()
	log.Println("conexão com o banco de dados estabelecida com sucesso")

	// ─── 2.1 Aplica as migrations pendentes no boot (estilo Flyway) ──────────────
	// Cria/atualiza as tabelas sozinho, em banco novo ou já existente. Não sobe a API
	// com o banco fora do esquema esperado.
	if err := database.AplicarMigracoes(cfg, migracoes.Arquivos); err != nil {
		log.Fatalf("erro ao aplicar migrations: %v", err)
	}

	// ─── 3. Inicializa o serviço de armazenamento S3 ─────────────────────────────
	s3Svc, err := storage.NovoArmazenamentoS3(cfg)
	if err != nil {
		log.Fatalf("erro ao inicializar serviço S3: %v", err)
	}

	// ─── 4. Monta as rotas e o grafo de dependências (ver rotas.go) ──────────────
	router := ConfigurarRotas(cfg, db, s3Svc)

	// ─── 5. Inicia o servidor HTTP ───────────────────────────────────────────────
	endereco := fmt.Sprintf(":%s", cfg.PortaAPI)
	log.Printf("servidor OneByOne API iniciado em http://localhost%s/api/v1", endereco)
	log.Printf("healthcheck: http://localhost%s/api/v1/health", endereco)

	if err := router.Run(endereco); err != nil {
		log.Fatalf("erro ao iniciar o servidor HTTP: %v", err)
	}
}
