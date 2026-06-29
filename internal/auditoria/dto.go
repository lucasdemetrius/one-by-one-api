package auditoria

import "time"

// EventoDTO é o payload enviado pelo frontend para registrar eventos de UI
type EventoDTO struct {
	// Acao descreve o que aconteceu (ex: VISUALIZAR, CLICAR, NAVEGAR)
	Acao string `json:"acao" binding:"required,max=50"`
	// Entidade identifica o contexto (ex: tela_organizacoes, btn_criar_equipe)
	Entidade string `json:"entidade" binding:"required,max=100"`
	// EntidadeID é o UUID do recurso relacionado (opcional)
	EntidadeID *string `json:"entidade_id" binding:"omitempty"`
}

// AuditoriaRespostaDTO representa um registro de auditoria retornado pela API
type AuditoriaRespostaDTO struct {
	ID         string    `json:"id"`
	UsuarioID  *string   `json:"usuario_id"`
	Acao       string    `json:"acao"`
	Entidade   string    `json:"entidade"`
	EntidadeID *string   `json:"entidade_id"`
	IP         *string   `json:"ip"`
	UserAgent  *string   `json:"user_agent"`
	CriadoEm  time.Time `json:"criado_em"`
}
