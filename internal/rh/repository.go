// Pacote: internal/rh
// Arquivo: repository.go
// Descrição: Acesso a dados do módulo de RH. Lê os gestores do tenant (tb_usuarios
//            filtrado por rh_id) e confere se um gestor pertence ao tenant do RH
//            (reusando a primitiva compartilhada de autorização).
// Autor: OneByOne API
// Criado em: 2026

package rh

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/autorizacao"
)

// GestorRow espelha as colunas lidas de tb_usuarios para a lista de gestores do RH.
type GestorRow struct {
	ID       string    `db:"id"`
	Nome     string    `db:"nome"`
	Email    string    `db:"email"`
	CriadoEm time.Time `db:"criado_em"`
}

// AgendaRow espelha um 1:1 da agenda consolidada do tenant (com gestor/equipe).
type AgendaRow struct {
	ID            string     `db:"id"`
	GestorID      string     `db:"gestor_id"`
	GestorNome    string     `db:"gestor_nome"`
	ColaboradorID string     `db:"colaborador_id"`
	LideradoNome  string     `db:"liderado_nome"`
	EquipeID      *string    `db:"equipe_id"`
	EquipeNome    *string    `db:"equipe_nome"`
	DataHora      time.Time  `db:"data_hora"`
	Recorrencia   string     `db:"recorrencia"`
	RepeteAte     *time.Time `db:"repete_ate"`
}

// MatrixRow espelha um liderado da 9-box consolidada do tenant (com gestor/equipe/classificação).
type MatrixRow struct {
	ColaboradorID string  `db:"colaborador_id"`
	LideradoNome  string  `db:"liderado_nome"`
	GestorID      string  `db:"gestor_id"`
	GestorNome    string  `db:"gestor_nome"`
	EquipeID      *string `db:"equipe_id"`
	EquipeNome    *string `db:"equipe_nome"`
	Desempenho    *string `db:"desempenho"`
	Potencial     *string `db:"potencial"`
}

// HumorRow é um ponto de humor (SENTIMENTO) de um liderado, já com o gestor dono. Vêm
// ordenados por liderado e data (antigo→recente) para o cálculo de tendência.
type HumorRow struct {
	GestorID   string `db:"gestor_id"`
	LideradoID string `db:"liderado_id"`
	Valor      int    `db:"valor"`
}

// PdiRow é um item de PDI de um liderado, com o gestor dono.
type PdiRow struct {
	GestorID   string     `db:"gestor_id"`
	LideradoID string     `db:"liderado_id"`
	Concluido  bool       `db:"concluido"`
	Prazo      *time.Time `db:"prazo"`
}

