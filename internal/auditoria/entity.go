package auditoria

import "time"

// Auditoria representa um evento registrado na tabela tb_auditoria
type Auditoria struct {
	ID         string    `db:"id"`
	UsuarioID  *string   `db:"usuario_id"`
	Acao       string    `db:"acao"`
	Entidade   string    `db:"entidade"`
	EntidadeID *string   `db:"entidade_id"`
	IP         *string   `db:"ip"`
	UserAgent  *string   `db:"user_agent"`
	CriadoEm  time.Time `db:"criado_em"`
}
