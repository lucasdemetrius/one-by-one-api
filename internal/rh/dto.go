// Pacote: internal/rh
// Arquivo: dto.go
// Descrição: Contratos de entrada/saída do módulo de RH (Recursos Humanos), o topo
//            do tenant. O RH cadastra gestores e tem visão consolidada deles.
// Autor: OneByOne API
// Criado em: 2026

package rh

import "time"

// CriarGestorDTO é o corpo do POST /rh/gestores. NÃO há rh_id aqui de propósito: o
// vínculo do gestor com o tenant é derivado do JWT do RH autenticado, nunca do corpo.
type CriarGestorDTO struct {
	// Nome é o nome do gestor (obrigatório, 2 a 100 caracteres)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
	// Email é o e-mail de login do gestor (obrigatório, único no sistema)
	Email string `json:"email" binding:"required,email,max=150"`
	// Password é a senha inicial do gestor (complexidade validada por pkg/senha)
	Password string `json:"password" binding:"required,max=100"`
	// Empresa é o nome da organização (empresa) que o RH monta para o gestor já usar.
	// Opcional — se vazio, vira "Minha empresa". O gestor NÃO cria empresa; usa esta.
	Empresa string `json:"empresa" binding:"omitempty,max=100"`
}

// GestorResumoDTO é um item da lista de gestores do RH, já com os KPIs de produtividade
// (saúde do 1:1) embutidos para o dashboard. KPIs vêm zerados se o gestor ainda não tem
// agenda montada.
type GestorResumoDTO struct {
	// ID é o UUID da conta do gestor
	ID string `json:"id"`
	// Nome do gestor
	Nome string `json:"nome"`
	// Email do gestor
	Email string `json:"email"`
	// CriadoEm é quando a conta do gestor foi criada
	CriadoEm time.Time `json:"criado_em"`
	// PercentualEmDia é a % da agenda de 1:1 em dia (0 a 100)
	PercentualEmDia int `json:"percentual_em_dia"`
	// TotalAgendados é o total de 1:1 agendados (ativos)
	TotalAgendados int `json:"total_agendados"`
	// Atrasados é quantos 1:1 estão vencidos
	Atrasados int `json:"atrasados"`
	// RealizadosUlt30 é quantos 1:1 foram realizados nos últimos 30 dias
	RealizadosUlt30 int `json:"realizados_ult_30"`
	// StreakSemanas é a sequência de semanas consecutivas com pelo menos um 1:1
	StreakSemanas int `json:"streak_semanas"`
}

// AgendaItemDTO é um 1:1 agendado na visão consolidada do RH (de todos os gestores do
// tenant), já com o gestor e a equipe para exibir e filtrar.
type AgendaItemDTO struct {
	ID            string `json:"id"`
	GestorID      string `json:"gestor_id"`
	GestorNome    string `json:"gestor_nome"`
	ColaboradorID string `json:"colaborador_id"`
	LideradoNome  string `json:"liderado_nome"`
	EquipeID      string `json:"equipe_id"`
	EquipeNome    string `json:"equipe_nome"`
	DataHora      string `json:"data_hora"` // "YYYY-MM-DDTHH:MM"
	Recorrencia   string `json:"recorrencia"`
	RepeteAte     string `json:"repete_ate"` // "YYYY-MM-DD" ou ""
}

// MatrixItemDTO é um liderado na visão 9-box consolidada do RH, com gestor/equipe e a
// classificação (desempenho × potencial); vazio se ainda não classificado.
type MatrixItemDTO struct {
	ColaboradorID string `json:"colaborador_id"`
	LideradoNome  string `json:"liderado_nome"`
	GestorID      string `json:"gestor_id"`
	GestorNome    string `json:"gestor_nome"`
	EquipeID      string `json:"equipe_id"`
	EquipeNome    string `json:"equipe_nome"`
	Desempenho    string `json:"desempenho"` // BAIXO/MEDIO/ALTO ou ""
	Potencial     string `json:"potencial"`  // BAIXO/MEDIO/ALTO ou ""
}

// LideradoRiscoDTO aponta um liderado que precisa de atenção e o porquê (humor caindo,
// humor baixo, PDI atrasado). É o que dá nome e motivo ao "com quem o RH deve sentar".
type LideradoRiscoDTO struct {
	ColaboradorID string `json:"colaborador_id"` // para o RH abrir o dossiê direto
	Nome          string `json:"nome"`
	Motivo        string `json:"motivo"`
}

// GestorEvolucaoDTO resume a EVOLUÇÃO dos liderados de um gestor — não a quantidade de 1:1.
// O foco é qualidade: tendência de humor do time, quem está em risco, progresso de PDI e
// lacunas de 9-box. A lista é ordenada por necessidade de atenção (mais risco primeiro).
type GestorEvolucaoDTO struct {
	GestorID         string             `json:"gestor_id"`
	GestorNome       string             `json:"gestor_nome"`
	TotalLiderados   int                `json:"total_liderados"`
	ComHumor         int                `json:"com_humor"`       // liderados com registro de humor
	HumorMedia       float64            `json:"humor_media"`     // 0..5 (0 = sem dados)
	HumorTendencia   float64            `json:"humor_tendencia"` // recente − anterior (>0 sobe, <0 cai)
	LideradosEmRisco int                `json:"liderados_em_risco"`
	PdiTotal         int                `json:"pdi_total"`
	PdiConcluidos    int                `json:"pdi_concluidos"`
	PdiAtrasados     int                `json:"pdi_atrasados"`
	SemClassificacao int                `json:"sem_classificacao"`
	Riscos           []LideradoRiscoDTO `json:"riscos"` // quem precisa de atenção (nome + motivo)
}
