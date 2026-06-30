// Pacote: internal/feedback
// Arquivo: mapper.go
// Descrição: Conversão entre a entidade Feedback e os DTOs de saída.
// Autor: OneByOne API
// Criado em: 2026

package feedback

// ParaRespostaDTO converte a entidade no DTO devolvido ao cliente após registrar.
func ParaRespostaDTO(f Feedback) FeedbackRespostaDTO {
	return FeedbackRespostaDTO{
		ID:       f.ID,
		Reacao:   f.Reacao,
		Contexto: f.Contexto,
		CriadoEm: f.CriadoEm,
	}
}