// Repositorio define o acesso a dados do módulo de RH.
type Repositorio interface {
	// ListarGestores retorna os gestores (LIDER) ativos vinculados a este RH (rh_id = rhID)
	ListarGestores(rhID string) ([]GestorRow, error)
	// GestorPertenceAoRH diz se o gestor pertence ao tenant do RH (fronteira do tenant)
	GestorPertenceAoRH(gestorID, rhID string) (bool, error)
	// ListarAgendaDoTenant traz todos os 1:1 ativos dos gestores do RH (com gestor/equipe)
	ListarAgendaDoTenant(rhID string) ([]AgendaRow, error)
	// ListarMatrixDoTenant traz todos os liderados ativos do RH com a 9-box (gestor/equipe)
	ListarMatrixDoTenant(rhID string) ([]MatrixRow, error)
	// ListarHumorDoTenant traz os pontos de humor (SENTIMENTO) de todos os liderados do RH
	ListarHumorDoTenant(rhID string) ([]HumorRow, error)
	// ListarPdiDoTenant traz os itens de PDI de todos os liderados do RH
	ListarPdiDoTenant(rhID string) ([]PdiRow, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório MySQL do módulo de RH.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// ListarGestores devolve os gestores ativos cujo rh_id é o RH informado, ordenados por nome.
func (r *repositorioMySQL) ListarGestores(rhID string) ([]GestorRow, error) {
	var gs []GestorRow
	query := `
		SELECT id, nome, email, criado_em
		FROM tb_usuarios
		WHERE rh_id = ? AND role = 'LIDER' AND deletado_em IS NULL
		ORDER BY nome ASC
	`
	if err := r.db.Select(&gs, query, rhID); err != nil {
		return nil, fmt.Errorf("erro ao listar gestores do RH: %w", err)
	}
	return gs, nil
}

// GestorPertenceAoRH delega à primitiva compartilhada (gestor.rh_id == rhID e rhID é RH).
func (r *repositorioMySQL) GestorPertenceAoRH(gestorID, rhID string) (bool, error) {
	return autorizacao.GestorPertenceAoRH(r.db, gestorID, rhID)
}

// ListarAgendaDoTenant traz todos os 1:1 ativos cujos gestores pertencem a este RH
// (g.rh_id = rhID), com o nome do gestor, do liderado e da equipe do liderado.
func (r *repositorioMySQL) ListarAgendaDoTenant(rhID string) ([]AgendaRow, error) {
	var rows []AgendaRow
	query := `
		SELECT a.id, a.usuario_id AS gestor_id, g.nome AS gestor_nome,
		       a.colaborador_id, c.nome AS liderado_nome,
		       c.equipe_id AS equipe_id, e.nome AS equipe_nome,
		       a.data_hora, a.recorrencia, a.repete_ate
		FROM tb_agendamentos a
		JOIN tb_usuarios g ON g.id = a.usuario_id
		JOIN tb_colaboradores c ON c.id = a.colaborador_id
		LEFT JOIN tb_equipes e ON e.id = c.equipe_id
		WHERE g.rh_id = ? AND a.ativo = 1 AND c.deletado_em IS NULL
		ORDER BY a.data_hora ASC
	`
	if err := r.db.Select(&rows, query, rhID); err != nil {
		return nil, fmt.Errorf("erro ao listar agenda do tenant: %w", err)
	}
	return rows, nil
}

// ListarMatrixDoTenant traz todos os liderados ativos (não desligados) dos gestores deste
// RH, com gestor/equipe e a classificação 9-box (desempenho × potencial), quando houver.
// O gestor do liderado é resolvido pela equipe (e.usuario_id).
func (r *repositorioMySQL) ListarMatrixDoTenant(rhID string) ([]MatrixRow, error) {
	var rows []MatrixRow
	query := `
		SELECT c.id AS colaborador_id, c.nome AS liderado_nome,
		       e.usuario_id AS gestor_id, g.nome AS gestor_nome,
		       c.equipe_id AS equipe_id, e.nome AS equipe_nome,
		       cl.desempenho, cl.potencial
		FROM tb_colaboradores c
		JOIN tb_equipes e ON e.id = c.equipe_id
		JOIN tb_usuarios g ON g.id = e.usuario_id
		LEFT JOIN tb_classificacoes cl ON cl.colaborador_id = c.id
		WHERE g.rh_id = ? AND c.deletado_em IS NULL AND c.desligado_em IS NULL
		ORDER BY g.nome, c.nome
	`
	if err := r.db.Select(&rows, query, rhID); err != nil {
		return nil, fmt.Errorf("erro ao listar matrix do tenant: %w", err)
	}
	return rows, nil
}

// ListarHumorDoTenant traz cada ponto de humor (SENTIMENTO) dos liderados ativos dos
// gestores deste RH, em ordem antigo→recente por liderado, para o cálculo de tendência.
func (r *repositorioMySQL) ListarHumorDoTenant(rhID string) ([]HumorRow, error) {
	var rows []HumorRow
	query := `
		SELECT e.usuario_id AS gestor_id, c.id AS liderado_id, a.valor
		FROM tb_colaboradores c
		JOIN tb_equipes e ON e.id = c.equipe_id
		JOIN tb_usuarios g ON g.id = e.usuario_id
		JOIN tb_acompanhamentos a ON a.colaborador_id COLLATE utf8mb4_unicode_ci = c.id
		WHERE g.rh_id = ? AND c.deletado_em IS NULL AND c.desligado_em IS NULL
		  AND a.tipo = 'SENTIMENTO' AND a.valor IS NOT NULL AND a.deletado_em IS NULL
		ORDER BY c.id, a.data_ref ASC, a.criado_em ASC
	`
	if err := r.db.Select(&rows, query, rhID); err != nil {
		return nil, fmt.Errorf("erro ao listar humor do tenant: %w", err)
	}
	return rows, nil
}

// ListarPdiDoTenant traz cada item de PDI dos liderados ativos dos gestores deste RH.
func (r *repositorioMySQL) ListarPdiDoTenant(rhID string) ([]PdiRow, error) {
	var rows []PdiRow
	query := `
		SELECT e.usuario_id AS gestor_id, c.id AS liderado_id, p.concluido, p.prazo
		FROM tb_colaboradores c
		JOIN tb_equipes e ON e.id = c.equipe_id
		JOIN tb_usuarios g ON g.id = e.usuario_id
		JOIN tb_pdi_itens p ON p.colaborador_id COLLATE utf8mb4_unicode_ci = c.id AND p.deletado_em IS NULL
		WHERE g.rh_id = ? AND c.deletado_em IS NULL AND c.desligado_em IS NULL
	`
	if err := r.db.Select(&rows, query, rhID); err != nil {
		return nil, fmt.Errorf("erro ao listar PDI do tenant: %w", err)
	}
	return rows, nil
}
