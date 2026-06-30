// Pacote: internal/admin
// Arquivo: repository.go
// Descrição: Acesso a dados do painel de ADMIN. SÓ leitura agregada — não escreve nada.
//            Todas as métricas saem de tabelas que JÁ existem (tb_usuarios, tb_auditoria,
//            tb_onebyone, tb_agendamentos, tb_organizacoes, tb_equipes, tb_colaboradores),
//            sem nenhuma tabela ou caminho de escrita novo. As datas usam NOW()/CURDATE()
//            do MySQL (mesma convenção do resto do projeto, que assume app e banco no
//            mesmo relógio). Todas essas tabelas compartilham a collation utf8mb4_unicode_ci,
//            então os JOINs são seguros (sem a "divisão de collation" das tabelas novas).
// Autor: OneByOne API
// Criado em: 2026

package admin

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// ─── Linhas (espelhos das consultas) ──────────────────────────────────────────

// ContasRow são os totais de contas por papel + cadastros recentes.
type ContasRow struct {
	Total     int `db:"total"`
	Admin     int `db:"admin"`
	RH        int `db:"rh"`
	Gestores  int `db:"gestores"`
	Liderados int `db:"liderados"`
	Inativas  int `db:"inativas"`
	NovasHoje int `db:"novas_hoje"`
	Novas7d   int `db:"novas_7d"`
	Novas30d  int `db:"novas_30d"`
}

// EstruturaRow são os totais da estrutura (organização → equipe → colaborador).
type EstruturaRow struct {
	Organizacoes          int `db:"organizacoes"`
	Equipes               int `db:"equipes"`
	Colaboradores         int `db:"colaboradores"`
	ColaboradoresComConta int `db:"colaboradores_com_conta"`
}

// AtividadeRow são os totais de uso/atividade (1:1, agenda, logins e usuários ativos).
type AtividadeRow struct {
	OnebyonesTotal     int `db:"onebyones_total"`
	RealizadosTotal    int `db:"realizados_total"`
	Realizados30d      int `db:"realizados_30d"`
	AgendamentosAtivos int `db:"agendamentos_ativos"`
	LoginsHoje         int `db:"logins_hoje"`
	Logins7d           int `db:"logins_7d"`
	AtivosHoje         int `db:"ativos_hoje"`
	Ativos7d           int `db:"ativos_7d"`
	Ativos30d          int `db:"ativos_30d"`
	EventosHoje        int `db:"eventos_hoje"`
}

// ContaRow é uma conta na listagem do admin, com o resumo de uso de cada uma.
type ContaRow struct {
	ID            string     `db:"id"`
	Nome          string     `db:"nome"`
	Email         string     `db:"email"`
	Role          string     `db:"role"`
	CriadoEm      time.Time  `db:"criado_em"`
	UltimoAcesso  *time.Time `db:"ultimo_acesso"`
	TotalEventos  int        `db:"total_eventos"`
	Equipes       int        `db:"equipes"`
	Colaboradores int        `db:"colaboradores"`
	Onebyones     int        `db:"onebyones"`
	Gestores      int        `db:"gestores"`
}

// AcessoDiaRow é um ponto da série temporal de acessos (um dia).
type AcessoDiaRow struct {
	Dia     string `db:"dia"`
	Logins  int    `db:"logins"`
	Ativos  int    `db:"ativos"`
	Eventos int    `db:"eventos"`
}

// FuncRow é o uso de uma funcionalidade (ação + entidade) no período.
type FuncRow struct {
	Entidade string `db:"entidade"`
	Acao     string `db:"acao"`
	Total    int    `db:"total"`
}

// ContagemRow é um par rótulo→total genérico (hora do dia, dia da semana, papel).
type ContagemRow struct {
	Rotulo string `db:"rotulo"`
	Total  int    `db:"total"`
}

// CrescimentoRow é o número de cadastros de um papel em um dia.
type CrescimentoRow struct {
	Dia   string `db:"dia"`
	Role  string `db:"role"`
	Total int    `db:"total"`
}

// SerieDiaRow é um ponto genérico (dia → total) — usado nos 1:1 realizados por dia.
type SerieDiaRow struct {
	Dia   string `db:"dia"`
	Total int    `db:"total"`
}

// SaudeRow são os números crus para os indicadores de saúde da plataforma.
type SaudeRow struct {
	Gestores          int `db:"gestores"`
	GestoresCom1a1    int `db:"gestores_com_1a1"`
	ContasComIA       int `db:"contas_com_ia"`
	ContasComFoto     int `db:"contas_com_foto"`
	LideradosAtivos   int `db:"liderados_ativos"`
	LideradosComConta int `db:"liderados_com_conta"`
	Realizados30d     int `db:"realizados_30d"`
}

