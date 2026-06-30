// Pacote: internal/feedback
// Arquivo: repository.go
// Descrição: Acesso a dados de tb_feedbacks. Grava a reação (escrita) e faz as leituras
//            agregadas do painel de ADMIN (totais, série temporal, por contexto e
//            comentários recentes). O JOIN com tb_usuarios é seguro: ambas as tabelas são
//            utf8mb4_unicode_ci (a 024 fixa a collation de propósito).
// Autor: OneByOne API
// Criado em: 2026

package feedback

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ─── Linhas das consultas agregadas ───────────────────────────────────────────

// ResumoFeedbackRow são os totais por reação no período.
type ResumoFeedbackRow struct {
	Total    int `db:"total"`
	Curti    int `db:"curti"`
	NaoCurti int `db:"nao_curti"`
	Irritado int `db:"irritado"`
}

// SerieFeedbackRow é a contagem de uma reação em um dia.
type SerieFeedbackRow struct {
	Dia    string `db:"dia"`
	Reacao string `db:"reacao"`
	Total  int    `db:"total"`
}

// ContextoRow são as reações de um contexto/tela.
type ContextoRow struct {
	Contexto string `db:"contexto"`
	Curti    int    `db:"curti"`
	NaoCurti int    `db:"nao_curti"`
	Irritado int    `db:"irritado"`
	Total    int    `db:"total"`
}

// ComentarioRow é um feedback recente com comentário, já com o autor.
type ComentarioRow struct {
	Reacao     string    `db:"reacao"`
	Contexto   *string   `db:"contexto"`
	Comentario string    `db:"comentario"`
	CriadoEm   time.Time `db:"criado_em"`
	AutorNome  string    `db:"autor_nome"`
	AutorPapel string    `db:"autor_papel"`
}

// Repositorio define o acesso a dados do módulo de feedback.
type Repositorio interface {
	// Criar grava uma reação (escrita).
	Criar(f Feedback) error
	// Resumo devolve os totais por reação no período.
	Resumo(dias int) (ResumoFeedbackRow, error)
	// Serie devolve as reações por dia (para o usecase pivotar/preencher buracos).
	Serie(dias int) ([]SerieFeedbackRow, error)
	// PorContexto devolve as reações agrupadas por contexto/tela.
	PorContexto(dias int) ([]ContextoRow, error)
	// Recentes devolve os feedbacks COM comentário mais recentes, com o autor.
	Recentes(dias, limite int) ([]ComentarioRow, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório MySQL de feedback.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// Criar insere uma reação. Os campos opcionais (*string) viram NULL quando nil.
func (r *repositorioMySQL) Criar(f Feedback) error {
	const q = `
		INSERT INTO tb_feedbacks (id, usuario_id, reacao, contexto, comentario, pagina, criado_em)
		VALUES (:id, :usuario_id, :reacao, :contexto, :comentario, :pagina, :criado_em)`
	if _, err := r.db.NamedExec(q, f); err != nil {
		return fmt.Errorf("erro ao gravar feedback: %w", err)
	}
	return nil
}

// Resumo conta as reações por tipo no período (COUNT(CASE) → inteiro limpo).
func (r *repositorioMySQL) Resumo(dias int) (ResumoFeedbackRow, error) {
	var row ResumoFeedbackRow
	const q = `
		SELECT
		  COUNT(*) AS total,
		  COUNT(CASE WHEN reacao = 'CURTI'     THEN 1 END) AS curti,
		  COUNT(CASE WHEN reacao = 'NAO_CURTI' THEN 1 END) AS nao_curti,
		  COUNT(CASE WHEN reacao = 'IRRITADO'  THEN 1 END) AS irritado
		FROM tb_feedbacks
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)`
	if err := r.db.Get(&row, q, dias-1); err != nil {
		return ResumoFeedbackRow{}, fmt.Errorf("erro ao resumir feedback: %w", err)
	}
	return row, nil
}

// Serie devolve (dia, reacao, total) no período; o usecase preenche os dias sem reação.
func (r *repositorioMySQL) Serie(dias int) ([]SerieFeedbackRow, error) {
	var rows []SerieFeedbackRow
	const q = `
		SELECT DATE_FORMAT(criado_em, '%Y-%m-%d') AS dia, reacao, COUNT(*) AS total
		FROM tb_feedbacks
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY dia, reacao
		ORDER BY dia ASC`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao montar série de feedback: %w", err)
	}
	return rows, nil
}

// PorContexto agrupa as reações por contexto/tela (NULL/” viram '(sem contexto)').
func (r *repositorioMySQL) PorContexto(dias int) ([]ContextoRow, error) {
	var rows []ContextoRow
	const q = `
		SELECT COALESCE(NULLIF(contexto, ''), '(sem contexto)') AS contexto,
		  COUNT(CASE WHEN reacao = 'CURTI'     THEN 1 END) AS curti,
		  COUNT(CASE WHEN reacao = 'NAO_CURTI' THEN 1 END) AS nao_curti,
		  COUNT(CASE WHEN reacao = 'IRRITADO'  THEN 1 END) AS irritado,
		  COUNT(*) AS total
		FROM tb_feedbacks
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY COALESCE(NULLIF(contexto, ''), '(sem contexto)')
		ORDER BY total DESC
		LIMIT 30`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao agrupar feedback por contexto: %w", err)
	}
	return rows, nil
}

// Recentes devolve os feedbacks COM comentário mais recentes, com nome e papel do autor.
func (r *repositorioMySQL) Recentes(dias, limite int) ([]ComentarioRow, error) {
	var rows []ComentarioRow
	const q = `
		SELECT f.reacao, f.contexto, f.comentario, f.criado_em,
		       u.nome AS autor_nome, u.role AS autor_papel
		FROM tb_feedbacks f
		JOIN tb_usuarios u ON u.id = f.usuario_id
		WHERE f.comentario IS NOT NULL AND f.comentario <> ''
		  AND f.criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		ORDER BY f.criado_em DESC
		LIMIT ?`
	if err := r.db.Select(&rows, q, dias-1, limite); err != nil {
		return nil, fmt.Errorf("erro ao listar comentários de feedback: %w", err)
	}
	return rows, nil
}
