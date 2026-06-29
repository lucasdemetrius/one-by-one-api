// Pacote: internal/notificacao
// Arquivo: entity.go
// Descrição: Entidades de notificação in-app: a notificação em si (com chave de
//            dedupe), as preferências do usuário (ligar/desligar cada tipo) e a
//            "agenda pendente" lida pelo cron para gerar os avisos.
// Autor: OneByOne API
// Criado em: 2026

package notificacao

import "time"

// Tipos de notificação gerados a partir da agenda.
const (
	TipoAgenda1Dia = "AGENDA_1DIA" // 1 dia antes
	TipoAgendaHoje = "AGENDA_HOJE" // no dia, de manhã
	TipoAgenda1H   = "AGENDA_1H"   // ~1 hora antes
)

// Notificacao mapeia a tabela tb_notificacoes.
type Notificacao struct {
	ID        string    `db:"id"`
	UsuarioID string    `db:"usuario_id"`
	Tipo      string    `db:"tipo"`
	Titulo    string    `db:"titulo"`
	Mensagem  string    `db:"mensagem"`
	Link      *string   `db:"link"`
	Chave     string    `db:"chave"`
	Lida      bool      `db:"lida"`
	CriadoEm  time.Time `db:"criado_em"`
}

// Pref mapeia tb_pref_notificacoes (preferências de notificação por usuário).
type Pref struct {
	UsuarioID  string `db:"usuario_id"`
	Agenda1Dia bool   `db:"agenda_1dia"`
	AgendaHoje bool   `db:"agenda_hoje"`
	Agenda1H   bool   `db:"agenda_1h"`
}

// PrefPadrao: tudo ligado (o usuário desliga o que não quiser).
func PrefPadrao(usuarioID string) Pref {
	return Pref{UsuarioID: usuarioID, Agenda1Dia: true, AgendaHoje: true, Agenda1H: true}
}

// Ligado diz se a preferência permite o tipo informado.
func (p Pref) Ligado(tipo string) bool {
	switch tipo {
	case TipoAgenda1Dia:
		return p.Agenda1Dia
	case TipoAgendaHoje:
		return p.AgendaHoje
	case TipoAgenda1H:
		return p.Agenda1H
	}
	return true
}

// AgendaPendente é uma linha lida pelo cron (agendamento + destinatários).
type AgendaPendente struct {
	AgendamentoID   string    `db:"agendamento_id"`
	GestorID        string    `db:"gestor_id"`
	GestorNome      string    `db:"gestor_nome"`
	LideradoUsuario *string   `db:"liderado_usuario"` // conta do liderado (nulo = não vinculou)
	LideradoNome    string    `db:"liderado_nome"`
	DataHora        time.Time `db:"data_hora"`
	Recorrencia     string    `db:"recorrencia"`
}
