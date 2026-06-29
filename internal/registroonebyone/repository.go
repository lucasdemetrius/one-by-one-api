// Pacote: internal/registroonebyone
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de registros
//            de one-on-one. Toda interação com tb_registros_onebyone passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package registroonebyone

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o contrato de acesso ao banco para a entidade RegistroOneByOne
type Repositorio interface {
	// Criar insere um novo registro e retorna o dado persistido
	Criar(registro RegistroOneByOne) (RegistroOneByOne, error)
	// BuscarPorId retorna um registro ativo pelo UUID
	BuscarPorId(id string) (RegistroOneByOne, error)
	// ListarPorOneByOne retorna todos os registros ativos de uma reunião
	ListarPorOneByOne(oneaoneID string) ([]RegistroOneByOne, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de registros de one-on-one
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// Criar insere um novo registro na tabela tb_registros_onebyone e retorna o dado completo
func (r *repositorioMySQL) Criar(registro RegistroOneByOne) (RegistroOneByOne, error) {
	query := `
		INSERT INTO tb_registros_onebyone (id, oneaone_id, template_id, criado_em)
		VALUES (:id, :oneaone_id, :template_id, :criado_em)
	`
	if _, err := r.db.NamedExec(query, registro); err != nil {
		return RegistroOneByOne{}, fmt.Errorf("erro ao inserir registro de one-on-one: %w", err)
	}
	return r.BuscarPorId(registro.ID)
}

// BuscarPorId retorna um registro ativo pelo UUID
func (r *repositorioMySQL) BuscarPorId(id string) (RegistroOneByOne, error) {
	var reg RegistroOneByOne
	query := `
		SELECT id, oneaone_id, template_id, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_registros_onebyone
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&reg, query, id); err != nil {
		return RegistroOneByOne{}, fmt.Errorf("registro de one-on-one não encontrado: %w", err)
	}
	return reg, nil
}

// ListarPorOneByOne retorna todos os registros ativos de uma reunião, ordenados do mais recente
func (r *repositorioMySQL) ListarPorOneByOne(oneaoneID string) ([]RegistroOneByOne, error) {
	var registros []RegistroOneByOne
	query := `
		SELECT id, oneaone_id, template_id, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_registros_onebyone
		WHERE oneaone_id = ? AND deletado_em IS NULL
		ORDER BY criado_em DESC
	`
	if err := r.db.Select(&registros, query, oneaoneID); err != nil {
		return nil, fmt.Errorf("erro ao listar registros do one-on-one: %w", err)
	}
	return registros, nil
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_registros_onebyone
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar registro de one-on-one: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("registro não encontrado ou já deletado")
	}
	return nil
}
