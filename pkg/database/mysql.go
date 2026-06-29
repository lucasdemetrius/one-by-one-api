// Pacote: pkg/database
// Arquivo: mysql.go
// Descrição: Responsável por estabelecer e retornar a conexão com o banco de
//            dados MySQL utilizando sqlx como extensão do database/sql padrão.
//            A conexão é configurada com charset UTF-8 e parse automático de datas.
// Autor: OneByOne API
// Criado em: 2025

package database

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql" // driver MySQL registrado via side-effect
	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/config"
)

// NovaConexao cria e valida uma nova conexão com o banco de dados MySQL.
// Retorna um ponteiro para *sqlx.DB pronto para uso ou um erro descritivo.
func NovaConexao(cfg *config.Config) (*sqlx.DB, error) {
	// Monta o DSN usando as variáveis de ambiente para evitar credenciais fixas no código
	// parseTime=true converte automaticamente colunas DATETIME para time.Time
	// loc=Local ajusta o fuso horário para o do servidor onde a API está rodando
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	// Abre o pool de conexões — ainda não estabelece a conexão física
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir pool de conexões MySQL: %w", err)
	}

	// Verifica se o banco está acessível realizando um ping
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao verificar conexão com MySQL (ping falhou): %w", err)
	}

	return db, nil
}
