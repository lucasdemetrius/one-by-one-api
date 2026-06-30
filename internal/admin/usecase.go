// Pacote: internal/admin
// Arquivo: usecase.go
// Descrição: Regras do painel de ADMIN. Orquestra as consultas do repositório e monta os
//            DTOs prontos para o front: preenche os buracos das séries temporais com zero
//            (gráficos contínuos), rotula dias da semana e calcula as taxas/percentuais dos
//            indicadores de saúde. Não conhece HTTP.
// Autor: OneByOne API
// Criado em: 2026

package admin

import (
	"time"
)

// Limites de sanidade dos parâmetros das consultas (evitam abuso e queries pesadas).
const (
	diasPadrao   = 30
	diasMax      = 365
	limiteContas = 50
	limiteMax    = 200
	topFuncMax   = 20
	topGestores  = 10
)

// nomesDiaSemana mapeia o DAYOFWEEK do MySQL (1=domingo … 7=sábado) para o nome curto.
var nomesDiaSemana = map[string]string{
	"1": "Dom", "2": "Seg", "3": "Ter", "4": "Qua", "5": "Qui", "6": "Sex", "7": "Sáb",
}

// UseCase define as operações de leitura do painel de admin.
type UseCase interface {
	// VisaoGeral devolve os cartões de KPI do topo do dashboard.
	VisaoGeral() (VisaoGeralDTO, error)
	// ListarContas devolve a página de contas (com resumo de uso de cada uma).
	ListarContas(papel, busca string, limite, offset int) (ContasPaginaDTO, error)
	// Acessos devolve a série temporal de acessos (gráfico estilo Google Analytics).
	Acessos(dias int) (SerieAcessosDTO, error)
	// Uso devolve as distribuições de uso (top funcionalidades, hora, dia, papel).
	Uso(dias int) (UsoDTO, error)
	// Crescimento devolve o crescimento de cadastros e de 1:1 ao longo do período.
	Crescimento(dias int) (CrescimentoDTO, error)
	// SaudePlataforma devolve os indicadores de engajamento/adoção + top gestores.
	SaudePlataforma() (SaudePlataformaDTO, error)
}

type useCaseImpl struct {
	repo Repositorio
}

// NovoUseCase cria o UseCase do painel de admin.
func NovoUseCase(repo Repositorio) UseCase {
	return &useCaseImpl{repo: repo}
}

// ─── Visão geral ──────────────────────────────────────────────────────────────

func (uc *useCaseImpl) VisaoGeral() (VisaoGeralDTO, error) {
	contas, err := uc.repo.ResumoContas()
	if err != nil {
		return VisaoGeralDTO{}, err
	}
	estrutura, err := uc.repo.ResumoEstrutura()
	if err != nil {
		return VisaoGeralDTO{}, err
	}
	atividade, err := uc.repo.ResumoAtividade()
	if err != nil {
		return VisaoGeralDTO{}, err
	}
	return VisaoGeralDTO{
		Contas: ContasResumoDTO{
			Total: contas.Total, Admin: contas.Admin, RH: contas.RH,
			Gestores: contas.Gestores, Liderados: contas.Liderados, Inativas: contas.Inativas,
			NovasHoje: contas.NovasHoje, Novas7d: contas.Novas7d, Novas30d: contas.Novas30d,
		},
		Estrutura: EstruturaDTO{
			Organizacoes: estrutura.Organizacoes, Equipes: estrutura.Equipes,
			Colaboradores: estrutura.Colaboradores, ColaboradoresComConta: estrutura.ColaboradoresComConta,
		},
		Atividade: AtividadeDTO{
			OnebyonesTotal: atividade.OnebyonesTotal, RealizadosTotal: atividade.RealizadosTotal,
			Realizados30d: atividade.Realizados30d, AgendamentosAtivos: atividade.AgendamentosAtivos,
			LoginsHoje: atividade.LoginsHoje, Logins7d: atividade.Logins7d,
			AtivosHoje: atividade.AtivosHoje, Ativos7d: atividade.Ativos7d, Ativos30d: atividade.Ativos30d,
			EventosHoje: atividade.EventosHoje,
		},
		GeradoEm: time.Now(),
	}, nil
}

// ─── Contas ───────────────────────────────────────────────────────────────────

