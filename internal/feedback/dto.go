// Pacote: internal/feedback
// Arquivo: dto.go
// Descrição: Contratos HTTP do módulo de feedback. A escrita (CriarFeedbackDTO) é simples
//            de propósito — qualquer usuário logado reage em um clique. Os DTOs do painel
//            de ADMIN vêm prontos para gráficos e leitura dos comentários.
// Autor: OneByOne API
// Criado em: 2026

package feedback

import "time"

// ─── Escrita (qualquer usuário logado) ────────────────────────────────────────

// CriarFeedbackDTO é o corpo de POST /feedback. O usuario_id NUNCA vem aqui — é do JWT.
type CriarFeedbackDTO struct {
	// Reacao é obrigatória: CURTI, NAO_CURTI ou IRRITADO.
	Reacao string `json:"reacao" binding:"required,oneof=CURTI NAO_CURTI IRRITADO"`
	// Contexto é a tela/recurso onde reagiu (ex.: "1a1", "pdi", "ajuda"); opcional.
	Contexto string `json:"contexto" binding:"omitempty,max=60"`
	// Comentario é um texto livre opcional (feedback qualitativo).
	Comentario string `json:"comentario" binding:"omitempty,max=500"`
	// Pagina é a rota/URL onde reagiu; opcional.
	Pagina string `json:"pagina" binding:"omitempty,max=255"`
}

// FeedbackRespostaDTO é o que o cliente recebe ao registrar um feedback.
type FeedbackRespostaDTO struct {
	ID       string    `json:"id"`
	Reacao   string    `json:"reacao"`
	Contexto *string   `json:"contexto"`
	CriadoEm time.Time `json:"criado_em"`
}

// ─── Painel do ADMIN (só leitura agregada) ────────────────────────────────────

// PainelFeedbackDTO é o resumo de feedback para o dashboard de gestão.
type PainelFeedbackDTO struct {
	Periodo int `json:"periodo"` // janela em dias
	// Totais por reação no período.
	Total    int `json:"total"`
	Curti    int `json:"curti"`
	NaoCurti int `json:"nao_curti"`
	Irritado int `json:"irritado"`
	// IndiceSatisfacao = curti / total (%), com uma casa. 0 se não houver feedback.
	IndiceSatisfacao float64 `json:"indice_satisfacao"`
	// Série temporal alinhada por índice com `dias` (buracos preenchidos com zero).
	Dias          []string `json:"dias"`
	SerieCurti    []int    `json:"serie_curti"`
	SerieNaoCurti []int    `json:"serie_nao_curti"`
	SerieIrritado []int    `json:"serie_irritado"`
	// Quebra por contexto/tela (onde as pessoas mais reagem e como).
	PorContexto []ContextoFeedbackDTO `json:"por_contexto"`
	// Comentários recentes (feedback qualitativo) com autor.
	Recentes []ComentarioFeedbackDTO `json:"recentes"`
	GeradoEm time.Time               `json:"gerado_em"`
}

// ContextoFeedbackDTO resume as reações de uma tela/recurso.
type ContextoFeedbackDTO struct {
	Contexto string `json:"contexto"`
	Curti    int    `json:"curti"`
	NaoCurti int    `json:"nao_curti"`
	Irritado int    `json:"irritado"`
	Total    int    `json:"total"`
}

// ComentarioFeedbackDTO é um feedback recente com comentário, já com o autor.
type ComentarioFeedbackDTO struct {
	Reacao     string    `json:"reacao"`
	Contexto   *string   `json:"contexto"`
	Comentario string    `json:"comentario"`
	AutorNome  string    `json:"autor_nome"`
	AutorPapel string    `json:"autor_papel"`
	CriadoEm   time.Time `json:"criado_em"`
}
