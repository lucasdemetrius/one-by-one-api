// Pacote: internal/tabuleiro
// Arquivo: entity.go
// Descrição: Entidade Tabuleiro — guarda o estado da pauta do 1:1 de um liderado
//            (banco/pauta/conversado) como JSON, para sobreviver ao recarregar.
//            Um tabuleiro por liderado (colaborador_id é a chave).
// Autor: OneByOne API
// Criado em: 2026

package tabuleiro

import "time"

// Tabuleiro mapeia a tabela tb_tabuleiros (um por liderado).
type Tabuleiro struct {
	ColaboradorID string     `db:"colaborador_id"`
	Estado        string     `db:"estado"` // JSON do tabuleiro (colunas + temas)
	CriadoEm      time.Time  `db:"criado_em"`
	AlteradoEm    *time.Time `db:"alterado_em"`
}
