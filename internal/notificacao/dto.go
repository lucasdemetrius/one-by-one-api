// Pacote: internal/notificacao
// Arquivo: dto.go
// Descrição: Contratos HTTP do módulo de notificação.
// Autor: OneByOne API
// Criado em: 2026

package notificacao

import "time"

// NotificacaoRespostaDTO é a notificação devolvida ao cliente (sino).
type NotificacaoRespostaDTO struct {
	ID       string    `json:"id"`
	Tipo     string    `json:"tipo"`
	Titulo   string    `json:"titulo"`
	Mensagem string    `json:"mensagem"`
	Link     *string   `json:"link"`
	Lida     bool      `json:"lida"`
	CriadoEm time.Time `json:"criado_em"`
}

// PrefDTO são as preferências de notificação (ligar/desligar cada tipo).
type PrefDTO struct {
	Agenda1Dia bool `json:"agenda_1dia"`
	AgendaHoje bool `json:"agenda_hoje"`
	Agenda1H   bool `json:"agenda_1h"`
}
