// Pacote: internal/saude1a1
// Arquivo: usecase_test.go
// Descrição: Testes do cálculo de streak (semanas consecutivas com 1:1). É a lógica
//            mais sutil do módulo — semente da suíte de testes do projeto.
//            Rodar: go test ./internal/saude1a1/...
// Autor: OneByOne API
// Criado em: 2026

package saude1a1

import (
	"testing"
	"time"
)

// agora fixo (quarta-feira, 24/06/2026) para resultados determinísticos.
var agoraFixo = time.Date(2026, 6, 24, 15, 0, 0, 0, time.UTC)

// semanasAtras devolve uma data n semanas antes de agoraFixo.
func semanasAtras(n int) time.Time {
	return agoraFixo.AddDate(0, 0, -7*n)
}

func TestStreakVazio(t *testing.T) {
	if got := calcularStreak(nil, agoraFixo); got != 0 {
		t.Errorf("sem datas deveria dar streak 0, deu %d", got)
	}
}

func TestStreakSemanaAtual(t *testing.T) {
	// 1:1 nesta semana → streak 1.
	if got := calcularStreak([]time.Time{semanasAtras(0)}, agoraFixo); got != 1 {
		t.Errorf("esperado 1, deu %d", got)
	}
}

func TestStreakConsecutivas(t *testing.T) {
	// Esta semana + 2 anteriores seguidas → streak 3.
	datas := []time.Time{semanasAtras(0), semanasAtras(1), semanasAtras(2)}
	if got := calcularStreak(datas, agoraFixo); got != 3 {
		t.Errorf("esperado 3, deu %d", got)
	}
}

func TestStreakToleranteSemanaAtualVazia(t *testing.T) {
	// Nada nesta semana, mas houve na anterior → não penaliza: streak 1.
	datas := []time.Time{semanasAtras(1)}
	if got := calcularStreak(datas, agoraFixo); got != 1 {
		t.Errorf("semana atual vazia não deveria quebrar; esperado 1, deu %d", got)
	}
}

func TestStreakComBuraco(t *testing.T) {
	// Esta semana presente, mas falta a anterior (buraco) → streak 1.
	datas := []time.Time{semanasAtras(0), semanasAtras(2)}
	if got := calcularStreak(datas, agoraFixo); got != 1 {
		t.Errorf("buraco deveria parar a contagem; esperado 1, deu %d", got)
	}
}

func TestStreakIgnoraDuplicatasNaMesmaSemana(t *testing.T) {
	// Dois 1:1 na mesma semana contam como uma só semana.
	datas := []time.Time{semanasAtras(0), agoraFixo.AddDate(0, 0, -1), semanasAtras(1)}
	if got := calcularStreak(datas, agoraFixo); got != 2 {
		t.Errorf("duplicatas na mesma semana não somam; esperado 2, deu %d", got)
	}
}
