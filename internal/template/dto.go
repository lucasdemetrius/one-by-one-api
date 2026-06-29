// Pacote: internal/template
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de template,
//            desacoplando o modelo de banco da camada HTTP.
// Autor: OneByOne API
// Criado em: 2025

package template

import "time"

// CriarTemplateDTO contém os dados enviados pelo cliente para criar um template
type CriarTemplateDTO struct {
	// Nome é o nome descritivo do template (obrigatório, 2 a 100 caracteres)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
}

// AtualizarTemplateDTO contém os campos alteráveis de um template
type AtualizarTemplateDTO struct {
	// Nome é o novo nome do template (obrigatório na atualização)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
}

// TemplateRespostaDTO representa os dados do template retornados pela API
type TemplateRespostaDTO struct {
	// ID é o identificador único do template
	ID string `json:"id"`
	// UsuarioID é o UUID do líder proprietário do template
	UsuarioID string `json:"usuario_id"`
	// Nome é o nome do template
	Nome string `json:"nome"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação (null se nunca alterado)
	AlteradoEm *time.Time `json:"alterado_em"`
}
