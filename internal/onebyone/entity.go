// Pacote: internal/onebyone
// Arquivo: entity.go
// Descrição: Define a entidade OneByOne que representa uma reunião agendada
//            entre um líder e um colaborador. Mapeia a tabela tb_onebyone.
// Autor: OneByOne API
// Criado em: 2025

package onebyone

import "time"

// OneByOne representa uma reunião one-on-one agendada entre líder e colaborador,
// mapeando diretamente os campos da tabela tb_onebyone.
type OneByOne struct {
	// ID é o identificador único da reunião no formato UUID v4
	ID string `db:"id"`
	// UsuarioID é o UUID do líder que agendou a reunião
	UsuarioID string `db:"usuario_id"`
	// OrganizacaoID é o UUID da organização contexto da reunião
	OrganizacaoID string `db:"organizacao_id"`
	// EquipeID é o UUID da equipe contexto da reunião
	EquipeID string `db:"equipe_id"`
	// ColaborID é o UUID do colaborador participante da reunião
	ColaborID string `db:"colabor_id"`
	// Recorrencia define a frequência da reunião: NENHUMA, MENSAL ou QUINZENAL
	Recorrencia string `db:"recorrencia"`
	// Status representa o estado atual da reunião: AGENDADO, REALIZADO ou PENDENTE
	Status string `db:"status"`
	// RealizadoEm marca QUANDO a reunião foi de fato realizada (nil enquanto não realizada).
	// Preenchido ao encerrar o 1:1; base para cadência e streak (ver módulo saude1a1).
	RealizadoEm *time.Time `db:"realizado_em"`
	// DataAgendada é a data planejada para a realização da reunião
	DataAgendada time.Time `db:"data_agendada"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
}
