// Pacote: internal/agendamento
// Arquivo: dto.go
// Descrição: DTOs de entrada e saída dos agendamentos.
// Autor: OneByOne API
// Criado em: 2025

package agendamento

// CriarAgendamentoDTO agenda um 1:1 com um liderado.
type CriarAgendamentoDTO struct {
	ColaboradorID string `json:"colaborador_id" binding:"required"`
	// DataHora no formato "YYYY-MM-DDTHH:MM" (do <input type=datetime-local>).
	DataHora string `json:"data_hora" binding:"required"`
	// Recorrencia opcional (padrão NENHUMA).
	Recorrencia string `json:"recorrencia" binding:"omitempty,oneof=NENHUMA SEMANAL QUINZENAL MENSAL BIMESTRAL TRIMESTRAL SEMESTRAL"`
	// RepeteAte (opcional, "YYYY-MM-DD") = data final da recorrência. Vazio = para sempre.
	// O app calcula esta data quando o gestor escolhe "repetir N vezes".
	RepeteAte string `json:"repete_ate" binding:"omitempty"`
}

// AgendamentoRespostaDTO é o agendamento como devolvido pela API.
type AgendamentoRespostaDTO struct {
	ID            string `json:"id"`
	ColaboradorID string `json:"colaborador_id"`
	LideradoNome  string `json:"liderado_nome"`
	DataHora      string `json:"data_hora"` // "YYYY-MM-DDTHH:MM" (local)
	Recorrencia   string `json:"recorrencia"`
	// RepeteAte ("YYYY-MM-DD") = fim da recorrência; vazio = para sempre. O app usa
	// para parar de projetar ocorrências além dessa data.
	RepeteAte string `json:"repete_ate"`
}
