// Pacote: internal/tabuleiro
// Arquivo: dto.go
// Descrição: Contratos HTTP do módulo de tabuleiro. O `estado` é repassado como
//            JSON bruto (json.RawMessage) — o backend só persiste/devolve, sem
//            precisar conhecer a estrutura interna da pauta.
// Autor: OneByOne API
// Criado em: 2026

package tabuleiro

import "encoding/json"

// SalvarTabuleiroDTO recebe o estado completo do tabuleiro (JSON).
type SalvarTabuleiroDTO struct {
	Estado json.RawMessage `json:"estado" binding:"required"`
}

// TabuleiroRespostaDTO devolve o estado salvo (ou null se nunca houve um).
type TabuleiroRespostaDTO struct {
	Estado json.RawMessage `json:"estado"`
}
