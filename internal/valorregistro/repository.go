// Pacote: internal/valorregistro
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de valores
//            de registro. Toda interação com tb_valores_registro passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package valorregistro

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o contrato de acesso ao banco para a entidade ValorRegistro
type Repositorio interface {
	// Criar insere um novo valor e retorna o registro persistido
	Criar(valor ValorRegistro) (ValorRegistro, error)
	// BuscarPorId retorna um valor ativo pelo UUID
	BuscarPorId(id string) (ValorRegistro, error)
	// ListarPorRegistro retorna todos os valores ativos de um registro de one-on-one
	ListarPorRegistro(registroID string) ([]ValorRegistro, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(valor ValorRegistro) (ValorRegistro, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de valores de registro
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// Criar insere um novo valor na tabela tb_valores_registro e retorna o dado completo
func (r *repositorioMySQL) Criar(valor ValorRegistro) (ValorRegistro, error) {
	query := `
		INSERT INTO tb_valores_registro (id, registro_id, bloco_id, valor_texto, valor_json, criado_em)
		VALUES (:id, :registro_id, :bloco_id, :valor_texto, :valor_json, :criado_em)
	`
	if _, err := r.db.NamedExec(query, valor); err != nil {
		return ValorRegistro{}, fmt.Errorf("erro ao inserir valor de registro: %w", err)
	}
	return r.BuscarPorId(valor.ID)
}

// BuscarPorId retorna um valor de registro ativo pelo UUID
func (r *repositorioMySQL) BuscarPorId(id string) (ValorRegistro, error) {
	var v ValorRegistro
	query := `
		SELECT id, registro_id, bloco_id, valor_texto, valor_json, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_valores_registro
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&v, query, id); err != nil {
		return ValorRegistro{}, fmt.Errorf("valor de registro não encontrado: %w", err)
	}
	return v, nil
}

// ListarPorRegistro retorna todos os valores ativos de um registro, ordenados pelo bloco
func (r *repositorioMySQL) ListarPorRegistro(registroID string) ([]ValorRegistro, error) {
	var valores []ValorRegistro
	query := `
		SELECT id, registro_id, bloco_id, valor_texto, valor_json, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_valores_registro
		WHERE registro_id = ? AND deletado_em IS NULL
		ORDER BY criado_em ASC
	`
	if err := r.db.Select(&valores, query, registroID); err != nil {
		return nil, fmt.Errorf("erro ao listar valores do registro: %w", err)
	}
	return valores, nil
}

// Atualizar aplica as modificações em um valor existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(valor ValorRegistro) (ValorRegistro, error) {
	agora := time.Now()
	valor.AlteradoEm = &agora

	query := `
		UPDATE tb_valores_registro
		SET valor_texto = :valor_texto, valor_json = :valor_json, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, valor); err != nil {
		return ValorRegistro{}, fmt.Errorf("erro ao atualizar valor de registro: %w", err)
	}
	return r.BuscarPorId(valor.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_valores_registro
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar valor de registro: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("valor de registro não encontrado ou já deletado")
	}
	return nil
}
