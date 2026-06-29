// Pacote: internal/convite
// Arquivo: repository.go
// Descrição: Interface Repositorio e implementação MySQL (sqlx) para a tabela
//            tb_convites. Apenas I/O de banco, sem regra de negócio.
// Autor: OneByOne API
// Criado em: 2025

package convite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define as operações de persistência de convites.
type Repositorio interface {
	// Criar insere um novo convite e retorna o registro persistido
	Criar(c Convite) (Convite, error)
	// BuscarPorToken localiza um convite pelo seu UUID (token do link)
	BuscarPorToken(token string) (Convite, error)
	// MarcarAceito muda o status para ACEITO e grava a data de aceite
	MarcarAceito(id string, quando time.Time) error
	// CancelarPendentesDoColaborador invalida convites pendentes anteriores do
	// mesmo colaborador (mantém apenas o convite mais recente válido)
	CancelarPendentesDoColaborador(colaboradorID string) error
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de convites com a conexão injetada.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Criar(c Convite) (Convite, error) {
	query := `INSERT INTO tb_convites (id, colaborador_id, codigo_hash, status, expira_em, criado_em)
	          VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := r.db.Exec(query, c.ID, c.ColaboradorID, c.CodigoHash, c.Status, c.ExpiraEm, c.CriadoEm); err != nil {
		return Convite{}, fmt.Errorf("erro ao inserir convite: %w", err)
	}
	return c, nil
}

func (r *repositorioMySQL) BuscarPorToken(token string) (Convite, error) {
	var c Convite
	query := `SELECT id, colaborador_id, codigo_hash, status, expira_em, criado_em, aceito_em
	          FROM tb_convites WHERE id = ?`
	if err := r.db.Get(&c, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Convite{}, fmt.Errorf("convite não encontrado")
		}
		return Convite{}, fmt.Errorf("erro ao buscar convite: %w", err)
	}
	return c, nil
}

func (r *repositorioMySQL) MarcarAceito(id string, quando time.Time) error {
	if _, err := r.db.Exec(`UPDATE tb_convites SET status = ?, aceito_em = ? WHERE id = ?`,
		StatusAceito, quando, id); err != nil {
		return fmt.Errorf("erro ao marcar convite como aceito: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) CancelarPendentesDoColaborador(colaboradorID string) error {
	if _, err := r.db.Exec(`UPDATE tb_convites SET status = ? WHERE colaborador_id = ? AND status = ?`,
		StatusCancelado, colaboradorID, StatusPendente); err != nil {
		return fmt.Errorf("erro ao cancelar convites anteriores: %w", err)
	}
	return nil
}
