// Pacote: internal/rh
// Arquivo: usecase.go
// Descrição: Regras de negócio do módulo de RH. Cadastra gestores (delegando ao módulo
//            usuario, que deriva o rh_id) e dá visão consolidada do tenant: lista de
//            gestores com KPIs (reusa saude1a1) e drill-down nos 1:1 e agendas de cada
//            gestor — sempre validando que o gestor pertence ao tenant do RH.
// Autor: OneByOne API
// Criado em: 2026

package rh

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"onebyone-api/internal/agendamento"
	"onebyone-api/internal/onebyone"
	"onebyone-api/internal/organizacao"
	"onebyone-api/internal/saude1a1"
	"onebyone-api/internal/usuario"
)

// ErrGestorForaDoTenant indica que o gestor pedido não pertence ao tenant do RH. Mensagem
// genérica "não encontrado" → 404 no controller (não revela existência de gestor alheio).
var ErrGestorForaDoTenant = errors.New("gestor não encontrado")

// UseCase expõe as operações do RH sobre o seu tenant.
type UseCase interface {
	// CriarGestor cadastra um novo gestor (LIDER) já vinculado ao tenant do RH
	CriarGestor(rhID string, dto CriarGestorDTO) (GestorResumoDTO, error)
	// ListarGestores retorna os gestores do RH com seus KPIs de produtividade (dashboard)
	ListarGestores(rhID string) ([]GestorResumoDTO, error)
	// OneByonesDoGestor lista os 1:1 de um gestor do tenant (drill-down)
	OneByonesDoGestor(rhID, gestorID string) ([]onebyone.OneByOneRespostaDTO, error)
	// AgendamentosDoGestor lista a agenda de 1:1 de um gestor do tenant (drill-down)
	AgendamentosDoGestor(rhID, gestorID string) ([]agendamento.AgendamentoRespostaDTO, error)
	// AgendaDoTenant lista TODOS os 1:1 do tenant (com gestor/equipe) p/ o calendário consolidado
	AgendaDoTenant(rhID string) ([]AgendaItemDTO, error)
	// MatrixDoTenant lista TODOS os liderados do tenant com a 9-box (gestor/equipe)
	MatrixDoTenant(rhID string) ([]MatrixItemDTO, error)
	// AcompanhamentoDosGestores resume a EVOLUÇÃO dos liderados de cada gestor (foco em
	// qualidade: humor, risco, PDI, 9-box), ordenado por necessidade de atenção
	AcompanhamentoDosGestores(rhID string) ([]GestorEvolucaoDTO, error)
}

type useCaseImpl struct {
	repo          Repositorio
	usuarioUC     usuario.UseCase
	organizacaoUC organizacao.UseCase
	saudeUC       saude1a1.UseCase
	onebyoneUC    onebyone.UseCase
	agendamentoUC agendamento.UseCase
}

// NovoUseCase cria o UseCase de RH com as dependências dos módulos reaproveitados.
func NovoUseCase(repo Repositorio, usuarioUC usuario.UseCase, organizacaoUC organizacao.UseCase, saudeUC saude1a1.UseCase, onebyoneUC onebyone.UseCase, agendamentoUC agendamento.UseCase) UseCase {
	return &useCaseImpl{
		repo:          repo,
		usuarioUC:     usuarioUC,
		organizacaoUC: organizacaoUC,
		saudeUC:       saudeUC,
		onebyoneUC:    onebyoneUC,
		agendamentoUC: agendamentoUC,
	}
}

// CriarGestor delega a criação ao módulo usuario, que força role LIDER e amarra o rh_id
// ao RH (derivado do JWT). Devolve o gestor recém-criado (sem KPIs ainda).
func (uc *useCaseImpl) CriarGestor(rhID string, dto CriarGestorDTO) (GestorResumoDTO, error) {
	criado, err := uc.usuarioUC.CriarGestorParaRH(usuario.CriarUsuarioDTO{
		Nome:     dto.Nome,
		Email:    dto.Email,
		Password: dto.Password,
	}, rhID)
	if err != nil {
		return GestorResumoDTO{}, err
	}

	// O RH já monta a EMPRESA (organização) do gestor — assim, ao entrar, o gestor NÃO vê o
	// onboarding de "criar empresa"; ele só usa a empresa que o RH criou. Best-effort: se
	// isto falhar, o gestor ainda existe e poderá montar a empresa depois.
	empresa := strings.TrimSpace(dto.Empresa)
	if empresa == "" {
		empresa = "Minha empresa"
	}
	_, _ = uc.organizacaoUC.Criar(criado.ID, organizacao.CriarOrganizacaoDTO{Nome: empresa})

	return GestorResumoDTO{
		ID:       criado.ID,
		Nome:     criado.Nome,
		Email:    criado.Email,
		CriadoEm: criado.CriadoEm,
	}, nil
}

