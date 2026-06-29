// Pacote: internal/recuperacao
// Arquivo: repository.go
// Descrição: Persistência da recuperação de senha (tb_recuperacoes_senha). Só I/O.
// Autor: OneByOne API
// Criado em: 2026

package recuperacao

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o acesso a dados das recuperações de senha.
type Repositorio interface {
	// Criar insere um novo pedido de recuperação
	Criar(r Recuperacao) error
	// BuscarPorToken localiza pelo UUID (token do link)
	BuscarPorToken(token string) (Recuperacao, error)
	// MarcarUsado encerra o pedido (uso único)
	MarcarUsado(id string, quando time.Time) error
	// IncrementarTentativa soma 1 às tentativas de código erradas (anti-brute-force)
	IncrementarTentativa(id string) error
	// InvalidarPendentesDoUsuario encerra pedidos anteriores ainda pendentes
	InvalidarPendentesDoUsuario(usuarioID string) error
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de recuperações.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Criar(rec Recuperacao) error {
	query := `INSERT INTO tb_recuperacoes_senha (id, usuario_id, codigo_hash, status, expira_em, criado_em)
	          VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := r.db.Exec(query, rec.ID, rec.UsuarioID, rec.CodigoHash, rec.Status, rec.ExpiraEm, rec.CriadoEm); err != nil {
		return fmt.Errorf("erro ao inserir recuperação: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) BuscarPorToken(token string) (Recuperacao, error) {
	var rec Recuperacao
	query := `SELECT id, usuario_id, codigo_hash, status, tentativas, expira_em, criado_em, usado_em
	          FROM tb_recuperacoes_senha WHERE id = ?`
	if err := r.db.Get(&rec, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Recuperacao{}, fmt.Errorf("recuperação não encontrada")
		}
		return Recuperacao{}, fmt.Errorf("erro ao buscar recuperação: %w", err)
	}
	return rec, nil
}

func (r *repositorioMySQL) MarcarUsado(id string, quando time.Time) error {
	if _, err := r.db.Exec(`UPDATE tb_recuperacoes_senha SET status = ?, usado_em = ? WHERE id = ?`,
		StatusUsado, quando, id); err != nil {
		return fmt.Errorf("erro ao marcar recuperação como usada: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) IncrementarTentativa(id string) error {
	if _, err := r.db.Exec(`UPDATE tb_recuperacoes_senha SET tentativas = tentativas + 1 WHERE id = ?`, id); err != nil {
		return fmt.Errorf("erro ao incrementar tentativa: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) InvalidarPendentesDoUsuario(usuarioID string) error {
	if _, err := r.db.Exec(`UPDATE tb_recuperacoes_senha SET status = ? WHERE usuario_id = ? AND status = ?`,
		StatusUsado, usuarioID, StatusPendente); err != nil {
		return fmt.Errorf("erro ao invalidar recuperações anteriores: %w", err)
	}
	return nil
}
