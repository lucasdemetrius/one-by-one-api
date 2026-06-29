// Pacote: internal/tabuleiro
// Arquivo: repository.go
// Descrição: Persistência do tabuleiro do 1:1 (tb_tabuleiros). Upsert por liderado
//            (INSERT ... ON DUPLICATE KEY UPDATE). Só I/O de banco.
// Autor: OneByOne API
// Criado em: 2026

package tabuleiro

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ErrSemTabuleiro indica que o liderado ainda não tem um tabuleiro salvo.
var ErrSemTabuleiro = errors.New("tabuleiro não encontrado")

// Repositorio define o acesso ao banco do tabuleiro.
type Repositorio interface {
	// Buscar devolve o JSON do tabuleiro do liderado; ErrSemTabuleiro se não houver.
	Buscar(colaboradorID string) (string, error)
	// Salvar grava (cria ou atualiza) o estado do tabuleiro do liderado.
	Salvar(colaboradorID, estado string) error
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de tabuleiro.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Buscar(colaboradorID string) (string, error) {
	var estado string
	query := `SELECT estado FROM tb_tabuleiros WHERE colaborador_id = ?`
	if err := r.db.Get(&estado, query, colaboradorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSemTabuleiro
		}
		return "", fmt.Errorf("erro ao buscar tabuleiro: %w", err)
	}
	return estado, nil
}

func (r *repositorioMySQL) Salvar(colaboradorID, estado string) error {
	query := `
		INSERT INTO tb_tabuleiros (colaborador_id, estado, criado_em)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE estado = VALUES(estado), alterado_em = ?
	`
	agora := time.Now()
	if _, err := r.db.Exec(query, colaboradorID, estado, agora, agora); err != nil {
		return fmt.Errorf("erro ao salvar tabuleiro: %w", err)
	}
	return nil
}