// ListarGestores devolve os gestores do tenant com os KPIs de produtividade embutidos.
// Os KPIs são best-effort: se a leitura da saúde de um gestor falhar, ele segue na lista
// com os contadores zerados (não derruba o dashboard inteiro).
func (uc *useCaseImpl) ListarGestores(rhID string) ([]GestorResumoDTO, error) {
	gs, err := uc.repo.ListarGestores(rhID)
	if err != nil {
		return nil, err
	}
	lista := make([]GestorResumoDTO, 0, len(gs))
	for _, g := range gs {
		item := GestorResumoDTO{ID: g.ID, Nome: g.Nome, Email: g.Email, CriadoEm: g.CriadoEm}
		if s, errSaude := uc.saudeUC.Obter(g.ID); errSaude == nil {
			item.PercentualEmDia = s.PercentualEmDia
			item.TotalAgendados = s.TotalAgendados
			item.Atrasados = s.Atrasados
			item.RealizadosUlt30 = s.RealizadosUlt30
			item.StreakSemanas = s.StreakSemanas
		}
		lista = append(lista, item)
	}
	return lista, nil
}

// OneByonesDoGestor valida que o gestor é do tenant do RH e então reaproveita a listagem
// já existente do módulo onebyone (escopada por usuario_id do gestor).
func (uc *useCaseImpl) OneByonesDoGestor(rhID, gestorID string) ([]onebyone.OneByOneRespostaDTO, error) {
	ok, err := uc.repo.GestorPertenceAoRH(gestorID, rhID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGestorForaDoTenant
	}
	return uc.onebyoneUC.ListarPorUsuario(gestorID)
}

// AgendamentosDoGestor valida o tenant e reaproveita a listagem de agendamentos do gestor.
func (uc *useCaseImpl) AgendamentosDoGestor(rhID, gestorID string) ([]agendamento.AgendamentoRespostaDTO, error) {
	ok, err := uc.repo.GestorPertenceAoRH(gestorID, rhID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGestorForaDoTenant
	}
	return uc.agendamentoUC.ListarPorUsuario(gestorID)
}

// AgendaDoTenant devolve a agenda consolidada de todos os gestores do RH, formatada para
// o calendário (com gestor e equipe para exibir e filtrar).
func (uc *useCaseImpl) AgendaDoTenant(rhID string) ([]AgendaItemDTO, error) {
	rows, err := uc.repo.ListarAgendaDoTenant(rhID)
	if err != nil {
		return nil, err
	}
	itens := make([]AgendaItemDTO, 0, len(rows))
	for _, r := range rows {
		itens = append(itens, AgendaItemDTO{
			ID:            r.ID,
			GestorID:      r.GestorID,
			GestorNome:    r.GestorNome,
			ColaboradorID: r.ColaboradorID,
			LideradoNome:  r.LideradoNome,
			EquipeID:      derefStr(r.EquipeID),
			EquipeNome:    derefStr(r.EquipeNome),
			DataHora:      r.DataHora.Format("2006-01-02T15:04"),
			Recorrencia:   r.Recorrencia,
			RepeteAte:     fmtDataOpc(r.RepeteAte),
		})
	}
	return itens, nil
}

// MatrixDoTenant devolve todos os liderados do tenant com a classificação 9-box.
func (uc *useCaseImpl) MatrixDoTenant(rhID string) ([]MatrixItemDTO, error) {
	rows, err := uc.repo.ListarMatrixDoTenant(rhID)
	if err != nil {
		return nil, err
	}
	itens := make([]MatrixItemDTO, 0, len(rows))
	for _, r := range rows {
		itens = append(itens, MatrixItemDTO{
			ColaboradorID: r.ColaboradorID,
			LideradoNome:  r.LideradoNome,
			GestorID:      r.GestorID,
			GestorNome:    r.GestorNome,
			EquipeID:      derefStr(r.EquipeID),
			EquipeNome:    derefStr(r.EquipeNome),
			Desempenho:    derefStr(r.Desempenho),
			Potencial:     derefStr(r.Potencial),
		})
	}
	return itens, nil
}

