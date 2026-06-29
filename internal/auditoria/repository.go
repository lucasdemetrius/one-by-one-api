package auditoria

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o contrato de acesso ao banco para auditoria
type Repositorio interface {
	Gravar(a Auditoria) error
	ListarPorUsuario(usuarioID string, limite int) ([]Auditoria, error)
	// ListarPorEntidade traz os últimos eventos cujo entidade_id bate (ex.: a
	// linha do tempo de um colaborador — blocos, classificação, edições etc.).
	ListarPorEntidade(entidadeID string, limite int) ([]Auditoria, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Gravar(a Auditoria) error {
	query := `
		INSERT INTO tb_auditoria (id, usuario_id, acao, entidade, entidade_id, ip, user_agent, criado_em)
		VALUES (:id, :usuario_id, :acao, :entidade, :entidade_id, :ip, :user_agent, :criado_em)
	`
	if _, err := r.db.NamedExec(query, a); err != nil {
		return fmt.Errorf("erro ao gravar auditoria: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ListarPorUsuario(usuarioID string, limite int) ([]Auditoria, error) {
	var registros []Auditoria
	query := `
		SELECT id, usuario_id, acao, entidade, entidade_id, ip, user_agent, criado_em
		FROM tb_auditoria
		WHERE usuario_id = ?
		ORDER BY criado_em DESC
		LIMIT ?
	`
	if err := r.db.Select(&registros, query, usuarioID, limite); err != nil {
		return nil, fmt.Errorf("erro ao listar auditoria: %w", err)
	}
	return registros, nil
}

func (r *repositorioMySQL) ListarPorEntidade(entidadeID string, limite int) ([]Auditoria, error) {
	var registros []Auditoria
	query := `
		SELECT id, usuario_id, acao, entidade, entidade_id, ip, user_agent, criado_em
		FROM tb_auditoria
		WHERE entidade_id = ?
		ORDER BY criado_em DESC
		LIMIT ?
	`
	if err := r.db.Select(&registros, query, entidadeID, limite); err != nil {
		return nil, fmt.Errorf("erro ao listar linha do tempo: %w", err)
	}
	return registros, nil
}
