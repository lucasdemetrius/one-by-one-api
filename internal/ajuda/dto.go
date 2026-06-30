// Pacote: internal/ajuda
// Arquivo: dto.go
// Descrição: Contratos HTTP da Central de Ajuda.
// Autor: OneByOne API
// Criado em: 2026

package ajuda

// TopicosDTO é a resposta da listagem de tópicos (já filtrada pelo papel do usuário).
type TopicosDTO struct {
	Itens []Topico `json:"itens"`
}

// TourDTO é a resposta do tour de boas-vindas (etapas na ordem).
type TourDTO struct {
	Passos []PassoTour `json:"passos"`
}

// PerguntarDTO é a pergunta livre do usuário ao assistente de IA.
type PerguntarDTO struct {
	// Pergunta é o texto do usuário. Limite de tamanho protege o custo da IA de plataforma.
	Pergunta string `json:"pergunta" binding:"required,max=1000"`
}

// RespostaIADTO é a resposta do assistente de IA.
type RespostaIADTO struct {
	// Resposta é o texto gerado (ou a mensagem amigável quando a IA não está disponível).
	Resposta string `json:"resposta"`
	// Fonte indica a origem: "plataforma" (chave da plataforma), "byok" (chave do gestor)
	// ou "indisponivel" (sem IA configurada — caímos no conteúdo curado).
	Fonte string `json:"fonte"`
	// IADisponivel facilita o front decidir se mostra o chat de IA ou só os tópicos.
	IADisponivel bool `json:"ia_disponivel"`
}