// AcompanhamentoDosGestores resume a EVOLUÇÃO dos liderados de cada gestor do tenant —
// tendência de humor do time, liderados em risco, progresso de PDI e lacunas de 9-box.
// Ordena por NECESSIDADE DE ATENÇÃO (mais liderados em risco primeiro), porque o objetivo
// do RH é descobrir com qual gestor sentar — não rankear quem fez mais reuniões.
func (uc *useCaseImpl) AcompanhamentoDosGestores(rhID string) ([]GestorEvolucaoDTO, error) {
	// Base: todos os liderados ativos com gestor e 9-box (reusa a query da Matrix).
	liderados, err := uc.repo.ListarMatrixDoTenant(rhID)
	if err != nil {
		return nil, err
	}
	humor, err := uc.repo.ListarHumorDoTenant(rhID)
	if err != nil {
		return nil, err
	}
	pdiItens, err := uc.repo.ListarPdiDoTenant(rhID)
	if err != nil {
		return nil, err
	}

	// Humor por liderado, na ordem antigo→recente (como veio do SQL).
	humorPorLiderado := map[string][]int{}
	for _, h := range humor {
		humorPorLiderado[h.LideradoID] = append(humorPorLiderado[h.LideradoID], h.Valor)
	}

	// PDI por liderado: total, concluídos e atrasados (prazo vencido e não concluído).
	type pdiStat struct{ total, concluidos, atrasados int }
	pdiPorLiderado := map[string]*pdiStat{}
	hoje := time.Now()
	for _, p := range pdiItens {
		st := pdiPorLiderado[p.LideradoID]
		if st == nil {
			st = &pdiStat{}
			pdiPorLiderado[p.LideradoID] = st
		}
		st.total++
		switch {
		case p.Concluido:
			st.concluidos++
		case p.Prazo != nil && p.Prazo.Before(hoje):
			st.atrasados++
		}
	}

	// Acumuladores de humor por gestor (média de médias e média de tendências).
	type acc struct {
		somaMedias, somaTend float64
		nMedias, nTend       int
	}
	accs := map[string]*acc{}
	indice := map[string]int{}
	resultado := []GestorEvolucaoDTO{}

	for _, l := range liderados {
		gi, ok := indice[l.GestorID]
		if !ok {
			gi = len(resultado)
			indice[l.GestorID] = gi
			resultado = append(resultado, GestorEvolucaoDTO{GestorID: l.GestorID, GestorNome: l.GestorNome, Riscos: []LideradoRiscoDTO{}})
			accs[l.GestorID] = &acc{}
		}
		g := &resultado[gi]
		a := accs[l.GestorID]

		g.TotalLiderados++
		if derefStr(l.Desempenho) == "" || derefStr(l.Potencial) == "" {
			g.SemClassificacao++
		}

		// PDI do liderado soma no total do gestor.
		var pdiAtrasado bool
		if st := pdiPorLiderado[l.ColaboradorID]; st != nil {
			g.PdiTotal += st.total
			g.PdiConcluidos += st.concluidos
			g.PdiAtrasados += st.atrasados
			pdiAtrasado = st.atrasados > 0
		}

		// Humor do liderado: média e tendência (recente − anterior).
		vals := humorPorLiderado[l.ColaboradorID]
		var media, tend float64
		temTend := false
		if len(vals) > 0 {
			media = mediaInts(vals)
			g.ComHumor++
			a.somaMedias += media
			a.nMedias++
		}
		if len(vals) >= 2 {
			meio := len(vals) / 2
			tend = mediaInts(vals[meio:]) - mediaInts(vals[:meio])
			temTend = true
			a.somaTend += tend
			a.nTend++
		}

		// Risco: humor caindo (queda relevante), humor baixo, ou PDI atrasado.
		motivo := ""
		switch {
		case temTend && tend <= -0.5:
			motivo = "humor caindo"
		case len(vals) > 0 && media <= 2.0:
			motivo = "humor baixo"
		}
		if pdiAtrasado {
			if motivo == "" {
				motivo = "PDI atrasado"
			} else {
				motivo += " + PDI atrasado"
			}
		}
		if motivo != "" {
			g.LideradosEmRisco++
			g.Riscos = append(g.Riscos, LideradoRiscoDTO{ColaboradorID: l.ColaboradorID, Nome: l.LideradoNome, Motivo: motivo})
		}
	}

	// Fecha as médias de humor por gestor.
	for i := range resultado {
		a := accs[resultado[i].GestorID]
		if a.nMedias > 0 {
			resultado[i].HumorMedia = arred1(a.somaMedias / float64(a.nMedias))
		}
		if a.nTend > 0 {
			resultado[i].HumorTendencia = arred1(a.somaTend / float64(a.nTend))
		}
	}

	// Ordena por necessidade de atenção: mais liderados em risco primeiro; no empate,
	// quem tem o humor caindo mais (tendência menor) vem antes.
	sort.SliceStable(resultado, func(i, j int) bool {
		if resultado[i].LideradosEmRisco != resultado[j].LideradosEmRisco {
			return resultado[i].LideradosEmRisco > resultado[j].LideradosEmRisco
		}
		return resultado[i].HumorTendencia < resultado[j].HumorTendencia
	})
	return resultado, nil
}

// mediaInts devolve a média de uma fatia de inteiros (0 se vazia).
func mediaInts(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	soma := 0
	for _, x := range xs {
		soma += x
	}
	return float64(soma) / float64(len(xs))
}

// arred1 arredonda para 1 casa decimal.
func arred1(f float64) float64 {
	return math.Round(f*10) / 10
}

// derefStr devolve o valor do ponteiro ou "" se nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// fmtDataOpc formata uma data opcional como "YYYY-MM-DD" (vazio se nil).
func fmtDataOpc(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
