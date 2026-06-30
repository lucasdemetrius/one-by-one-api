// Pacote: internal/admin
// Arquivo: dto.go
// Descrição: Contratos HTTP do painel de ADMIN. Tudo pensado para o front montar
//            gráficos e cartões diretamente (séries com rótulos prontos, percentuais
//            já calculados). Nada aqui expõe dado sensível — são números agregados da
//            plataforma (sem senhas, sem conteúdo de 1:1).
// Autor: OneByOne API
// Criado em: 2026

package admin

import "time"

// ─── Visão geral (cartões de KPI no topo do dashboard) ────────────────────────

// VisaoGeralDTO é o resumo executivo da plataforma (a primeira tela do admin).
type VisaoGeralDTO struct {
	Contas    ContasResumoDTO `json:"contas"`
	Estrutura EstruturaDTO    `json:"estrutura"`
	Atividade AtividadeDTO    `json:"atividade"`
	GeradoEm  time.Time       `json:"gerado_em"`
}

// ContasResumoDTO resume as contas por papel + cadastros recentes.
type ContasResumoDTO struct {
	Total     int `json:"total"`     // contas ativas
	Admin     int `json:"admin"`     // ADMIN
	RH        int `json:"rh"`        // RH
	Gestores  int `json:"gestores"`  // LIDER
	Liderados int `json:"liderados"` // COLABORADOR
	Inativas  int `json:"inativas"`  // contas com soft delete
	NovasHoje int `json:"novas_hoje"`
	Novas7d   int `json:"novas_7d"`
	Novas30d  int `json:"novas_30d"`
}

// EstruturaDTO resume a árvore organização → equipe → colaborador.
type EstruturaDTO struct {
	Organizacoes          int `json:"organizacoes"`
	Equipes               int `json:"equipes"`
	Colaboradores         int `json:"colaboradores"`           // ativos (não desligados)
	ColaboradoresComConta int `json:"colaboradores_com_conta"` // liderados que já aceitaram o convite
}

// AtividadeDTO resume o uso recente (1:1, agenda, logins, usuários ativos).
type AtividadeDTO struct {
	OnebyonesTotal     int `json:"onebyones_total"`
	RealizadosTotal    int `json:"realizados_total"`
	Realizados30d      int `json:"realizados_30d"`
	AgendamentosAtivos int `json:"agendamentos_ativos"`
	LoginsHoje         int `json:"logins_hoje"`
	Logins7d           int `json:"logins_7d"`
	AtivosHoje         int `json:"ativos_hoje"` // DAU — usuários ativos hoje
	Ativos7d           int `json:"ativos_7d"`   // WAU — ativos nos últimos 7 dias
	Ativos30d          int `json:"ativos_30d"`  // MAU — ativos nos últimos 30 dias
	EventosHoje        int `json:"eventos_hoje"`
}

// ─── Listagem de contas (resumo de cada conta + uso) ──────────────────────────

// ContasPaginaDTO é a página de contas com o total para paginar.
type ContasPaginaDTO struct {
	Itens  []ContaDTO `json:"itens"`
	Total  int        `json:"total"`
	Limite int        `json:"limite"`
	Offset int        `json:"offset"`
}