func (uc *useCaseImpl) ListarContas(papel, busca string, limite, offset int) (ContasPaginaDTO, error) {
	limite = limitarFaixa(limite, 1, limiteMax, limiteContas)
	if offset < 0 {
		offset = 0
	}
	rows, total, err := uc.repo.ListarContas(papel, busca, limite, offset)
	if err != nil {
		return ContasPaginaDTO{}, err
	}
	itens := make([]ContaDTO, 0, len(rows))
	for _, r := range rows {
		itens = append(itens, ContaDTO{
			ID: r.ID, Nome: r.Nome, Email: r.Email, Role: r.Role, CriadoEm: r.CriadoEm,
			UltimoAcesso: r.UltimoAcesso, TotalEventos: r.TotalEventos,
			Equipes: r.Equipes, Colaboradores: r.Colaboradores, Onebyones: r.Onebyones, Gestores: r.Gestores,
		})
	}
	return ContasPaginaDTO{Itens: itens, Total: total, Limite: limite, Offset: offset}, nil
}

// ─── Acessos (série temporal) ─────────────────────────────────────────────────

func (uc *useCaseImpl) Acessos(dias int) (SerieAcessosDTO, error) {
	dias = limitarFaixa(dias, 1, diasMax, diasPadrao)
	rows, err := uc.repo.SerieAcessos(dias)
	if err != nil {
		return SerieAcessosDTO{}, err
	}
	// Indexa o resultado por dia para preencher os buracos com zero.
	porDia := make(map[string]AcessoDiaRow, len(rows))
	for _, r := range rows {
		porDia[r.Dia] = r
	}
	labels := gerarDias(dias)
	logins := make([]int, dias)
	ativos := make([]int, dias)
	eventos := make([]int, dias)
	totalLogs := 0
	for i, dia := range labels {
		if r, ok := porDia[dia]; ok {
			logins[i], ativos[i], eventos[i] = r.Logins, r.Ativos, r.Eventos
			totalLogs += r.Logins
		}
	}
	return SerieAcessosDTO{
		Dias: labels, Logins: logins, Ativos: ativos, Eventos: eventos,
		Periodo: dias, TotalLogs: totalLogs,
	}, nil
}

// ─── Uso (distribuições) ──────────────────────────────────────────────────────

func (uc *useCaseImpl) Uso(dias int) (UsoDTO, error) {
	dias = limitarFaixa(dias, 1, diasMax, diasPadrao)

	topRows, err := uc.repo.TopFuncionalidades(dias, topFuncMax)
	if err != nil {
		return UsoDTO{}, err
	}
	horaRows, err := uc.repo.DistribuicaoHora(dias)
	if err != nil {
		return UsoDTO{}, err
	}
	diaRows, err := uc.repo.DistribuicaoDiaSemana(dias)
	if err != nil {
		return UsoDTO{}, err
	}
	papelRows, err := uc.repo.AtividadePorPapel(dias)
	if err != nil {
		return UsoDTO{}, err
	}

	top := make([]FuncDTO, 0, len(topRows))
	for _, r := range topRows {
		top = append(top, FuncDTO{Entidade: r.Entidade, Acao: r.Acao, Total: r.Total})
	}

	// Hora: 24 baldes fixos (0–23), preenchendo com zero as horas sem atividade.
	horaTotais := map[string]int{}
	for _, r := range horaRows {
		horaTotais[r.Rotulo] = r.Total
	}
	porHora := make([]ContagemDTO, 24)
	for h := 0; h < 24; h++ {
		rot := numeroParaTexto(h)
		porHora[h] = ContagemDTO{Rotulo: rot + "h", Total: horaTotais[rot]}
	}

	// Dia da semana: 7 baldes fixos (Dom–Sáb), preenchendo com zero.
	diaTotais := map[string]int{}
	for _, r := range diaRows {
		diaTotais[r.Rotulo] = r.Total
	}
	porDiaSemana := make([]ContagemDTO, 7)
	for d := 1; d <= 7; d++ {
		chave := numeroParaTexto(d)
		porDiaSemana[d-1] = ContagemDTO{Rotulo: nomesDiaSemana[chave], Total: diaTotais[chave]}
	}

	porPapel := make([]ContagemDTO, 0, len(papelRows))
	for _, r := range papelRows {
		porPapel = append(porPapel, ContagemDTO{Rotulo: r.Rotulo, Total: r.Total})
	}

	return UsoDTO{
		Periodo: dias, TopFunc: top, PorHora: porHora,
		PorDiaSemana: porDiaSemana, PorPapel: porPapel,
	}, nil
}

// ─── Crescimento ──────────────────────────────────────────────────────────────

