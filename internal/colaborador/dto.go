// Pacote: internal/colaborador
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de colaborador,
//            desacoplando o modelo de banco da camada HTTP.
// Autor: OneByOne API
// Criado em: 2025

package colaborador

import "time"

// CriarColaboradorDTO contém os dados enviados pelo cliente para criar um colaborador
type CriarColaboradorDTO struct {
	// OrganizacaoID é o UUID da organização à qual o colaborador pertence (obrigatório)
	OrganizacaoID string `json:"organizacao_id" binding:"required"`
	// EquipeID é o UUID da equipe à qual o colaborador pertence (obrigatório)
	EquipeID string `json:"equipe_id" binding:"required"`
	// Nome é o nome completo do colaborador (obrigatório, 2 a 100 caracteres)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
	// Email é o endereço de e-mail do colaborador (obrigatório)
	Email string `json:"email" binding:"required,email,max=150"`
	// UsuarioID é o UUID da conta de sistema do colaborador (opcional)
	UsuarioID *string `json:"usuario_id" binding:"omitempty"`
	// TemplateID é o UUID do template exclusivo deste colaborador (opcional; máxima prioridade)
	TemplateID *string `json:"template_id" binding:"omitempty"`
	// Whatsapp é o número de WhatsApp com DDD (opcional)
	Whatsapp *string `json:"whatsapp" binding:"omitempty,max=20"`
	// DataNascimento é a data de nascimento no formato YYYY-MM-DD (opcional)
	DataNascimento string `json:"data_nascimento" binding:"omitempty"`
}

// AtualizarColaboradorDTO contém os campos alteráveis de um colaborador.
// Todos os campos são opcionais — apenas os informados serão atualizados.
type AtualizarColaboradorDTO struct {
	// Nome é o novo nome completo do colaborador (opcional)
	Nome string `json:"nome" binding:"omitempty,min=2,max=100"`
	// Email é o novo endereço de e-mail (opcional)
	Email string `json:"email" binding:"omitempty,email,max=150"`
	// EquipeID é o UUID da nova equipe (opcional — para transferência)
	EquipeID string `json:"equipe_id" binding:"omitempty"`
	// UsuarioID é o UUID da conta de sistema (opcional; envie null para desvincular)
	UsuarioID *string `json:"usuario_id" binding:"omitempty"`
	// TemplateID é o novo UUID do template exclusivo (opcional; envie null para remover)
	TemplateID *string `json:"template_id" binding:"omitempty"`
	// Whatsapp é o novo número de WhatsApp (opcional)
	Whatsapp *string `json:"whatsapp" binding:"omitempty,max=20"`
	// DataNascimento é a nova data de nascimento no formato YYYY-MM-DD (opcional)
	DataNascimento string `json:"data_nascimento" binding:"omitempty"`
}

// DesligarColaboradorDTO contém a data opcional de desligamento (default: hoje)
type DesligarColaboradorDTO struct {
	// DataDesligamento no formato YYYY-MM-DD (opcional — se vazio, usa a data atual)
	DataDesligamento string `json:"data_desligamento" binding:"omitempty"`
}

// ColaboradorRespostaDTO representa os dados do colaborador retornados pela API
type ColaboradorRespostaDTO struct {
	// ID é o identificador único do colaborador
	ID string `json:"id"`
	// UsuarioID é o UUID da conta de sistema (null se não vinculado)
	UsuarioID *string `json:"usuario_id"`
	// OrganizacaoID é o UUID da organização
	OrganizacaoID string `json:"organizacao_id"`
	// EquipeID é o UUID da equipe
	EquipeID string `json:"equipe_id"`
	// TemplateID é o UUID do template exclusivo (null se não configurado)
	TemplateID *string `json:"template_id"`
	// Nome é o nome completo do colaborador
	Nome string `json:"nome"`
	// Email é o endereço de e-mail
	Email string `json:"email"`
	// Whatsapp é o número de WhatsApp (null se não informado)
	Whatsapp *string `json:"whatsapp"`
	// DataNascimento é a data de nascimento (null se não informada)
	DataNascimento *time.Time `json:"data_nascimento"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação
	AlteradoEm *time.Time `json:"alterado_em"`
	// FotoURL é a URL presignada temporária da foto do colaborador (null se sem foto; expira em 2h)
	FotoURL *string `json:"foto_url"`
	// DesligadoEm é a data de desligamento (null = ativo)
	DesligadoEm *time.Time `json:"desligado_em"`
	// Ativo é true quando o colaborador NÃO está desligado (conveniência para o frontend)
	Ativo bool `json:"ativo"`
}

// ── Importação em lote (CSV) ─────────────────────────────────────────────────

// ItemImportacaoDTO é uma linha do CSV (nome + e-mail). Sem binding rígido: cada
// linha é validada individualmente para que uma linha ruim não derrube o lote todo.
type ItemImportacaoDTO struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

// ImportarColaboradoresDTO é o corpo do import em lote: a equipe-alvo + as linhas.
type ImportarColaboradoresDTO struct {
	OrganizacaoID string              `json:"organizacao_id" binding:"required"`
	EquipeID      string              `json:"equipe_id" binding:"required"`
	Itens         []ItemImportacaoDTO `json:"itens" binding:"required,min=1,max=500"`
}

// ErroImportacaoDTO descreve por que uma linha não foi importada.
type ErroImportacaoDTO struct {
	Linha  int    `json:"linha"`
	Nome   string `json:"nome"`
	Email  string `json:"email"`
	Motivo string `json:"motivo"`
}

// ResultadoImportacaoDTO resume o lote: quem foi criado e quais linhas falharam.
type ResultadoImportacaoDTO struct {
	Criados []ColaboradorRespostaDTO `json:"criados"`
	Erros   []ErroImportacaoDTO      `json:"erros"`
}