// GestorEngajamentoRow é um gestor no ranking de engajamento (mais 1:1 realizados).
type GestorEngajamentoRow struct {
	ID         string `db:"id"`
	Nome       string `db:"nome"`
	Email      string `db:"email"`
	Realizados int    `db:"realizados"`
	Liderados  int    `db:"liderados"`
}

// ─── Interface ────────────────────────────────────────────────────────────────

// Repositorio define o acesso a dados (somente leitura agregada) do painel de admin.
type Repositorio interface {
	ResumoContas() (ContasRow, error)
	ResumoEstrutura() (EstruturaRow, error)
	ResumoAtividade() (AtividadeRow, error)
	ListarContas(papel, busca string, limite, offset int) ([]ContaRow, int, error)
	SerieAcessos(dias int) ([]AcessoDiaRow, error)
	TopFuncionalidades(dias, limite int) ([]FuncRow, error)
	DistribuicaoHora(dias int) ([]ContagemRow, error)
	DistribuicaoDiaSemana(dias int) ([]ContagemRow, error)
	AtividadePorPapel(dias int) ([]ContagemRow, error)
	CrescimentoContas(dias int) ([]CrescimentoRow, error)
	SerieRealizados(dias int) ([]SerieDiaRow, error)
	SaudePlataforma() (SaudeRow, error)
	TopGestores(limite int) ([]GestorEngajamentoRow, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório MySQL do painel de admin.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// ─── Resumos (KPIs da visão geral) ────────────────────────────────────────────

// ResumoContas conta as contas ativas por papel e os cadastros recentes (hoje/7d/30d).
// Usa COUNT(CASE WHEN ...) — que retorna inteiro limpo — em vez de SUM(bool).
func (r *repositorioMySQL) ResumoContas() (ContasRow, error) {
	var row ContasRow
	const q = `
		SELECT
		  COUNT(*) AS total,
		  COUNT(CASE WHEN role = 'ADMIN'        THEN 1 END) AS admin,
		  COUNT(CASE WHEN role = 'RH'           THEN 1 END) AS rh,
		  COUNT(CASE WHEN role = 'LIDER'        THEN 1 END) AS gestores,
		  COUNT(CASE WHEN role = 'COLABORADOR'  THEN 1 END) AS liderados,
		  (SELECT COUNT(*) FROM tb_usuarios WHERE deletado_em IS NOT NULL) AS inativas,
		  COUNT(CASE WHEN criado_em >= CURDATE()                          THEN 1 END) AS novas_hoje,
		  COUNT(CASE WHEN criado_em >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)  THEN 1 END) AS novas_7d,
		  COUNT(CASE WHEN criado_em >= DATE_SUB(CURDATE(), INTERVAL 30 DAY) THEN 1 END) AS novas_30d
		FROM tb_usuarios
		WHERE deletado_em IS NULL`
	if err := r.db.Get(&row, q); err != nil {
		return ContasRow{}, fmt.Errorf("erro ao resumir contas: %w", err)
	}
	return row, nil
}

// ResumoEstrutura conta organizações, equipes e colaboradores ativos.
func (r *repositorioMySQL) ResumoEstrutura() (EstruturaRow, error) {
	var row EstruturaRow
	const q = `
		SELECT
		  (SELECT COUNT(*) FROM tb_organizacoes WHERE deletado_em IS NULL) AS organizacoes,
		  (SELECT COUNT(*) FROM tb_equipes      WHERE deletado_em IS NULL) AS equipes,
		  (SELECT COUNT(*) FROM tb_colaboradores WHERE deletado_em IS NULL AND desligado_em IS NULL) AS colaboradores,
		  (SELECT COUNT(*) FROM tb_colaboradores WHERE deletado_em IS NULL AND usuario_id IS NOT NULL) AS colaboradores_com_conta`
	if err := r.db.Get(&row, q); err != nil {
		return EstruturaRow{}, fmt.Errorf("erro ao resumir estrutura: %w", err)
	}
	return row, nil
}

// ResumoAtividade conta 1:1, agendamentos, logins e usuários ativos (DAU/WAU/MAU).
func (r *repositorioMySQL) ResumoAtividade() (AtividadeRow, error) {
	var row AtividadeRow
	const q = `
		SELECT
		  (SELECT COUNT(*) FROM tb_onebyone WHERE deletado_em IS NULL) AS onebyones_total,
		  (SELECT COUNT(*) FROM tb_onebyone WHERE status = 'REALIZADO' AND deletado_em IS NULL) AS realizados_total,
		  (SELECT COUNT(*) FROM tb_onebyone WHERE status = 'REALIZADO' AND deletado_em IS NULL
		      AND realizado_em >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)) AS realizados_30d,
		  (SELECT COUNT(*) FROM tb_agendamentos WHERE ativo = 1) AS agendamentos_ativos,
		  (SELECT COUNT(*) FROM tb_auditoria WHERE acao = 'LOGIN' AND criado_em >= CURDATE()) AS logins_hoje,
		  (SELECT COUNT(*) FROM tb_auditoria WHERE acao = 'LOGIN'
		      AND criado_em >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)) AS logins_7d,
		  (SELECT COUNT(DISTINCT usuario_id) FROM tb_auditoria WHERE usuario_id IS NOT NULL
		      AND criado_em >= CURDATE()) AS ativos_hoje,
		  (SELECT COUNT(DISTINCT usuario_id) FROM tb_auditoria WHERE usuario_id IS NOT NULL
		      AND criado_em >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)) AS ativos_7d,
		  (SELECT COUNT(DISTINCT usuario_id) FROM tb_auditoria WHERE usuario_id IS NOT NULL
		      AND criado_em >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)) AS ativos_30d,
		  (SELECT COUNT(*) FROM tb_auditoria WHERE criado_em >= CURDATE()) AS eventos_hoje`
	if err := r.db.Get(&row, q); err != nil {
		return AtividadeRow{}, fmt.Errorf("erro ao resumir atividade: %w", err)
	}
	return row, nil
}

// ─── Listagem de contas (com resumo de uso por conta) ─────────────────────────

// ListarContas devolve a página de contas ativas (filtros opcionais por papel e por
// busca em nome/e-mail) + o total para a paginação. Cada conta traz o último acesso e os
// contadores de uso (via subconsultas correlacionadas — o volume de dados é pequeno).
func (r *repositorioMySQL) ListarContas(papel, busca string, limite, offset int) ([]ContaRow, int, error) {
	// Monta os filtros dinamicamente, sempre com placeholders (sem concatenar valores).
	var cond []string
	var args []interface{}
	cond = append(cond, "u.deletado_em IS NULL")
	if papel != "" {
		cond = append(cond, "u.role = ?")
		args = append(args, papel)
	}
	if busca != "" {
		cond = append(cond, "(u.nome LIKE ? OR u.email LIKE ?)")
		curinga := "%" + busca + "%"
		args = append(args, curinga, curinga)
	}
	where := "WHERE " + strings.Join(cond, " AND ")

	// 1) Total para a paginação (mesmos filtros).
	var total int
	if err := r.db.Get(&total, "SELECT COUNT(*) FROM tb_usuarios u "+where, args...); err != nil {
		return nil, 0, fmt.Errorf("erro ao contar contas: %w", err)
	}

	// 2) A página em si.
	q := `
		SELECT u.id, u.nome, u.email, u.role, u.criado_em,
		  (SELECT MAX(a.criado_em) FROM tb_auditoria a WHERE a.usuario_id = u.id) AS ultimo_acesso,
		  (SELECT COUNT(*)         FROM tb_auditoria a WHERE a.usuario_id = u.id) AS total_eventos,
		  (SELECT COUNT(*) FROM tb_equipes e WHERE e.usuario_id = u.id AND e.deletado_em IS NULL) AS equipes,
		  (SELECT COUNT(*) FROM tb_colaboradores c
		      JOIN tb_equipes e ON e.id = c.equipe_id
		      WHERE e.usuario_id = u.id AND c.deletado_em IS NULL) AS colaboradores,
		  (SELECT COUNT(*) FROM tb_onebyone o WHERE o.usuario_id = u.id AND o.deletado_em IS NULL) AS onebyones,
		  (SELECT COUNT(*) FROM tb_usuarios g WHERE g.rh_id = u.id AND g.deletado_em IS NULL) AS gestores
		FROM tb_usuarios u
		` + where + `
		ORDER BY u.criado_em DESC
		LIMIT ? OFFSET ?`
	argsPagina := append(append([]interface{}{}, args...), limite, offset)

	var rows []ContaRow
	if err := r.db.Select(&rows, q, argsPagina...); err != nil {
		return nil, 0, fmt.Errorf("erro ao listar contas: %w", err)
	}
	return rows, total, nil
}

// ─── Séries temporais e distribuições (gráficos estilo Google Analytics) ──────

// SerieAcessos devolve, por dia (últimos `dias` dias), os logins, os usuários ativos
// (distintos) e o total de eventos. Dias sem evento NÃO vêm aqui — o usecase preenche
// os buracos com zero para o gráfico ficar contínuo.
func (r *repositorioMySQL) SerieAcessos(dias int) ([]AcessoDiaRow, error) {
	var rows []AcessoDiaRow
	const q = `
		SELECT DATE_FORMAT(criado_em, '%Y-%m-%d') AS dia,
		  COUNT(CASE WHEN acao = 'LOGIN' THEN 1 END) AS logins,
		  COUNT(DISTINCT CASE WHEN usuario_id IS NOT NULL THEN usuario_id END) AS ativos,
		  COUNT(*) AS eventos
		FROM tb_auditoria
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY dia
		ORDER BY dia ASC`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao montar série de acessos: %w", err)
	}
	return rows, nil
}

// TopFuncionalidades devolve as ações mais usadas (entidade + ação) no período.
func (r *repositorioMySQL) TopFuncionalidades(dias, limite int) ([]FuncRow, error) {
	var rows []FuncRow
	const q = `
		SELECT entidade, acao, COUNT(*) AS total
		FROM tb_auditoria
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		  AND entidade IS NOT NULL AND entidade <> ''
		GROUP BY entidade, acao
		ORDER BY total DESC
		LIMIT ?`
	if err := r.db.Select(&rows, q, dias-1, limite); err != nil {
		return nil, fmt.Errorf("erro ao listar funcionalidades mais usadas: %w", err)
	}
	return rows, nil
}

// DistribuicaoHora devolve a atividade por hora do dia (0–23) no período (heatmap simples).
// Agrupa pelo ALIAS `rotulo` (idêntico à expressão projetada) para satisfazer o sql_mode
// only_full_group_by do MySQL 8 — agrupar por HOUR(criado_em) com SELECT CAST(HOUR(...))
// é tratado como expressão diferente e o banco recusa. A ordenação fica por conta do
// usecase, que reencaixa os totais em 24 baldes fixos (0–23).
func (r *repositorioMySQL) DistribuicaoHora(dias int) ([]ContagemRow, error) {
	var rows []ContagemRow
	const q = `
		SELECT CAST(HOUR(criado_em) AS CHAR) AS rotulo, COUNT(*) AS total
		FROM tb_auditoria
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY rotulo`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao distribuir por hora: %w", err)
	}
	return rows, nil
}

