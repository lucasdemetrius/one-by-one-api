// Pacote: internal/classificacao
// Arquivo: dto.go
// Descrição: DTOs de entrada e saída da classificação 9-box.
// Autor: OneByOne API
// Criado em: 2025

package classificacao

// DefinirClassificacaoDTO posiciona um liderado na 9-box.
type DefinirClassificacaoDTO struct {
	Desempenho string `json:"desempenho" binding:"required,oneof=BAIXO MEDIO ALTO"`
	Potencial  string `json:"potencial" binding:"required,oneof=BAIXO MEDIO ALTO"`
}

// ClassificacaoRespostaDTO é a classificação como devolvida pela API.
type ClassificacaoRespostaDTO struct {
	ColaboradorID string `json:"colaborador_id"`
	Desempenho    string `json:"desempenho"`
	Potencial     string `json:"potencial"`
}
