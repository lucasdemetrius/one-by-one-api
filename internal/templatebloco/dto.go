// Pacote: internal/templatebloco
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de bloco de template,
//            desacoplando o modelo de banco da camada HTTP.
// Autor: OneByOne API
// Criado em: 2025

package templatebloco

import "time"

// CriarTemplateBlocoDTO contém os dados enviados pelo cliente para criar um bloco
type CriarTemplateBlocoDTO struct {
	// TemplateID é o UUID do template ao qual este bloco pertencerá (obrigatório)
	TemplateID string `json:"template_id" binding:"required"`
	// Tipo é o tipo do bloco: TEXT, IMAGE, LIST ou HIGHLIGHT (obrigatório)
	Tipo string `json:"tipo" binding:"required,oneof=TEXT IMAGE LIST HIGHLIGHT"`
	// Posicao é a ordem de exibição do bloco no formulário (obrigatório, >= 0)
	Posicao int `json:"posicao" binding:"min=0"`
	// Rotulo é o label exibido ao usuário para este campo (obrigatório, max 150 chars)
	Rotulo string `json:"rotulo" binding:"required,min=1,max=150"`
}

// AtualizarTemplateBlocoDTO contém os campos alteráveis de um bloco.
// Todos os campos são opcionais — apenas os informados serão atualizados.
type AtualizarTemplateBlocoDTO struct {
	// Tipo é o novo tipo do bloco (opcional): TEXT, IMAGE, LIST ou HIGHLIGHT
	Tipo string `json:"tipo" binding:"omitempty,oneof=TEXT IMAGE LIST HIGHLIGHT"`
	// Posicao é a nova ordem de exibição (opcional)
	Posicao *int `json:"posicao" binding:"omitempty,min=0"`
	// Rotulo é o novo label do campo (opcional)
	Rotulo string `json:"rotulo" binding:"omitempty,min=1,max=150"`
}

// TemplateBlocoRespostaDTO representa os dados do bloco retornados pela API
type TemplateBlocoRespostaDTO struct {
	// ID é o identificador único do bloco
	ID string `json:"id"`
	// TemplateID é o UUID do template ao qual o bloco pertence
	TemplateID string `json:"template_id"`
	// Tipo é o tipo do bloco: TEXT, IMAGE, LIST ou HIGHLIGHT
	Tipo string `json:"tipo"`
	// Posicao é a ordem de exibição do bloco
	Posicao int `json:"posicao"`
	// Rotulo é o label exibido ao usuário
	Rotulo string `json:"rotulo"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação (null se nunca alterado)
	AlteradoEm *time.Time `json:"alterado_em"`
}
