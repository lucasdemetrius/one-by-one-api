// Pacote: internal/notificacao
// Arquivo: repository.go
// Descrição: Persistência das notificações e preferências. CriarSeNova usa a chave
//            única para deduplicar (INSERT IGNORE). A leitura da agenda pendente
//            junta agendamento → gestor → liderado (mesmos JOINs do scheduler de
//            e-mail, que já funcionam).
// Autor: OneByOne API
// Criado em: 2026

package notificacao

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o acesso ao banco das notificações.
type Repositorio interface {
	// CriarSeNova insere a notificação; se a chave já existe (dedupe), ignora.
	CriarSeNova(n Notificacao) error
	ListarPorUsuario(usuarioID string, limite int) ([]Notificacao, error)
	ContarNaoLidas(usuarioID string) (int, error)
	MarcarLida(id, usuarioID string) error
	MarcarTodasLidas(usuarioID string) error
	ObterPref(usuarioID string) (Pref, error)
	SalvarPref(p Pref) error
	// ListarAgendaPendente traz os agendamentos ativos com gestor + liderado (cron).
	ListarAgendaPendente() ([]AgendaPendente, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de notificações.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) CriarSeNova(n Notificacao) error {
	query := `INSERT IGNORE INTO tb_notificacoes (id, usuario_id, tipo, titulo, mensagem, link, chave, lida, criado_em)
	          VALUES (:id, :usuario_id, :tipo, :titulo, :mensagem, :link, :chave, :lida, :criado_em)`
	if _, err := r.db.NamedExec(query, n); err != nil {
		return fmt.Errorf("erro ao criar notificação: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ListarPorUsuario(usuarioID string, limite int) ([]Notificacao, error) {
	var lista []Notificacao
	query := `SELECT id, usuario_id, tipo, titulo, mensagem, link, chave, lida, criado_em
	          FROM tb_notificacoes WHERE usuario_id = ?
	          ORDER BY criado_em DESC LIMIT ?`
	if err := r.db.Select(&lista, query, usuarioID, limite); err != nil {
		return nil, fmt.Errorf("erro ao listar notificações: %w", err)
	}
	return lista, nil
}

func (r *repositorioMySQL) ContarNaoLidas(usuarioID string) (int, error) {
	var n int
	if err := r.db.Get(&n, `SELECT COUNT(*) FROM tb_notificacoes WHERE usuario_id = ? AND lida = 0`, usuarioID); err != nil {
		return 0, fmt.Errorf("erro ao contar notificações: %w", err)
	}
	return n, nil
}

func (r *repositorioMySQL) MarcarLida(id, usuarioID string) error {
	if _, err := r.db.Exec(`UPDATE tb_notificacoes SET lida = 1 WHERE id = ? AND usuario_id = ?`, id, usuarioID); err != nil {
		return fmt.Errorf("erro ao marcar notificação: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) MarcarTodasLidas(usuarioID string) error {
	if _, err := r.db.Exec(`UPDATE tb_notificacoes SET lida = 1 WHERE usuario_id = ? AND lida = 0`, usuarioID); err != nil {
		return fmt.Errorf("erro ao marcar notificações: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ObterPref(usuarioID string) (Pref, error) {
	var p Pref
	query := `SELECT usuario_id, agenda_1dia, agenda_hoje, agenda_1h FROM tb_pref_notificacoes WHERE usuario_id = ?`
	if err := r.db.Get(&p, query, usuarioID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PrefPadrao(usuarioID), nil // sem registro = tudo ligado
		}
		return Pref{}, fmt.Errorf("erro ao obter preferências: %w", err)
	}
	return p, nil
}

func (r *repositorioMySQL) SalvarPref(p Pref) error {
	query := `INSERT INTO tb_pref_notificacoes (usuario_id, agenda_1dia, agenda_hoje, agenda_1h, alterado_em)
	          VALUES (:usuario_id, :agenda_1dia, :agenda_hoje, :agenda_1h, :alterado_em)
	          ON DUPLICATE KEY UPDATE agenda_1dia = VALUES(agenda_1dia), agenda_hoje = VALUES(agenda_hoje),
	                                  agenda_1h = VALUES(agenda_1h), alterado_em = VALUES(alterado_em)`
	type linha struct {
		Pref
		AlteradoEm time.Time `db:"alterado_em"`
	}
	if _, err := r.db.NamedExec(query, linha{Pref: p, AlteradoEm: time.Now()}); err != nil {
		return fmt.Errorf("erro ao salvar preferências: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ListarAgendaPendente() ([]AgendaPendente, error) {
	var lista []AgendaPendente
	query := `SELECT a.id AS agendamento_id, a.usuario_id AS gestor_id,
	                 u.nome AS gestor_nome, c.usuario_id AS liderado_usuario,
	                 c.nome AS liderado_nome, a.data_hora, a.recorrencia
	          FROM tb_agendamentos a
	          JOIN tb_usuarios u ON u.id = a.usuario_id
	          JOIN tb_colaboradores c ON c.id = a.colaborador_id
	          WHERE a.ativo = 1 AND u.deletado_em IS NULL AND c.deletado_em IS NULL`
	if err := r.db.Select(&lista, query); err != nil {
		return nil, fmt.Errorf("erro ao listar agenda pendente: %w", err)
	}
	return lista, nil
}
