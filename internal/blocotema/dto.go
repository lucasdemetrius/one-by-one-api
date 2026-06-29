// Pacote: internal/blocotema
// Arquivo: dto.go
// Descrição: DTOs de entrada e saída dos blocos de tema.
// Autor: OneByOne API
// Criado em: 2025

package blocotema

// CriarBlocoDTO cria um bloco de texto, link ou marco (a imagem entra por upload).
type CriarBlocoDTO struct {
	// Tema é o título do tema ao qual o bloco pertence (obrigatório)
	Tema string `json:"tema" binding:"required,max=150"`
	// Tipo é TEXTO, LINK ou MARCO (a IMAGEM é criada pela rota de upload)
	Tipo string `json:"tipo" binding:"required,oneof=TEXTO LINK MARCO"`
	// Texto é o conteúdo/rótulo/descrição (opcional conforme o tipo)
	Texto *string `json:"texto"`
	// URL é o endereço do link/curso (para tipo LINK)
	URL *string `json:"url"`
	// DataInicio e DataFim são usadas no tipo MARCO (formato YYYY-MM-DD)
	DataInicio *string `json:"data_inicio"`
	DataFim    *string `json:"data_fim"`
}

// BlocoRespostaDTO é o bloco como devolvido pela API (imagem já como URL presignada).
type BlocoRespostaDTO struct {
	ID         string  `json:"id"`
	Tema       string  `json:"tema"`
	Tipo       string  `json:"tipo"`
	Texto      *string `json:"texto"`
	URL        *string `json:"url"`
	ImagemURL  *string `json:"imagem_url"`
	DataInicio *string `json:"data_inicio"`
	DataFim    *string `json:"data_fim"`
	Ordem      int     `json:"ordem"`
}
