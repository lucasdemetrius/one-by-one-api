// Pacote: internal/notificacao
// Arquivo: scheduler.go
// Descrição: "Cron" das notificações in-app. A cada 30 min varre a agenda e gera
//            avisos por FAIXA (robusto a atraso/reinício): ~1h antes, no dia de
//            manhã e 1 dia antes. Cada aviso é deduplicado pela `chave` (uma vez
//            por usuário+tipo+agendamento+ocorrência) e só é criado se a PREFERÊNCIA
//            do destinatário permitir. Gera para o gestor e para o liderado.
// Autor: OneByOne API
// Criado em: 2026

package notificacao

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Scheduler gera as notificações periodicamente.
type Scheduler struct {
	repo Repositorio
}

// NovoScheduler cria o agendador de notificações.
func NovoScheduler(repo Repositorio) *Scheduler {
	return &Scheduler{repo: repo}
}

// Iniciar roda ao subir e, depois, a cada 30 minutos, em segundo plano.
func (s *Scheduler) Iniciar() {
	go func() {
		s.Executar()
		t := time.NewTicker(30 * time.Minute)
		defer t.Stop()
		for range t.C {
			s.Executar()
		}
	}()
}

func mesmaData(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// proxima devolve a próxima ocorrência (>= agora) considerando a recorrência.
// Para agendamento NÃO recorrente que já passou, devolve ok=false.
func proxima(dh time.Time, rec string, agora time.Time) (time.Time, bool) {
	if !dh.Before(agora) {
		return dh, true
	}
	for i := 0; i < 600 && dh.Before(agora); i++ {
		switch rec {
		case "SEMANAL":
			dh = dh.AddDate(0, 0, 7)
		case "QUINZENAL":
			dh = dh.AddDate(0, 0, 14)
		case "MENSAL":
			dh = dh.AddDate(0, 1, 0)
		default:
			return time.Time{}, false // NENHUMA — já passou
		}
	}
	if dh.Before(agora) {
		return time.Time{}, false
	}
	return dh, true
}

// Executar varre a agenda e gera as notificações pendentes (uma passada do cron).
func (s *Scheduler) Executar() {
	pend, err := s.repo.ListarAgendaPendente()
	if err != nil {
		log.Printf("[notif] erro ao ler agenda: %v", err)
		return
	}
	agora := time.Now()
	prefs := map[string]Pref{} // cache por usuário nesta passada
	getPref := func(uid string) Pref {
		if p, ok := prefs[uid]; ok {
			return p
		}
		p, e := s.repo.ObterPref(uid)
		if e != nil {
			p = PrefPadrao(uid)
		}
		prefs[uid] = p
		return p
	}

	criadas := 0
	criar := func(uid, tipo, titulo, msg string, link *string, agID string, occ time.Time) {
		if uid == "" || !getPref(uid).Ligado(tipo) {
			return
		}
		chave := fmt.Sprintf("%s|%s|%s|%s", uid, tipo, agID, occ.Format("2006-01-02"))
		if err := s.repo.CriarSeNova(Notificacao{
			ID: uuid.New().String(), UsuarioID: uid, Tipo: tipo, Titulo: titulo,
			Mensagem: msg, Link: link, Chave: chave, CriadoEm: time.Now(),
		}); err == nil {
			criadas++
		}
	}

	linkAgenda := "/agenda"
	for _, p := range pend {
		occ, ok := proxima(p.DataHora, p.Recorrencia, agora)
		if !ok {
			continue
		}
		hora := occ.Format("15:04")
		diff := occ.Sub(agora)

		var tipos []string
		if diff > 0 && diff <= 90*time.Minute {
			tipos = append(tipos, TipoAgenda1H)
		}
		if mesmaData(occ, agora) && agora.Hour() >= 6 {
			tipos = append(tipos, TipoAgendaHoje)
		}
		if mesmaData(occ, agora.AddDate(0, 0, 1)) {
			tipos = append(tipos, TipoAgenda1Dia)
		}

		for _, tipo := range tipos {
			tg, mg := conteudo(tipo, "com "+p.LideradoNome, hora)
			criar(p.GestorID, tipo, tg, mg, &linkAgenda, p.AgendamentoID, occ)
			if p.LideradoUsuario != nil && *p.LideradoUsuario != "" {
				tl, ml := conteudo(tipo, "com "+p.GestorNome, hora)
				criar(*p.LideradoUsuario, tipo, tl, ml, nil, p.AgendamentoID, occ)
			}
		}
	}
	log.Printf("[notif] verificação concluída — %d notificação(ões) nova(s)", criadas)
}

// conteudo monta (título, mensagem) de um tipo. `quem` ex.: "com Maria".
func conteudo(tipo, quem, hora string) (string, string) {
	switch tipo {
	case TipoAgenda1Dia:
		return "🗓️ 1:1 amanhã", fmt.Sprintf("Amanhã às %s você tem um 1:1 %s.", hora, quem)
	case TipoAgendaHoje:
		return "🗓️ 1:1 hoje", fmt.Sprintf("Hoje às %s você tem um 1:1 %s.", hora, quem)
	case TipoAgenda1H:
		return "⏰ 1:1 em ~1h", fmt.Sprintf("Seu 1:1 %s começa às %s.", quem, hora)
	}
	return "Lembrete de 1:1", fmt.Sprintf("1:1 %s às %s.", quem, hora)
}
