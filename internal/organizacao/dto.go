// Pacote: internal/organizacao
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de organização,
//            desacoplando o modelo de banco da camada HTTP.
// Autor: OneByOne API
// Criado em: 2025

package organizacao

import "time"

// CriarOrganizacaoDTO contém os dados enviados pelo cliente para criar uma organização
type CriarOrganizacaoDTO struct {
	// Nome é o nome da organização (obrigatório, 2 a 100 caracteres)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
	// TemplateID é o UUID do template padrão (opcional)
	TemplateID *string `json:"template_id" binding:"omitempty"`
}

// AtualizarOrganizacaoDTO contém os campos alteráveis de uma organização.
// Todos os campos são opcionais — apenas os informados serão atualizados.
type AtualizarOrganizacaoDTO struct {
	// Nome é o novo nome da organização (opcional)
	Nome string `json:"nome" binding:"omitempty,min=2,max=100"`
	// TemplateID é o novo UUID do template padrão (opcional; envie null para remover)
	TemplateID *string `json:"template_id" binding:"omitempty"`
}

// OrganizacaoRespostaDTO representa os dados da organização retornados pela API
type OrganizacaoRespostaDTO struct {
	// ID é o identificador único da organização
	ID string `json:"id"`
	// UsuarioID é o UUID do líder proprietário
	UsuarioID string `json:"usuario_id"`
	// TemplateID é o UUID do template padrão (null se não configurado)
	TemplateID *string `json:"template_id"`
	// Nome é o nome da organização
	Nome string `json:"nome"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação (null se nunca alterado)
	AlteradoEm *time.Time `json:"alterado_em"`
	// FotoURL é a URL presignada temporária da foto da organização (null se sem foto; expira em 2h)
	FotoURL *string `json:"foto_url"`
}
