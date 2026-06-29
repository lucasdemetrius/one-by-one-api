// Pacote: internal/classificacao
// Arquivo: entity.go
// Descrição: Entidade Classificacao — posição do liderado na matriz 9-box
//            (desempenho × potencial). Mapeia a tabela tb_classificacoes.
// Autor: OneByOne API
// Criado em: 2025

package classificacao

import "time"

// Níveis de cada eixo da 9-box.
const (
	NivelBaixo = "BAIXO"
	NivelMedio = "MEDIO"
	NivelAlto  = "ALTO"
)

// Classificacao é o ponto de um liderado na matriz 9-box.
type Classificacao struct {
	ColaboradorID string    `db:"colaborador_id"`
	Desempenho    string    `db:"desempenho"`
	Potencial     string    `db:"potencial"`
	AtualizadoEm  time.Time `db:"atualizado_em"`
}
