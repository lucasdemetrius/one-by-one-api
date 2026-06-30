// Pacote: internal/feedback
// Arquivo: entity.go
// Descrição: Entidade Feedback — espelho da tabela tb_feedbacks. Cada registro é uma
//            reação de um usuário (curti / não curti / irritado), com contexto e comentário
//            opcionais. É um log append-only (não é toggle de "like").
// Autor: OneByOne API
// Criado em: 2026

package feedback

import "time"

// Reações suportadas (valores gravados em tb_feedbacks.reacao).
const (
	ReacaoCurti    = "CURTI"
	ReacaoNaoCurti = "NAO_CURTI"
	ReacaoIrritado = "IRRITADO"
)

// ReacaoValida confere se a reação informada é uma das suportadas.
func ReacaoValida(r string) bool {
	switch r {
	case ReacaoCurti, ReacaoNaoCurti, ReacaoIrritado:
		return true
	}
	return false
}

// Feedback representa uma reação registrada na tabela tb_feedbacks.
type Feedback struct {
	// ID é o identificador único (UUID v4)
	ID string `db:"id"`
	// UsuarioID é quem deu o feedback (vem do JWT, nunca do corpo)
	UsuarioID string `db:"usuario_id"`
	// Reacao é CURTI, NAO_CURTI ou IRRITADO
	Reacao string `db:"reacao"`
	// Contexto é a tela/recurso onde reagiu (ex.: "1a1", "pdi", "ajuda"); opcional
	Contexto *string `db:"contexto"`
	// Comentario é um texto livre opcional (feedback qualitativo)
	Comentario *string `db:"comentario"`
	// Pagina é a rota/URL onde reagiu; opcional
	Pagina *string `db:"pagina"`
	// CriadoEm é o timestamp de criação
	CriadoEm time.Time `db:"criado_em"`
}
