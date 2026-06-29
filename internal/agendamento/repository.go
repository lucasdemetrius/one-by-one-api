// Pacote: internal/agendamento
// Arquivo: repository.go
// Descrição: Interface Repositorio e implementação MySQL (sqlx) da tabela
//            tb_agendamentos. Apenas I/O de banco.
// Autor: OneByOne API
// Criado em: 2025

package agendamento

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define as operações de persistência dos agendamentos.
type Repositorio interface {
	Criar(a Agendamento) error
	// ListarPorUsuario traz os agendamentos ativos de um gestor (com nome do liderado)
	ListarPorUsuario(usuarioID string) ([]AgendamentoContexto, error)
	// ListarParaLembrete traz todos os agendamentos ativos com nomes/e-mail (scheduler)
	ListarParaLembrete() ([]AgendamentoContexto, error)
	// Deletar remove um agendamento garantindo que pertence ao gestor
	Deletar(id, usuarioID string) error
	// AtualizarDataHora avança a próxima ocorrência (recorrência)
	AtualizarDataHora(id string, dh time.Time) error
	// Reagendar muda a data/hora garantindo que o agendamento é do gestor (drag-reschedule)
	Reagendar(id, usuarioID string, dh time.Time) (bool, error)
	// Desativar marca o agendamento como inativo (recorrência NENHUMA já passou)
	Desativar(id string) error
	// DeletarPorColaborador remove TODOS os agendamentos de um liderado (do gestor dono)
	DeletarPorColaborador(colaboradorID, usuarioID string) (int64, error)
	// BuscarPorId traz um agendamento do gestor (com nome do liderado) — ok=false se não há
	BuscarPorId(id, usuarioID string) (AgendamentoContexto, bool, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de agendamentos.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Criar(a Agendamento) error {
	query := `INSERT INTO tb_agendamentos (id, usuario_id, colaborador_id, data_hora, recorrencia, repete_ate, ativo, criado_em)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.Exec(query, a.ID, a.UsuarioID, a.ColaboradorID, a.DataHora, a.Recorrencia, a.RepeteAte, a.Ativo, a.CriadoEm); err != nil {
		return fmt.Errorf("erro ao inserir agendamento: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ListarPorUsuario(usuarioID string) ([]AgendamentoContexto, error) {
	var lista []AgendamentoContexto
	query := `SELECT a.id, a.usuario_id, a.colaborador_id, c.nome AS liderado_nome,
	                 a.data_hora, a.recorrencia, a.repete_ate
	          FROM tb_agendamentos a
	          JOIN tb_colaboradores c ON c.id = a.colaborador_id
	          WHERE a.usuario_id = ? AND a.ativo = 1
	          ORDER BY a.data_hora ASC`
	if err := r.db.Select(&lista, query, usuarioID); err != nil {
		return nil, fmt.Errorf("erro ao listar agendamentos: %w", err)
	}
	return lista, nil
}

func (r *repositorioMySQL) ListarParaLembrete() ([]AgendamentoContexto, error) {
	var lista []AgendamentoContexto
	query := `SELECT a.id, a.usuario_id, a.colaborador_id,
	                 u.nome AS gestor_nome, u.email AS gestor_email,
	                 c.nome AS liderado_nome, a.data_hora, a.recorrencia, a.repete_ate
	          FROM tb_agendamentos a
	          JOIN tb_usuarios u ON u.id = a.usuario_id
	          JOIN tb_colaboradores c ON c.id = a.colaborador_id
	          WHERE a.ativo = 1 AND u.deletado_em IS NULL AND c.deletado_em IS NULL`
	if err := r.db.Select(&lista, query); err != nil {
		return nil, fmt.Errorf("erro ao listar agendamentos para lembrete: %w", err)
	}
	return lista, nil
}

func (r *repositorioMySQL) Deletar(id, usuarioID string) error {
	if _, err := r.db.Exec(`DELETE FROM tb_agendamentos WHERE id = ? AND usuario_id = ?`, id, usuarioID); err != nil {
		return fmt.Errorf("erro ao deletar agendamento: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) AtualizarDataHora(id string, dh time.Time) error {
	if _, err := r.db.Exec(`UPDATE tb_agendamentos SET data_hora = ? WHERE id = ?`, dh, id); err != nil {
		return fmt.Errorf("erro ao atualizar data do agendamento: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) Reagendar(id, usuarioID string, dh time.Time) (bool, error) {
	resultado, err := r.db.Exec(
		`UPDATE tb_agendamentos SET data_hora = ? WHERE id = ? AND usuario_id = ?`,
		dh, id, usuarioID,
	)
	if err != nil {
		return false, fmt.Errorf("erro ao reagendar: %w", err)
	}
	linhas, _ := resultado.RowsAffected()
	return linhas > 0, nil
}

func (r *repositorioMySQL) Desativar(id string) error {
	if _, err := r.db.Exec(`UPDATE tb_agendamentos SET ativo = 0 WHERE id = ?`, id); err != nil {
		return fmt.Errorf("erro ao desativar agendamento: %w", err)
	}
	return nil
}

// BuscarPorId traz um agendamento do gestor (com nome do liderado), para montar o aviso
// por e-mail ao cancelar/remarcar. ok=false se o agendamento não existe / não é do gestor.
func (r *repositorioMySQL) BuscarPorId(id, usuarioID string) (AgendamentoContexto, bool, error) {
	var a AgendamentoContexto
	query := `SELECT a.id, a.usuario_id, a.colaborador_id, c.nome AS liderado_nome,
	                 a.data_hora, a.recorrencia, a.repete_ate
	          FROM tb_agendamentos a
	          JOIN tb_colaboradores c ON c.id = a.colaborador_id
	          WHERE a.id = ? AND a.usuario_id = ?`
	if err := r.db.Get(&a, query, id, usuarioID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return a, false, nil
		}
		return a, false, fmt.Errorf("erro ao buscar agendamento: %w", err)
	}
	return a, true, nil
}

// DeletarPorColaborador remove de uma vez todos os agendamentos de um liderado. O filtro
// por usuario_id garante que um gestor só cancela a agenda dos SEUS liderados.
func (r *repositorioMySQL) DeletarPorColaborador(colaboradorID, usuarioID string) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM tb_agendamentos WHERE colaborador_id = ? AND usuario_id = ?`, colaboradorID, usuarioID)
	if err != nil {
		return 0, fmt.Errorf("erro ao cancelar agendamentos do liderado: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
