// Pacote: internal/convite
// Arquivo: entity.go
// Descrição: Define a entidade Convite, que representa o convite enviado a um
//            colaborador (liderado) para ele criar/vincular seu acesso ao sistema.
//            Mapeia a tabela tb_convites.
// Autor: OneByOne API
// Criado em: 2025

package convite

import "time"

// Status possíveis de um convite.
const (
	StatusPendente  = "PENDENTE"
	StatusAceito    = "ACEITO"
	StatusCancelado = "CANCELADO"
)

// Convite liga um colaborador a um link (UUID) + código (contra-senha) de acesso.
type Convite struct {
	// ID é o UUID do convite — também é o token usado no link /convite/{id}
	ID string `db:"id"`
	// ColaboradorID é o UUID do colaborador (liderado) que está sendo convidado
	ColaboradorID string `db:"colaborador_id"`
	// CodigoHash é o hash bcrypt do código (contra-senha) que o liderado deve informar
	CodigoHash string `db:"codigo_hash"`
	// Status indica a situação: PENDENTE, ACEITO ou CANCELADO
	Status string `db:"status"`
	// ExpiraEm é a data/hora em que o convite deixa de ser válido
	ExpiraEm time.Time `db:"expira_em"`
	// CriadoEm é o timestamp de criação do convite
	CriadoEm time.Time `db:"criado_em"`
	// AceitoEm é o timestamp em que o convite foi aceito (nil se ainda pendente)
	AceitoEm *time.Time `db:"aceito_em"`
}
