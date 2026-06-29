// Pacote: internal/pdi
// Arquivo: repository.go
// Descrição: Persistência dos itens de PDI (tb_pdi_itens). Só I/O de banco.
// Autor: OneByOne API
// Criado em: 2026

package pdi

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o acesso ao banco dos itens de PDI.
type Repositorio interface {
	ListarPorColaborador(colaboradorID string) ([]ItemPDI, error)
	Criar(item ItemPDI) (ItemPDI, error)
	BuscarPorId(id string) (ItemPDI, error)
	Atualizar(item ItemPDI) (ItemPDI, error)
	DeletarSoft(id string) error
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de PDI.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

const colunas = `id, colaborador_id, titulo, descricao, prazo, concluido, concluido_em, criado_em, alterado_em, deletado_em`

func (r *repositorioMySQL) ListarPorColaborador(colaboradorID string) ([]ItemPDI, error) {
	var itens []ItemPDI
	query := `SELECT ` + colunas + ` FROM tb_pdi_itens
	          WHERE colaborador_id = ? AND deletado_em IS NULL
	          ORDER BY concluido ASC, prazo IS NULL, prazo ASC, criado_em DESC`
	if err := r.db.Select(&itens, query, colaboradorID); err != nil {
		return nil, fmt.Errorf("erro ao listar PDI: %w", err)
	}
	return itens, nil
}

func (r *repositorioMySQL) Criar(item ItemPDI) (ItemPDI, error) {
	query := `INSERT INTO tb_pdi_itens (id, colaborador_id, titulo, descricao, prazo, concluido, criado_em)
	          VALUES (:id, :colaborador_id, :titulo, :descricao, :prazo, :concluido, :criado_em)`
	if _, err := r.db.NamedExec(query, item); err != nil {
		return ItemPDI{}, fmt.Errorf("erro ao criar item de PDI: %w", err)
	}
	return r.BuscarPorId(item.ID)
}

func (r *repositorioMySQL) BuscarPorId(id string) (ItemPDI, error) {
	var item ItemPDI
	query := `SELECT ` + colunas + ` FROM tb_pdi_itens WHERE id = ? AND deletado_em IS NULL`
	if err := r.db.Get(&item, query, id); err != nil {
		return ItemPDI{}, fmt.Errorf("item de PDI não encontrado: %w", err)
	}
	return item, nil
}

func (r *repositorioMySQL) Atualizar(item ItemPDI) (ItemPDI, error) {
	agora := time.Now()
	item.AlteradoEm = &agora
	query := `UPDATE tb_pdi_itens
	          SET titulo = :titulo, prazo = :prazo, concluido = :concluido, concluido_em = :concluido_em, alterado_em = :alterado_em
	          WHERE id = :id AND deletado_em IS NULL`
	if _, err := r.db.NamedExec(query, item); err != nil {
		return ItemPDI{}, fmt.Errorf("erro ao atualizar item de PDI: %w", err)
	}
	return r.BuscarPorId(item.ID)
}

func (r *repositorioMySQL) DeletarSoft(id string) error {
	resultado, err := r.db.Exec(`UPDATE tb_pdi_itens SET deletado_em = ? WHERE id = ? AND deletado_em IS NULL`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao remover item de PDI: %w", err)
	}
	linhas, _ := resultado.RowsAffected()
	if linhas == 0 {
		return fmt.Errorf("item de PDI não encontrado")
	}
	return nil
}
