// Pacote: internal/acompanhamento
// Arquivo: entity.go
// Descrição: Entidade Acompanhamento — um registro de acompanhamento do liderado.
//            Unifica num só lugar os quatro tipos exibidos no painel: SENTIMENTO
//            (humor da semana, 1-5), ENTREGA, FEEDBACK (recebido) e ESTUDO. Cada
//            registro tem data de referência, título e detalhe opcional.
// Autor: OneByOne API
// Criado em: 2026

package acompanhamento

import "time"

// Tipos válidos de acompanhamento (espelhados no binding do DTO).
const (
	TipoSentimento = "SENTIMENTO"
	TipoEntrega    = "ENTREGA"
	TipoFeedback   = "FEEDBACK"
	TipoEstudo     = "ESTUDO"
)

// Acompanhamento mapeia a tabela tb_acompanhamentos.
type Acompanhamento struct {
	ID            string     `db:"id"`
	ColaboradorID string     `db:"colaborador_id"`
	Tipo          string     `db:"tipo"`
	Titulo        string     `db:"titulo"`
	Detalhe       *string    `db:"detalhe"`
	Valor         *int       `db:"valor"` // humor 1-5 no SENTIMENTO; nulo nos demais
	DataRef       time.Time  `db:"data_ref"`
	CriadoEm      time.Time  `db:"criado_em"`
	AlteradoEm    *time.Time `db:"alterado_em"`
	DeletadoEm    *time.Time `db:"deletado_em"`
}