func (uc *useCaseImpl) Crescimento(dias int) (CrescimentoDTO, error) {
	dias = limitarFaixa(dias, 1, diasMax, 90)

	cresceRows, err := uc.repo.CrescimentoContas(dias)
	if err != nil {
		return CrescimentoDTO{}, err
	}
	realRows, err := uc.repo.SerieRealizados(dias)
	if err != nil {
		return CrescimentoDTO{}, err
	}

	labels := gerarDias(dias)
	pos := make(map[string]int, dias)
	for i, d := range labels {
		pos[d] = i
	}

	novosRH := make([]int, dias)
	novosGest := make([]int, dias)
	novosLid := make([]int, dias)
	novosTotal := make([]int, dias)
	realizados := make([]int, dias)

	for _, r := range cresceRows {
		i, ok := pos[r.Dia]
		if !ok {
			continue
		}
		switch r.Role {
		case "RH":
			novosRH[i] += r.Total
		case "LIDER":
			novosGest[i] += r.Total
		case "COLABORADOR":
			novosLid[i] += r.Total
		}
		// ADMIN não entra nas séries de crescimento (é conta operacional, não de cliente).
		if r.Role != "ADMIN" {
			novosTotal[i] += r.Total
		}
	}
	for _, r := range realRows {
		if i, ok := pos[r.Dia]; ok {
			realizados[i] += r.Total
		}
	}

	// Curva acumulada (soma corrente dos novos cadastros dentro da janela).
	acumulado := make([]int, dias)
	soma := 0
	for i := 0; i < dias; i++ {
		soma += novosTotal[i]
		acumulado[i] = soma
	}

	return CrescimentoDTO{
		Periodo: dias, Dias: labels,
		NovosRH: novosRH, NovosGestores: novosGest, NovosLiderados: novosLid,
		NovosTotal: novosTotal, AcumuladoTotal: acumulado, Realizados: realizados,
	}, nil
}

// ─── Saúde da plataforma ──────────────────────────────────────────────────────

func (uc *useCaseImpl) SaudePlataforma() (SaudePlataformaDTO, error) {
	s, err := uc.repo.SaudePlataforma()
	if err != nil {
		return SaudePlataformaDTO{}, err
	}
	topRows, err := uc.repo.TopGestores(topGestores)
	if err != nil {
		return SaudePlataformaDTO{}, err
	}

	top := make([]GestorEngajamentoDTO, 0, len(topRows))
	for _, r := range topRows {
		top = append(top, GestorEngajamentoDTO{
			ID: r.ID, Nome: r.Nome, Email: r.Email, Realizados: r.Realizados, Liderados: r.Liderados,
		})
	}

	return SaudePlataformaDTO{
		Gestores:                s.Gestores,
		GestoresCom1a1:          s.GestoresCom1a1,
		GestoresSem1a1:          naoNegativo(s.Gestores - s.GestoresCom1a1),
		PctGestoresEngajados:    percentual(s.GestoresCom1a1, s.Gestores),
		MediaLideradosPorGestor: media(s.LideradosAtivos, s.Gestores),
		LideradosAtivos:         s.LideradosAtivos,
		LideradosComConta:       s.LideradosComConta,
		PctLideradosVinculados:  percentual(s.LideradosComConta, s.LideradosAtivos),
		ContasComIA:             s.ContasComIA,
		ContasComFoto:           s.ContasComFoto,
		Realizados30d:           s.Realizados30d,
		TopGestores:             top,
		GeradoEm:                time.Now(),
	}, nil
}

// ─── Auxiliares ───────────────────────────────────────────────────────────────

// gerarDias devolve os rótulos YYYY-MM-DD dos últimos `dias` dias terminando hoje
// (do mais antigo ao mais recente), para alinhar as séries temporais.
func gerarDias(dias int) []string {
	hoje := time.Now()
	out := make([]string, dias)
	for i := 0; i < dias; i++ {
		d := hoje.AddDate(0, 0, -(dias - 1 - i))
		out[i] = d.Format("2006-01-02")
	}
	return out
}

// limitarFaixa limita `v` ao intervalo [minimo, maximo]; valores ≤ 0 viram o padrão.
func limitarFaixa(v, minimo, maximo, padrao int) int {
	if v <= 0 {
		return padrao
	}
	if v < minimo {
		return minimo
	}
	if v > maximo {
		return maximo
	}
	return v
}

// percentual devolve parte/total em percentual (0–100) com uma casa, protegendo divisão por
// zero e travando o teto em 100 (defesa contra eventual numerador > denominador).
func percentual(parte, total int) float64 {
	if total <= 0 {
		return 0
	}
	p := arredondar1(float64(parte) * 100 / float64(total))
	if p > 100 {
		return 100
	}
	return p
}

// media devolve parte/total protegendo divisão por zero (uma casa decimal).
func media(parte, total int) float64 {
	if total <= 0 {
		return 0
	}
	return arredondar1(float64(parte) / float64(total))
}

// arredondar1 arredonda para 1 casa decimal.
func arredondar1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

// naoNegativo evita negativos por inconsistência momentânea entre contagens.
func naoNegativo(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// numeroParaTexto converte um inteiro pequeno (0–23) em texto sem importar strconv à toa.
func numeroParaTexto(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
