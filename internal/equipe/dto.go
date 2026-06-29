// Pacote: internal/equipe
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de equipe,
//            desacoplando o modelo de banco da camada HTTP.
// Autor: OneByOne API
// Criado em: 2025

package equipe

import "time"

// CriarEquipeDTO contém os dados enviados pelo cliente para criar uma equipe
type CriarEquipeDTO struct {
	// OrganizacaoID é o UUID da organização à qual a equipe pertencerá (obrigatório)
	OrganizacaoID string `json:"organizacao_id" binding:"required"`
	// Nome é o nome da equipe (obrigatório, 2 a 100 caracteres)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
	// TemplateID é o UUID do template padrão da equipe (opcional)
	TemplateID *string `json:"template_id" binding:"omitempty"`
}

// AtualizarEquipeDTO contém os campos alteráveis de uma equipe.
// Todos os campos são opcionais — apenas os informados serão atualizados.
type AtualizarEquipeDTO struct {
	// Nome é o novo nome da equipe (opcional)
	Nome string `json:"nome" binding:"omitempty,min=2,max=100"`
	// TemplateID é o novo UUID do template padrão (opcional; envie null para remover)
	TemplateID *string `json:"template_id" binding:"omitempty"`
}

// EquipeRespostaDTO representa os dados da equipe retornados pela API
type EquipeRespostaDTO struct {
	// ID é o identificador único da equipe
	ID string `json:"id"`
	// UsuarioID é o UUID do líder responsável pela equipe
	UsuarioID string `json:"usuario_id"`
	// OrganizacaoID é o UUID da organização à qual a equipe pertence
	OrganizacaoID string `json:"organizacao_id"`
	// TemplateID é o UUID do template padrão (null se não configurado)
	TemplateID *string `json:"template_id"`
	// Nome é o nome da equipe
	Nome string `json:"nome"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação (null se nunca alterado)
	AlteradoEm *time.Time `json:"alterado_em"`
	// FotoURL é a URL presignada temporária da foto da equipe (null se sem foto; expira em 2h)
	FotoURL *string `json:"foto_url"`
}
