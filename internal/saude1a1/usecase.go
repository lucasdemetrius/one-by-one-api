// Pacote: internal/saude1a1
// Arquivo: usecase.go
// Descrição: Calcula a "Saúde do 1:1" do gestor: % da agenda em dia, atrasados,
//            realizados (30d) e o STREAK de semanas consecutivas com 1:1. O streak é
//            calculado em Go (não em SQL) e é "tolerante": a semana atual ainda sem
//            1:1 não quebra a sequência (você ainda tem a semana para fazer).
// Autor: OneByOne API
// Criado em: 2026

package saude1a1

import (
	"fmt"
	"math"
	"time"
)

// UseCase expõe a leitura da saúde do 1:1.
type UseCase interface {
	Obter(usuarioID string) (SaudeRespostaDTO, error)
}

type useCaseImpl struct {
	repo Repositorio
}

// NovoUseCase cria o UseCase de saúde do 1:1.
func NovoUseCase(repo Repositorio) UseCase {
	return &useCaseImpl{repo: repo}
}

func (uc *useCaseImpl) Obter(usuarioID string) (SaudeRespostaDTO, error) {
	agora := time.Now()
	m, err := uc.repo.Coletar(usuarioID, agora)
	if err != nil {
		return SaudeRespostaDTO{}, err
	}

	// % em dia = (agendados - atrasados) / agendados. Sem agenda → 100 (nada vencido).
	percentual := 100
	if m.TotalAgendados > 0 {
		emDia := m.TotalAgendados - m.Atrasados
		percentual = int(math.Round(float64(emDia) / float64(m.TotalAgendados) * 100))
		if percentual < 0 {
			percentual = 0
		}
		if percentual > 100 {
			percentual = 100
		}
	}

	return SaudeRespostaDTO{
		PercentualEmDia: percentual,
		TotalAgendados:  m.TotalAgendados,
		Atrasados:       m.Atrasados,
		RealizadosUlt30: m.RealizadosUlt30,
		StreakSemanas:   calcularStreak(m.DatasRealizados, agora),
	}, nil
}

// chaveSemana identifica a semana ISO de uma data como "ano-semana" (ex.: "2026-25").
func chaveSemana(t time.Time) string {
	ano, semana := t.ISOWeek()
	return fmt.Sprintf("%d-%02d", ano, semana)
}

// calcularStreak conta semanas ISO consecutivas (a partir de agora, andando para trás)
// que tiveram pelo menos um 1:1 realizado. É tolerante: se a semana atual ainda não tem
// 1:1, a contagem começa na semana passada (não penaliza o meio da semana).
func calcularStreak(datas []time.Time, agora time.Time) int {
	if len(datas) == 0 {
		return 0
	}
	semanas := make(map[string]bool, len(datas))
	for _, d := range datas {
		semanas[chaveSemana(d)] = true
	}

	cursor := agora
	if !semanas[chaveSemana(cursor)] {
		// semana atual sem 1:1 ainda não quebra o streak — começa pela anterior
		cursor = cursor.AddDate(0, 0, -7)
	}

	streak := 0
	for semanas[chaveSemana(cursor)] {
		streak++
		cursor = cursor.AddDate(0, 0, -7)
	}
	return streak
}
