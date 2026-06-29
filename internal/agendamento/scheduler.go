// Pacote: internal/agendamento
// Arquivo: scheduler.go
// Descrição: "Cron" dos lembretes de 1:1. Roda em segundo plano: avança as
//            ocorrências recorrentes que já passaram e envia, uma vez por dia, um
//            e-mail para cada gestor com os 1:1 de HOJE e AMANHÃ. (Dormente se o
//            SMTP não estiver configurado — o serviço de e-mail apenas loga.)
// Autor: OneByOne API
// Criado em: 2025

package agendamento

import (
	"log"
	"time"

	"onebyone-api/pkg/email"
)

// Scheduler dispara os lembretes periodicamente.
type Scheduler struct {
	repo     Repositorio
	emailSvc email.Servico
}

// NovoScheduler cria o agendador de lembretes.
func NovoScheduler(repo Repositorio, emailSvc email.Servico) *Scheduler {
	return &Scheduler{repo: repo, emailSvc: emailSvc}
}

// Iniciar roda a verificação ao subir e, depois, a cada 24h, em segundo plano.
func (s *Scheduler) Iniciar() {
	go func() {
		s.Executar()
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for range t.C {
			s.Executar()
		}
	}()
}

// avancar soma o período da recorrência a partir de uma data.
func avancar(t time.Time, rec string) time.Time {
	switch rec {
	case RecSemanal:
		return t.AddDate(0, 0, 7)
	case RecQuinzenal:
		return t.AddDate(0, 0, 14)
	case RecMensal:
		return t.AddDate(0, 1, 0)
	case RecBimestral:
		return t.AddDate(0, 2, 0)
	case RecTrimestral:
		return t.AddDate(0, 3, 0)
	case RecSemestral:
		return t.AddDate(0, 6, 0)
	default:
		return t
	}
}

func mesmaData(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// aposData diz se a DATA de `a` é estritamente posterior à DATA de `b` (ignora a hora).
// Usado para encerrar a recorrência quando a próxima ocorrência passa de repete_ate.
func aposData(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return time.Date(ay, am, ad, 0, 0, 0, 0, time.Local).After(time.Date(by, bm, bd, 0, 0, 0, 0, time.Local))
}

// Executar é a rotina do scheduler (também útil para acionar manualmente/testar).
func (s *Scheduler) Executar() {
	lista, err := s.repo.ListarParaLembrete()
	if err != nil {
		log.Printf("[agenda] erro ao buscar agendamentos: %v", err)
		return
	}

	agora := time.Now()
	amanha := agora.AddDate(0, 0, 1)

	type digest struct {
		nome  string
		itens []email.ItemLembrete
	}
	porGestor := make(map[string]*digest)

	for _, a := range lista {
		dh := a.DataHora

		// Avança (ou desativa) ocorrências que já passaram.
		if dh.Before(agora) {
			if a.Recorrencia == RecNenhuma {
				_ = s.repo.Desativar(a.ID)
				continue
			}
			for dh.Before(agora) {
				dh = avancar(dh, a.Recorrencia)
			}
			// Recorrência com fim ("repete até"): se a próxima ocorrência passou da data
			// final, encerra (desativa) em vez de avançar para sempre.
			if a.RepeteAte != nil && aposData(dh, *a.RepeteAte) {
				_ = s.repo.Desativar(a.ID)
				continue
			}
			_ = s.repo.AtualizarDataHora(a.ID, dh)
		}

		// Entra no digest se for hoje ou amanhã.
		if mesmaData(dh, agora) || mesmaData(dh, amanha) {
			g := porGestor[a.GestorEmail]
			if g == nil {
				g = &digest{nome: a.GestorNome}
				porGestor[a.GestorEmail] = g
			}
			prefixo := "hoje"
			if mesmaData(dh, amanha) {
				prefixo = "amanhã"
			}
			g.itens = append(g.itens, email.ItemLembrete{
				Liderado: a.LideradoNome,
				Quando:   prefixo + ", " + dh.Format("02/01 15:04"),
			})
		}
	}

	for endereco, g := range porGestor {
		if len(g.itens) == 0 {
			continue
		}
		assunto, html := email.TemplateLembrete(g.nome, g.itens)
		_ = s.emailSvc.EnviarHTML([]string{endereco}, assunto, html)
	}

	log.Printf("[agenda] lembretes verificados — %d gestor(es) com 1:1 hoje/amanhã", len(porGestor))
}
