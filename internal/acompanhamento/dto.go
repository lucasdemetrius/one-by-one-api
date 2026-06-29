// Pacote: internal/acompanhamento
// Arquivo: dto.go
// Descrição: Contratos HTTP do módulo de acompanhamento (entrada/saída da API).
// Autor: OneByOne API
// Criado em: 2026

package acompanhamento

import "time"

// AcompanhamentoRespostaDTO é o registro devolvido pela API.
type AcompanhamentoRespostaDTO struct {
	ID            string    `json:"id"`
	ColaboradorID string    `json:"colaborador_id"`
	Tipo          string    `json:"tipo"`
	Titulo        string    `json:"titulo"`
	Detalhe       *string   `json:"detalhe"`
	Valor         *int      `json:"valor"`
	DataRef       string    `json:"data_ref"` // "YYYY-MM-DD"
	CriadoEm      time.Time `json:"criado_em"`
}

// CriarAcompanhamentoDTO cria um registro. Para SENTIMENTO o `valor` (1-5) é
// obrigatório (título é opcional); nos demais tipos o `titulo` é obrigatório.
type CriarAcompanhamentoDTO struct {
	Tipo    string `json:"tipo" binding:"required,oneof=SENTIMENTO ENTREGA FEEDBACK ESTUDO"`
	Titulo  string `json:"titulo" binding:"omitempty,max=255"`
	Detalhe string `json:"detalhe" binding:"omitempty"`
	Valor   *int   `json:"valor" binding:"omitempty,min=1,max=5"`
	DataRef string `json:"data_ref" binding:"omitempty"` // "YYYY-MM-DD" (padrão: hoje)
}

// AtualizarAcompanhamentoDTO atualiza campos (todos opcionais).
type AtualizarAcompanhamentoDTO struct {
	Titulo  string `json:"titulo" binding:"omitempty,max=255"`
	Detalhe *string `json:"detalhe" binding:"omitempty"`
	Valor   *int   `json:"valor" binding:"omitempty,min=1,max=5"`
	DataRef string `json:"data_ref" binding:"omitempty"`
}
