// Pacote: internal/saude1a1
// Arquivo: repository.go
// Descrição: Leitura agregada (somente COUNT/datas, sem carregar listas) sobre
//            tb_onebyone e tb_agendamentos, sempre escopada pelo usuario_id do gestor
//            (Cadeia A de posse). Não escreve nada.
// Autor: OneByOne API
// Criado em: 2026

package saude1a1

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o acesso de leitura para a saúde do 1:1.
type Repositorio interface {
	// Coletar reúne as métricas cruas do gestor para uma data de referência (agora).
	Coletar(usuarioID string, agora time.Time) (Metricas, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de saúde do 1:1.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Coletar(usuarioID string, agora time.Time) (Metricas, error) {
	var m Metricas

	// Cadência esperada: agendamentos ativos do gestor.
	if err := r.db.Get(&m.TotalAgendados,
		`SELECT COUNT(*) FROM tb_agendamentos WHERE usuario_id = ? AND ativo = 1`, usuarioID); err != nil {
		return m, fmt.Errorf("erro ao contar agendamentos: %w", err)
	}

	// Atrasados: agendamento ativo cuja próxima ocorrência já passou.
	if err := r.db.Get(&m.Atrasados,
		`SELECT COUNT(*) FROM tb_agendamentos WHERE usuario_id = ? AND ativo = 1 AND data_hora < ?`,
		usuarioID, agora); err != nil {
		return m, fmt.Errorf("erro ao contar atrasados: %w", err)
	}

	// Realizados nos últimos 30 dias (livro-razão tb_onebyone).
	if err := r.db.Get(&m.RealizadosUlt30,
		`SELECT COUNT(*) FROM tb_onebyone
		  WHERE usuario_id = ? AND status = 'REALIZADO'
		    AND realizado_em >= ? AND deletado_em IS NULL`,
		usuarioID, agora.AddDate(0, 0, -30)); err != nil {
		return m, fmt.Errorf("erro ao contar realizados: %w", err)
	}

	// Datas (sem hora) dos realizados — recentes primeiro, limitadas para não pesar.
	// Usadas no cálculo do streak (semanas consecutivas).
	var datas []time.Time
	if err := r.db.Select(&datas,
		`SELECT DISTINCT DATE(realizado_em) AS dia
		   FROM tb_onebyone
		  WHERE usuario_id = ? AND status = 'REALIZADO'
		    AND realizado_em IS NOT NULL AND deletado_em IS NULL
		  ORDER BY dia DESC
		  LIMIT 120`, usuarioID); err != nil {
		return m, fmt.Errorf("erro ao listar datas de realizados: %w", err)
	}
	m.DatasRealizados = datas

	return m, nil
}
