// Pacote: internal/acompanhamento
// Arquivo: repository.go
// Descrição: Persistência dos registros de acompanhamento (tb_acompanhamentos).
//            Só I/O de banco. Soft delete (deletado_em) como o resto do projeto.
// Autor: OneByOne API
// Criado em: 2026

package acompanhamento

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o acesso ao banco dos acompanhamentos.
type Repositorio interface {
	// ListarPorColaborador lista os registros do liderado; tipo "" = todos.
	ListarPorColaborador(colaboradorID, tipo string) ([]Acompanhamento, error)
	Criar(a Acompanhamento) (Acompanhamento, error)
	BuscarPorId(id string) (Acompanhamento, error)
	Atualizar(a Acompanhamento) (Acompanhamento, error)
	DeletarSoft(id string) error
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de acompanhamento.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

const colunas = `id, colaborador_id, tipo, titulo, detalhe, valor, data_ref, criado_em, alterado_em, deletado_em`

func (r *repositorioMySQL) ListarPorColaborador(colaboradorID, tipo string) ([]Acompanhamento, error) {
	var itens []Acompanhamento
	query := `SELECT ` + colunas + ` FROM tb_acompanhamentos
	          WHERE colaborador_id = ? AND deletado_em IS NULL`
	args := []any{colaboradorID}
	if tipo != "" {
		query += ` AND tipo = ?`
		args = append(args, tipo)
	}
	query += ` ORDER BY data_ref DESC, criado_em DESC`
	if err := r.db.Select(&itens, query, args...); err != nil {
		return nil, fmt.Errorf("erro ao listar acompanhamentos: %w", err)
	}
	return itens, nil
}

func (r *repositorioMySQL) Criar(a Acompanhamento) (Acompanhamento, error) {
	query := `INSERT INTO tb_acompanhamentos
	            (id, colaborador_id, tipo, titulo, detalhe, valor, data_ref, criado_em)
	          VALUES
	            (:id, :colaborador_id, :tipo, :titulo, :detalhe, :valor, :data_ref, :criado_em)`
	if _, err := r.db.NamedExec(query, a); err != nil {
		return Acompanhamento{}, fmt.Errorf("erro ao criar acompanhamento: %w", err)
	}
	return r.BuscarPorId(a.ID)
}

func (r *repositorioMySQL) BuscarPorId(id string) (Acompanhamento, error) {
	var a Acompanhamento
	query := `SELECT ` + colunas + ` FROM tb_acompanhamentos WHERE id = ? AND deletado_em IS NULL`
	if err := r.db.Get(&a, query, id); err != nil {
		return Acompanhamento{}, fmt.Errorf("acompanhamento não encontrado: %w", err)
	}
	return a, nil
}

func (r *repositorioMySQL) Atualizar(a Acompanhamento) (Acompanhamento, error) {
	agora := time.Now()
	a.AlteradoEm = &agora
	query := `UPDATE tb_acompanhamentos
	          SET titulo = :titulo, detalhe = :detalhe, valor = :valor, data_ref = :data_ref, alterado_em = :alterado_em
	          WHERE id = :id AND deletado_em IS NULL`
	if _, err := r.db.NamedExec(query, a); err != nil {
		return Acompanhamento{}, fmt.Errorf("erro ao atualizar acompanhamento: %w", err)
	}
	return r.BuscarPorId(a.ID)
}

func (r *repositorioMySQL) DeletarSoft(id string) error {
	resultado, err := r.db.Exec(`UPDATE tb_acompanhamentos SET deletado_em = ? WHERE id = ? AND deletado_em IS NULL`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao remover acompanhamento: %w", err)
	}
	linhas, _ := resultado.RowsAffected()
	if linhas == 0 {
		return fmt.Errorf("acompanhamento não encontrado")
	}
	return nil
}
