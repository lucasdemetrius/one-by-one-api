// Pacote: internal/template
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de templates.
//            Toda interação com a tabela tb_template passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package template

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/autorizacao"
)

// Repositorio define o contrato de acesso ao banco para a entidade Template
type Repositorio interface {
	// Criar insere um novo template e retorna o registro persistido
	Criar(t Template) (Template, error)
	// BuscarPorId retorna um template ativo pelo UUID
	BuscarPorId(id string) (Template, error)
	// ListarPorUsuario retorna todos os templates ativos de um líder, ordenados por criado_em
	ListarPorUsuario(usuarioID string) ([]Template, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(t Template) (Template, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
	// GestorPertenceAoRH diz se o gestor dono (usuario_id) pertence ao tenant do RH
	// informado — fallback de posse para o papel RH (Cadeia A).
	GestorPertenceAoRH(gestorID, rhID string) (bool, error)
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de templates
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// GestorPertenceAoRH delega à primitiva compartilhada: o gestor dono (usuario_id) pertence
// ao tenant do RH informado? Fallback de posse para o papel RH (Cadeia A).
func (r *repositorioMySQL) GestorPertenceAoRH(gestorID, rhID string) (bool, error) {
	return autorizacao.GestorPertenceAoRH(r.db, gestorID, rhID)
}

// Criar insere um novo template na tabela tb_template e retorna o registro completo
func (r *repositorioMySQL) Criar(t Template) (Template, error) {
	query := `
		INSERT INTO tb_template (id, usuario_id, nome, criado_em)
		VALUES (:id, :usuario_id, :nome, :criado_em)
	`
	if _, err := r.db.NamedExec(query, t); err != nil {
		return Template{}, fmt.Errorf("erro ao inserir template: %w", err)
	}
	return r.BuscarPorId(t.ID)
}

// BuscarPorId retorna um template ativo pelo UUID; erro se não existir ou estiver deletado
func (r *repositorioMySQL) BuscarPorId(id string) (Template, error) {
	var t Template
	query := `
		SELECT id, usuario_id, nome, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_template
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&t, query, id); err != nil {
		return Template{}, fmt.Errorf("template não encontrado: %w", err)
	}
	return t, nil
}

// ListarPorUsuario retorna todos os templates ativos de um líder, ordenados do mais antigo ao mais novo.
// A ordenação por criado_em ASC é intencional: o primeiro template criado é considerado o padrão
// do líder pela regra de herança de template do módulo oneaone.
func (r *repositorioMySQL) ListarPorUsuario(usuarioID string) ([]Template, error) {
	var templates []Template
	query := `
		SELECT id, usuario_id, nome, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_template
		WHERE usuario_id = ? AND deletado_em IS NULL
		ORDER BY criado_em ASC
	`
	if err := r.db.Select(&templates, query, usuarioID); err != nil {
		return nil, fmt.Errorf("erro ao listar templates: %w", err)
	}
	return templates, nil
}

// Atualizar aplica as modificações em um template existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(t Template) (Template, error) {
	agora := time.Now()
	t.AlteradoEm = &agora

	query := `
		UPDATE tb_template
		SET nome = :nome, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, t); err != nil {
		return Template{}, fmt.Errorf("erro ao atualizar template: %w", err)
	}
	return r.BuscarPorId(t.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_template
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar template: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("template não encontrado ou já deletado")
	}
	return nil
}
