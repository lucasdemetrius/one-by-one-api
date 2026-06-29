// Pacote: internal/pdi
// Arquivo: dto.go
// Descrição: Contratos HTTP do módulo de PDI.
// Autor: OneByOne API
// Criado em: 2026

package pdi

import "time"

// ItemPDIRespostaDTO é o item de PDI devolvido pela API.
type ItemPDIRespostaDTO struct {
	ID            string    `json:"id"`
	ColaboradorID string    `json:"colaborador_id"`
	Titulo        string    `json:"titulo"`
	Descricao     *string   `json:"descricao"`
	Prazo         *string   `json:"prazo"` // "YYYY-MM-DD" ou null
	Concluido     bool      `json:"concluido"`
	ConcluidoEm   *string   `json:"concluido_em"` // "YYYY-MM-DD" ou null (quando foi feito)
	CriadoEm      time.Time `json:"criado_em"`
}

// CriarItemPDIDTO cria um objetivo/ação.
type CriarItemPDIDTO struct {
	Titulo    string `json:"titulo" binding:"required,min=2,max=255"`
	Descricao string `json:"descricao" binding:"omitempty"`
	Prazo     string `json:"prazo" binding:"omitempty"` // "YYYY-MM-DD"
}

// AtualizarItemPDIDTO atualiza campos do item (todos opcionais).
type AtualizarItemPDIDTO struct {
	Titulo    string `json:"titulo" binding:"omitempty,min=2,max=255"`
	Prazo     string `json:"prazo" binding:"omitempty"`
	Concluido *bool  `json:"concluido" binding:"omitempty"`
}
