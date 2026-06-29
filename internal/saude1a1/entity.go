// Pacote: internal/saude1a1
// Arquivo: entity.go
// Descrição: Estrutura interna com as métricas cruas lidas do banco para compor a
//            "Saúde do 1:1". Não há tabela própria: o módulo só LÊ tb_onebyone
//            (realizados) e tb_agendamentos (cadência esperada) do próprio gestor.
// Autor: OneByOne API
// Criado em: 2026

package saude1a1

import "time"

// Metricas reúne os números crus que o repositório coleta para o usecase calcular
// percentual e streak.
type Metricas struct {
	// TotalAgendados é a quantidade de agendamentos ATIVOS do gestor (cadência esperada).
	TotalAgendados int
	// Atrasados são agendamentos ativos cuja próxima ocorrência já passou.
	Atrasados int
	// RealizadosUlt30 é a contagem de 1:1 marcados REALIZADO nos últimos 30 dias.
	RealizadosUlt30 int
	// DatasRealizados são as datas (sem hora) dos 1:1 realizados, recentes primeiro —
	// usadas para calcular a sequência (streak) de semanas consecutivas.
	DatasRealizados []time.Time
}