// DistribuicaoDiaSemana devolve a atividade por dia da semana (1=domingo … 7=sábado, padrão
// do DAYOFWEEK do MySQL) no período. Agrupa pelo ALIAS `rotulo` (mesmo motivo do
// only_full_group_by descrito em DistribuicaoHora); o usecase converte o número no nome do
// dia e reencaixa em 7 baldes fixos.
func (r *repositorioMySQL) DistribuicaoDiaSemana(dias int) ([]ContagemRow, error) {
	var rows []ContagemRow
	const q = `
		SELECT CAST(DAYOFWEEK(criado_em) AS CHAR) AS rotulo, COUNT(*) AS total
		FROM tb_auditoria
		WHERE criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY rotulo`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao distribuir por dia da semana: %w", err)
	}
	return rows, nil
}

// AtividadePorPapel devolve quantos eventos cada papel gerou no período (eventos sem
// usuário identificado caem em 'ANONIMO' — ex.: logins antigos sem atribuição).
func (r *repositorioMySQL) AtividadePorPapel(dias int) ([]ContagemRow, error) {
	var rows []ContagemRow
	const q = `
		SELECT COALESCE(u.role, 'ANONIMO') AS rotulo, COUNT(*) AS total
		FROM tb_auditoria a
		LEFT JOIN tb_usuarios u ON u.id = a.usuario_id
		WHERE a.criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY COALESCE(u.role, 'ANONIMO')
		ORDER BY total DESC`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao distribuir atividade por papel: %w", err)
	}
	return rows, nil
}

