// Pacote: internal/onebyone
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de one-on-one,
//            desacoplando o modelo de banco da camada HTTP.
// Autor: OneByOne API
// Criado em: 2025

package onebyone

import "time"

// CriarOneByOneDTO contém os dados enviados pelo cliente para agendar uma reunião
type CriarOneByOneDTO struct {
	// OrganizacaoID é o UUID da organização contexto da reunião (obrigatório)
	OrganizacaoID string `json:"organizacao_id" binding:"required"`
	// EquipeID é o UUID da equipe contexto da reunião (obrigatório)
	EquipeID string `json:"equipe_id" binding:"required"`
	// ColaborID é o UUID do colaborador que participará da reunião (obrigatório)
	ColaborID string `json:"colabor_id" binding:"required"`
	// Recorrencia define a frequência: NENHUMA, MENSAL ou QUINZENAL (padrão: NENHUMA)
	Recorrencia string `json:"recorrencia" binding:"omitempty,oneof=NENHUMA MENSAL QUINZENAL"`
	// DataAgendada é a data planejada para a reunião no formato YYYY-MM-DD (obrigatório)
	DataAgendada string `json:"data_agendada" binding:"required"`
}

// EncerrarOneByOneDTO registra um 1:1 como REALIZADO no "livro-razão" (tb_onebyone).
// O fluxo ao vivo é por colaborador (não tem reunião pré-criada), então o encerrar
// CRIA a linha já realizada. O resumo do encontro é salvo à parte como bloco de
// histórico (módulo blocotema), por isso aqui só precisamos do colaborador.
type EncerrarOneByOneDTO struct {
	// ColaborID é o UUID do colaborador cujo 1:1 acabou de ser realizado (obrigatório)
	ColaborID string `json:"colabor_id" binding:"required"`
}

// AtualizarOneByOneDTO contém os campos alteráveis de uma reunião agendada.
// Todos os campos são opcionais — apenas os informados serão atualizados.
type AtualizarOneByOneDTO struct {
	// Status é o novo status da reunião: AGENDADO, REALIZADO ou PENDENTE (opcional)
	Status string `json:"status" binding:"omitempty,oneof=AGENDADO REALIZADO PENDENTE"`
	// Recorrencia é a nova recorrência (opcional)
	Recorrencia string `json:"recorrencia" binding:"omitempty,oneof=NENHUMA MENSAL QUINZENAL"`
	// DataAgendada é a nova data da reunião no formato YYYY-MM-DD (opcional)
	DataAgendada string `json:"data_agendada" binding:"omitempty"`
}

// OneByOneRespostaDTO representa os dados da reunião retornados pela API
type OneByOneRespostaDTO struct {
	// ID é o identificador único da reunião
	ID string `json:"id"`
	// UsuarioID é o UUID do líder que agendou
	UsuarioID string `json:"usuario_id"`
	// OrganizacaoID é o UUID da organização
	OrganizacaoID string `json:"organizacao_id"`
	// EquipeID é o UUID da equipe
	EquipeID string `json:"equipe_id"`
	// ColaborID é o UUID do colaborador
	ColaborID string `json:"colabor_id"`
	// Recorrencia é a frequência da reunião
	Recorrencia string `json:"recorrencia"`
	// Status é o estado atual da reunião
	Status string `json:"status"`
	// RealizadoEm é quando a reunião foi realizada (nil se ainda não realizada)
	RealizadoEm *time.Time `json:"realizado_em"`
	// DataAgendada é a data planejada da reunião
	DataAgendada time.Time `json:"data_agendada"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação
	AlteradoEm *time.Time `json:"alterado_em"`
}