// ContaDTO é o resumo de uma conta na visão do admin (medir o uso de cada uma).
type ContaDTO struct {
	ID           string     `json:"id"`
	Nome         string     `json:"nome"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	CriadoEm     time.Time  `json:"criado_em"`
	UltimoAcesso *time.Time `json:"ultimo_acesso"` // último evento na auditoria (nil = nunca)
	TotalEventos int        `json:"total_eventos"` // total de ações registradas da conta
	// Contadores de uso — relevantes conforme o papel (o front mostra os que importam):
	Equipes       int `json:"equipes"`       // gestor: equipes que criou
	Colaboradores int `json:"colaboradores"` // gestor: liderados sob ele
	Onebyones     int `json:"onebyones"`     // gestor: 1:1 que conduziu
	Gestores      int `json:"gestores"`      // RH: gestores no tenant dele
}

// ─── Séries temporais (gráfico estilo Google Analytics) ───────────────────────

// SerieAcessosDTO é a série diária de acessos para o gráfico de linha do dashboard.
// Os arrays vêm alinhados por índice com `dias` (um ponto por dia, buracos preenchidos
// com zero), prontos para alimentar qualquer biblioteca de gráficos.
type SerieAcessosDTO struct {
	Dias      []string `json:"dias"`       // rótulos do eixo X (YYYY-MM-DD)
	Logins    []int    `json:"logins"`     // logins por dia
	Ativos    []int    `json:"ativos"`     // usuários ativos (distintos) por dia
	Eventos   []int    `json:"eventos"`    // total de eventos por dia
	Periodo   int      `json:"periodo"`    // janela em dias
	TotalLogs int      `json:"total_logs"` // soma de logins no período
}

// UsoDTO traz as distribuições de uso (o "como" a plataforma é usada).
type UsoDTO struct {
	Periodo      int           `json:"periodo"`
	TopFunc      []FuncDTO     `json:"top_funcionalidades"`
	PorHora      []ContagemDTO `json:"por_hora"`       // 24 baldes (0–23h)
	PorDiaSemana []ContagemDTO `json:"por_dia_semana"` // 7 baldes (Dom–Sáb)
	PorPapel     []ContagemDTO `json:"por_papel"`
}

// FuncDTO é uma funcionalidade no ranking de uso.
type FuncDTO struct {
	Entidade string `json:"entidade"`
	Acao     string `json:"acao"`
	Total    int    `json:"total"`
}

// ContagemDTO é um par rótulo→valor genérico para gráficos de barra/pizza.
type ContagemDTO struct {
	Rotulo string `json:"rotulo"`
	Total  int    `json:"total"`
}

// CrescimentoDTO traz o crescimento de cadastros e de 1:1 ao longo do período.
type CrescimentoDTO struct {
	Periodo int      `json:"periodo"`
	Dias    []string `json:"dias"` // eixo X comum (YYYY-MM-DD)
	// Novos cadastros por dia, por papel (uma série por papel — todas alinhadas a `dias`):
	NovosRH        []int `json:"novos_rh"`
	NovosGestores  []int `json:"novos_gestores"`
	NovosLiderados []int `json:"novos_liderados"`
	NovosTotal     []int `json:"novos_total"`     // soma dos papéis por dia
	AcumuladoTotal []int `json:"acumulado_total"` // contas acumuladas (curva de crescimento)
	// 1:1 realizados por dia (alinhado a `dias`):
	Realizados []int `json:"realizados"`
}

// ─── Saúde da plataforma (indicadores de engajamento/adoção) ──────────────────

// SaudePlataformaDTO traz indicadores derivados (médias e percentuais já calculados)
// + o ranking de gestores mais engajados.
type SaudePlataformaDTO struct {
	Gestores                int                    `json:"gestores"`
	GestoresCom1a1          int                    `json:"gestores_com_1a1"`
	GestoresSem1a1          int                    `json:"gestores_sem_1a1"`
	PctGestoresEngajados    float64                `json:"pct_gestores_engajados"` // % de gestores que já fizeram 1:1
	MediaLideradosPorGestor float64                `json:"media_liderados_por_gestor"`
	LideradosAtivos         int                    `json:"liderados_ativos"`
	LideradosComConta       int                    `json:"liderados_com_conta"`
	PctLideradosVinculados  float64                `json:"pct_liderados_vinculados"` // % de liderados que aceitaram o convite
	ContasComIA             int                    `json:"contas_com_ia"`
	ContasComFoto           int                    `json:"contas_com_foto"`
	Realizados30d           int                    `json:"realizados_30d"`
	TopGestores             []GestorEngajamentoDTO `json:"top_gestores"`
	GeradoEm                time.Time              `json:"gerado_em"`
}

// GestorEngajamentoDTO é um gestor no ranking de engajamento.
type GestorEngajamentoDTO struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Email      string `json:"email"`
	Realizados int    `json:"realizados"`
	Liderados  int    `json:"liderados"`
}