// ─── Crescimento ──────────────────────────────────────────────────────────────

// CrescimentoContas devolve os novos cadastros por dia e por papel no período.
func (r *repositorioMySQL) CrescimentoContas(dias int) ([]CrescimentoRow, error) {
	var rows []CrescimentoRow
	const q = `
		SELECT DATE_FORMAT(criado_em, '%Y-%m-%d') AS dia, role, COUNT(*) AS total
		FROM tb_usuarios
		WHERE deletado_em IS NULL
		  AND criado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY dia, role
		ORDER BY dia ASC`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao montar crescimento de contas: %w", err)
	}
	return rows, nil
}

// SerieRealizados devolve os 1:1 REALIZADOS por dia no período.
func (r *repositorioMySQL) SerieRealizados(dias int) ([]SerieDiaRow, error) {
	var rows []SerieDiaRow
	const q = `
		SELECT DATE_FORMAT(realizado_em, '%Y-%m-%d') AS dia, COUNT(*) AS total
		FROM tb_onebyone
		WHERE status = 'REALIZADO' AND deletado_em IS NULL AND realizado_em IS NOT NULL
		  AND realizado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY dia
		ORDER BY dia ASC`
	if err := r.db.Select(&rows, q, dias-1); err != nil {
		return nil, fmt.Errorf("erro ao montar série de 1:1 realizados: %w", err)
	}
	return rows, nil
}

