// Pacote: internal/saude1a1
// Arquivo: dto.go
// Descrição: Contrato HTTP do painel "Saúde do 1:1" — números que o card do /painel
//            mostra ao gestor (motivacional, não punitivo).
// Autor: OneByOne API
// Criado em: 2026

package saude1a1

// SaudeRespostaDTO é o resumo de cadência de 1:1 do gestor.
type SaudeRespostaDTO struct {
	// PercentualEmDia: % dos 1:1 agendados que NÃO estão atrasados (0–100). 100 se não há agenda.
	PercentualEmDia int `json:"percentual_em_dia"`
	// TotalAgendados: agendamentos ativos (cadência esperada).
	TotalAgendados int `json:"total_agendados"`
	// Atrasados: agendamentos ativos cuja ocorrência já venceu.
	Atrasados int `json:"atrasados"`
	// RealizadosUlt30: 1:1 realizados nos últimos 30 dias.
	RealizadosUlt30 int `json:"realizados_ult_30"`
	// StreakSemanas: semanas consecutivas com pelo menos um 1:1 realizado 🔥.
	StreakSemanas int `json:"streak_semanas"`
}
