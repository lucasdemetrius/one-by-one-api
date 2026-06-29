// Pacote: internal/pdi
// Arquivo: entity.go
// Descrição: Entidade ItemPDI — um objetivo/ação do Plano de Desenvolvimento
//            Individual de um liderado, com prazo e status de conclusão.
// Autor: OneByOne API
// Criado em: 2026

package pdi

import "time"

// ItemPDI mapeia a tabela tb_pdi_itens.
type ItemPDI struct {
	ID            string     `db:"id"`
	ColaboradorID string     `db:"colaborador_id"`
	Titulo        string     `db:"titulo"`
	Descricao     *string    `db:"descricao"`
	Prazo         *time.Time `db:"prazo"`
	Concluido     bool       `db:"concluido"`
	ConcluidoEm   *time.Time `db:"concluido_em"` // quando foi marcado como feito (p/ evolução)
	CriadoEm      time.Time  `db:"criado_em"`
	AlteradoEm    *time.Time `db:"alterado_em"`
	DeletadoEm    *time.Time `db:"deletado_em"`
}