// ─── Saúde da plataforma ──────────────────────────────────────────────────────

// SaudePlataforma traz os números crus dos indicadores de engajamento/adoção. O usecase
// deriva as taxas (médias e percentuais) a partir deles.
func (r *repositorioMySQL) SaudePlataforma() (SaudeRow, error) {
	var row SaudeRow
	const q = `
		SELECT
		  (SELECT COUNT(*) FROM tb_usuarios WHERE role = 'LIDER' AND deletado_em IS NULL) AS gestores,
		  -- Numerador alinhado ao MESMO recorte do denominador (gestor LIDER ativo): sem o
		  -- JOIN, um dono de 1:1 que mudou de papel ou foi removido contaria aqui mas nao no
		  -- total de gestores, podendo gerar adocao acima de 100 por cento.
		  (SELECT COUNT(DISTINCT o.usuario_id) FROM tb_onebyone o
		      JOIN tb_usuarios u ON u.id = o.usuario_id
		      WHERE o.status = 'REALIZADO' AND o.deletado_em IS NULL
		        AND u.role = 'LIDER' AND u.deletado_em IS NULL) AS gestores_com_1a1,
		  (SELECT COUNT(*) FROM tb_usuarios WHERE deletado_em IS NULL AND ia_provedor IS NOT NULL AND ia_provedor <> '') AS contas_com_ia,
		  (SELECT COUNT(*) FROM tb_usuarios WHERE deletado_em IS NULL AND foto_key IS NOT NULL) AS contas_com_foto,
		  (SELECT COUNT(*) FROM tb_colaboradores WHERE deletado_em IS NULL AND desligado_em IS NULL) AS liderados_ativos,
		  (SELECT COUNT(*) FROM tb_colaboradores WHERE deletado_em IS NULL AND desligado_em IS NULL AND usuario_id IS NOT NULL) AS liderados_com_conta,
		  (SELECT COUNT(*) FROM tb_onebyone WHERE status = 'REALIZADO' AND deletado_em IS NULL
		      AND realizado_em >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)) AS realizados_30d`
	if err := r.db.Get(&row, q); err != nil {
		return SaudeRow{}, fmt.Errorf("erro ao calcular saúde da plataforma: %w", err)
	}
	return row, nil
}

// TopGestores devolve o ranking de gestores por 1:1 realizados (engajamento).
func (r *repositorioMySQL) TopGestores(limite int) ([]GestorEngajamentoRow, error) {
	var rows []GestorEngajamentoRow
	const q = `
		SELECT u.id, u.nome, u.email,
		  (SELECT COUNT(*) FROM tb_onebyone o WHERE o.usuario_id = u.id AND o.status = 'REALIZADO' AND o.deletado_em IS NULL) AS realizados,
		  (SELECT COUNT(*) FROM tb_colaboradores c
		      JOIN tb_equipes e ON e.id = c.equipe_id
		      WHERE e.usuario_id = u.id AND c.deletado_em IS NULL) AS liderados
		FROM tb_usuarios u
		WHERE u.role = 'LIDER' AND u.deletado_em IS NULL
		ORDER BY realizados DESC, liderados DESC
		LIMIT ?`
	if err := r.db.Select(&rows, q, limite); err != nil {
		return nil, fmt.Errorf("erro ao listar top gestores: %w", err)
	}
	return rows, nil
}
