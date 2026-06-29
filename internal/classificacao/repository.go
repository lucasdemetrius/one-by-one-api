// Pacote: internal/classificacao
// Arquivo: repository.go
// Descrição: Interface Repositorio e implementação MySQL (sqlx) para a tabela
//            tb_classificacoes. Apenas I/O de banco.
// Autor: OneByOne API
// Criado em: 2025

package classificacao

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repositorio define as operações de persistência das classificações.
type Repositorio interface {
	// Definir insere ou atualiza a classificação de um colaborador (upsert)
	Definir(c Classificacao) error
	// ListarPorOrganizacao retorna as classificações dos liderados de uma organização
	ListarPorOrganizacao(organizacaoID string) ([]Classificacao, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de classificações.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Definir(c Classificacao) error {
	query := `INSERT INTO tb_classificacoes (colaborador_id, desempenho, potencial, atualizado_em)
	          VALUES (?, ?, ?, ?)
	          ON DUPLICATE KEY UPDATE
	            desempenho = VALUES(desempenho),
	            potencial = VALUES(potencial),
	            atualizado_em = VALUES(atualizado_em)`
	if _, err := r.db.Exec(query, c.ColaboradorID, c.Desempenho, c.Potencial, c.AtualizadoEm); err != nil {
		return fmt.Errorf("erro ao salvar classificação: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ListarPorOrganizacao(organizacaoID string) ([]Classificacao, error) {
	var lista []Classificacao
	// Junta com colaboradores para trazer só os liderados ativos da organização.
	query := `SELECT cl.colaborador_id, cl.desempenho, cl.potencial, cl.atualizado_em
	          FROM tb_classificacoes cl
	          JOIN tb_colaboradores co ON co.id = cl.colaborador_id
	          WHERE co.organizacao_id = ? AND co.deletado_em IS NULL`
	if err := r.db.Select(&lista, query, organizacaoID); err != nil {
		return nil, fmt.Errorf("erro ao listar classificações: %w", err)
	}
	return lista, nil
}
