// Pacote: internal/feedback
// Arquivo: usecase.go
// Descrição: Regras do feedback. Registrar (escrita de uma reação) e PainelAdmin (leitura
//            agregada para o dashboard: totais, índice de satisfação, série temporal por
//            reação, quebra por contexto e comentários recentes). Não conhece HTTP.
// Autor: OneByOne API
// Criado em: 2026

package feedback

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrReacaoInvalida: reação fora do conjunto suportado (defesa além do binding do DTO).
var ErrReacaoInvalida = errors.New("reação inválida")

// Limites de sanidade do painel.
const (
	diasPadrao  = 30
	diasMax     = 365
	recentesMax = 20
)

// UseCase define as operações do módulo de feedback.
type UseCase interface {
	// Registrar grava uma reação do usuário autenticado e devolve o resumo do que foi salvo.
	Registrar(usuarioID string, dto CriarFeedbackDTO) (FeedbackRespostaDTO, error)
	// PainelAdmin devolve o resumo de feedback para o dashboard de ADMIN.
	PainelAdmin(dias int) (PainelFeedbackDTO, error)
}

type useCaseImpl struct {
	repo Repositorio
}

// NovoUseCase cria o UseCase de feedback.
func NovoUseCase(repo Repositorio) UseCase {
	return &useCaseImpl{repo: repo}
}

// Registrar valida a reação, monta a entidade (campos opcionais vazios viram NULL) e grava.
func (uc *useCaseImpl) Registrar(usuarioID string, dto CriarFeedbackDTO) (FeedbackRespostaDTO, error) {
	if !ReacaoValida(dto.Reacao) {
		return FeedbackRespostaDTO{}, ErrReacaoInvalida
	}
	f := Feedback{
		ID:         uuid.New().String(),
		UsuarioID:  usuarioID,
		Reacao:     dto.Reacao,
		Contexto:   ptrSeNaoVazio(dto.Contexto),
		Comentario: ptrSeNaoVazio(dto.Comentario),
		Pagina:     ptrSeNaoVazio(dto.Pagina),
		CriadoEm:   time.Now(),
	}
	if err := uc.repo.Criar(f); err != nil {
		return FeedbackRespostaDTO{}, err
	}
	return ParaRespostaDTO(f), nil
}

// PainelAdmin orquestra as consultas e monta o DTO do dashboard.
func (uc *useCaseImpl) PainelAdmin(dias int) (PainelFeedbackDTO, error) {
	dias = limitarFaixa(dias, 1, diasMax, diasPadrao)

	resumo, err := uc.repo.Resumo(dias)
	if err != nil {
		return PainelFeedbackDTO{}, err
	}
	serieRows, err := uc.repo.Serie(dias)
	if err != nil {
		return PainelFeedbackDTO{}, err
	}
	contextoRows, err := uc.repo.PorContexto(dias)
	if err != nil {
		return PainelFeedbackDTO{}, err
	}
	recentesRows, err := uc.repo.Recentes(dias, recentesMax)
	if err != nil {
		return PainelFeedbackDTO{}, err
	}

	// Série: pivota (dia, reacao) em 3 arrays alinhados a `dias`, preenchendo com zero.
	labels := gerarDias(dias)
	pos := make(map[string]int, dias)
	for i, d := range labels {
		pos[d] = i
	}
	serieCurti := make([]int, dias)
	serieNaoCurti := make([]int, dias)
	serieIrritado := make([]int, dias)
	for _, r := range serieRows {
		i, ok := pos[r.Dia]
		if !ok {
			continue
		}
		switch r.Reacao {
		case ReacaoCurti:
			serieCurti[i] += r.Total
		case ReacaoNaoCurti:
			serieNaoCurti[i] += r.Total
		case ReacaoIrritado:
			serieIrritado[i] += r.Total
		}
	}

	porContexto := make([]ContextoFeedbackDTO, 0, len(contextoRows))
	for _, r := range contextoRows {
		porContexto = append(porContexto, ContextoFeedbackDTO{
			Contexto: r.Contexto, Curti: r.Curti, NaoCurti: r.NaoCurti,
			Irritado: r.Irritado, Total: r.Total,
		})
	}

	recentes := make([]ComentarioFeedbackDTO, 0, len(recentesRows))
	for _, r := range recentesRows {
		recentes = append(recentes, ComentarioFeedbackDTO{
			Reacao: r.Reacao, Contexto: r.Contexto, Comentario: r.Comentario,
			AutorNome: r.AutorNome, AutorPapel: r.AutorPapel, CriadoEm: r.CriadoEm,
		})
	}

	return PainelFeedbackDTO{
		Periodo:          dias,
		Total:            resumo.Total,
		Curti:            resumo.Curti,
		NaoCurti:         resumo.NaoCurti,
		Irritado:         resumo.Irritado,
		IndiceSatisfacao: percentual(resumo.Curti, resumo.Total),
		Dias:             labels,
		SerieCurti:       serieCurti,
		SerieNaoCurti:    serieNaoCurti,
		SerieIrritado:    serieIrritado,
		PorContexto:      porContexto,
		Recentes:         recentes,
		GeradoEm:         time.Now(),
	}, nil
}

// ─── Auxiliares ───────────────────────────────────────────────────────────────

// ptrSeNaoVazio devolve *s sem espaços nas pontas, ou nil quando vazio (para gravar NULL).
func ptrSeNaoVazio(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// gerarDias devolve os rótulos YYYY-MM-DD dos últimos `dias` dias terminando hoje.
func gerarDias(dias int) []string {
	hoje := time.Now()
	out := make([]string, dias)
	for i := 0; i < dias; i++ {
		out[i] = hoje.AddDate(0, 0, -(dias - 1 - i)).Format("2006-01-02")
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

// percentual devolve parte/total em % (0–100, uma casa), protegendo divisão por zero.
func percentual(parte, total int) float64 {
	if total <= 0 {
		return 0
	}
	p := float64(int(float64(parte)*1000/float64(total)+0.5)) / 10
	if p > 100 {
		return 100
	}
	return p
}
