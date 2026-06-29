// Pacote: internal/agendamento
// Arquivo: entity.go
// Descrição: Entidade Agendamento (1:1 agendado, com recorrência) + uma struct de
//            contexto com os nomes/e-mail para os lembretes. Tabela tb_agendamentos.
// Autor: OneByOne API
// Criado em: 2025

package agendamento

import "time"

// Tipos de recorrência.
const (
	RecNenhuma    = "NENHUMA"
	RecSemanal    = "SEMANAL"
	RecQuinzenal  = "QUINZENAL"
	RecMensal     = "MENSAL"     // 1 mês
	RecBimestral  = "BIMESTRAL"  // 2 meses
	RecTrimestral = "TRIMESTRAL" // 3 meses
	RecSemestral  = "SEMESTRAL"  // 6 meses
)

// Agendamento é um 1:1 marcado entre um gestor e um liderado.
type Agendamento struct {
	ID            string     `db:"id"`
	UsuarioID     string     `db:"usuario_id"`     // gestor dono
	ColaboradorID string     `db:"colaborador_id"` // liderado
	DataHora      time.Time  `db:"data_hora"`      // próxima ocorrência
	Recorrencia   string     `db:"recorrencia"`
	RepeteAte     *time.Time `db:"repete_ate"` // fim da recorrência (nil = para sempre)
	Ativo         bool       `db:"ativo"`
	CriadoEm      time.Time  `db:"criado_em"`
}

// AgendamentoContexto junta o agendamento com nomes/e-mail (para listagem e lembretes).
type AgendamentoContexto struct {
	ID            string     `db:"id"`
	UsuarioID     string     `db:"usuario_id"`
	ColaboradorID string     `db:"colaborador_id"`
	GestorNome    string     `db:"gestor_nome"`
	GestorEmail   string     `db:"gestor_email"`
	LideradoNome  string     `db:"liderado_nome"`
	DataHora      time.Time  `db:"data_hora"`
	Recorrencia   string     `db:"recorrencia"`
	RepeteAte     *time.Time `db:"repete_ate"`
}
