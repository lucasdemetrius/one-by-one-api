// Pacote: internal/onebyone
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de one-on-ones.
//            Inclui o método ResolverTemplateID que implementa a regra de herança
//            de template via COALESCE SQL — ver oneaone/usecase.go para a documentação
//            completa da regra de negócio.
// Autor: OneByOne API
// Criado em: 2025

package onebyone

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/autorizacao"
)

// Repositorio define o contrato de acesso ao banco para a entidade OneByOne
type Repositorio interface {
	// Criar insere uma nova reunião e retorna o registro persistido
	Criar(reuniao OneByOne) (OneByOne, error)
	// BuscarPorId retorna uma reunião ativa pelo UUID
	BuscarPorId(id string) (OneByOne, error)
	// ListarPorUsuario retorna todas as reuniões ativas de um líder
	ListarPorUsuario(usuarioID string) ([]OneByOne, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(reuniao OneByOne) (OneByOne, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
	// ResolverTemplateID resolve o template a ser usado para uma reunião seguindo
	// a ordem de prioridade: colaborador → equipe → organização → padrão do líder
	ResolverTemplateID(oneaoneID string) (string, error)
	// BuscarRealizadoNoDia retorna o 1:1 REALIZADO do colaborador num dia (idempotência
	// do encerrar). ok=false se não houver — evita duplicar a linha do livro-razão.
	BuscarRealizadoNoDia(colaborID string, dia time.Time) (OneByOne, bool, error)
	// GestorPertenceAoRH diz se o gestor dono (usuario_id) pertence ao tenant do RH
	// informado — fallback de posse para o papel RH (Cadeia A).
	GestorPertenceAoRH(gestorID, rhID string) (bool, error)
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de one-on-ones
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// GestorPertenceAoRH delega à primitiva compartilhada: o gestor dono (usuario_id) pertence
// ao tenant do RH informado? Fallback de posse para o papel RH (Cadeia A).
func (r *repositorioMySQL) GestorPertenceAoRH(gestorID, rhID string) (bool, error) {
	return autorizacao.GestorPertenceAoRH(r.db, gestorID, rhID)
}

// Criar insere uma nova reunião na tabela tb_onebyone e retorna o registro completo
func (r *repositorioMySQL) Criar(reuniao OneByOne) (OneByOne, error) {
	query := `
		INSERT INTO tb_onebyone
			(id, usuario_id, organizacao_id, equipe_id, colabor_id, recorrencia, status, realizado_em, data_agendada, criado_em)
		VALUES
			(:id, :usuario_id, :organizacao_id, :equipe_id, :colabor_id, :recorrencia, :status, :realizado_em, :data_agendada, :criado_em)
	`
	if _, err := r.db.NamedExec(query, reuniao); err != nil {
		return OneByOne{}, fmt.Errorf("erro ao inserir one-on-one: %w", err)
	}
	return r.BuscarPorId(reuniao.ID)
}

// BuscarPorId retorna uma reunião ativa pelo UUID
func (r *repositorioMySQL) BuscarPorId(id string) (OneByOne, error) {
	var o OneByOne
	query := `
		SELECT id, usuario_id, organizacao_id, equipe_id, colabor_id,
		       recorrencia, status, realizado_em, data_agendada, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_onebyone
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&o, query, id); err != nil {
		return OneByOne{}, fmt.Errorf("one-on-one não encontrado: %w", err)
	}
	return o, nil
}

// ListarPorUsuario retorna todas as reuniões ativas de um líder, ordenadas por data agendada
func (r *repositorioMySQL) ListarPorUsuario(usuarioID string) ([]OneByOne, error) {
	var reunioes []OneByOne
	query := `
		SELECT id, usuario_id, organizacao_id, equipe_id, colabor_id,
		       recorrencia, status, realizado_em, data_agendada, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_onebyone
		WHERE usuario_id = ? AND deletado_em IS NULL
		ORDER BY data_agendada DESC
	`
	if err := r.db.Select(&reunioes, query, usuarioID); err != nil {
		return nil, fmt.Errorf("erro ao listar one-on-ones: %w", err)
	}
	return reunioes, nil
}

// Atualizar aplica as modificações em uma reunião existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(reuniao OneByOne) (OneByOne, error) {
	agora := time.Now()
	reuniao.AlteradoEm = &agora

	query := `
		UPDATE tb_onebyone
		SET status = :status, recorrencia = :recorrencia, realizado_em = :realizado_em, data_agendada = :data_agendada, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, reuniao); err != nil {
		return OneByOne{}, fmt.Errorf("erro ao atualizar one-on-one: %w", err)
	}
	return r.BuscarPorId(reuniao.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_onebyone
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar one-on-one: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("one-on-one não encontrado ou já deletado")
	}
	return nil
}

// ResolverTemplateID resolve em uma única query SQL o template a ser usado para a reunião.
// A lógica usa COALESCE para aplicar a prioridade: colaborador > equipe > organização > padrão do líder.
// O "padrão do líder" é o template mais antigo criado por ele (ORDER BY criado_em ASC LIMIT 1).
func (r *repositorioMySQL) ResolverTemplateID(oneaoneID string) (string, error) {
	var templateID string

	// COALESCE retorna o primeiro valor não-nulo da lista, implementando a herança:
	// 1. colaborador.template_id  → prioridade máxima (configuração individual)
	// 2. equipe.template_id       → sobrescreve a organização se configurado
	// 3. organizacao.template_id  → fallback para toda a organização
	// 4. subquery do leader       → template padrão do líder (mais antigo = primeiro criado)
	query := `
		SELECT COALESCE(
			c.template_id,
			e.template_id,
			o.template_id,
			(
				SELECT t.id
				FROM tb_template t
				WHERE t.usuario_id = oo.usuario_id
				  AND t.deletado_em IS NULL
				ORDER BY t.criado_em ASC
				LIMIT 1
			)
		) AS template_id
		FROM tb_onebyone oo
		JOIN tb_colaboradores c  ON oo.colabor_id     = c.id  AND c.deletado_em IS NULL
		JOIN tb_equipes e        ON oo.equipe_id      = e.id  AND e.deletado_em IS NULL
		JOIN tb_organizacoes o   ON oo.organizacao_id = o.id  AND o.deletado_em IS NULL
		WHERE oo.id = ? AND oo.deletado_em IS NULL
	`

	if err := r.db.Get(&templateID, query, oneaoneID); err != nil {
		return "", fmt.Errorf("erro ao resolver template do one-on-one: %w", err)
	}

	// templateID vazio significa que nenhum template foi encontrado em nenhum nível da hierarquia
	if templateID == "" {
		return "", fmt.Errorf("nenhum template encontrado para esta reunião — configure um template no colaborador, equipe, organização ou crie um template padrão")
	}

	return templateID, nil
}

// BuscarRealizadoNoDia procura um 1:1 REALIZADO daquele colaborador na data informada.
// Serve para o encerrar ser idempotente: se o gestor encerrar duas vezes no mesmo dia,
// reaproveitamos a linha existente em vez de inflar a métrica com duplicatas.
func (r *repositorioMySQL) BuscarRealizadoNoDia(colaborID string, dia time.Time) (OneByOne, bool, error) {
	var o OneByOne
	query := `
		SELECT id, usuario_id, organizacao_id, equipe_id, colabor_id,
		       recorrencia, status, realizado_em, data_agendada, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_onebyone
		WHERE colabor_id = ? AND status = 'REALIZADO'
		  AND DATE(realizado_em) = ? AND deletado_em IS NULL
		LIMIT 1
	`
	err := r.db.Get(&o, query, colaborID, dia.Format("2006-01-02"))
	if err != nil {
		// Sem linha = ainda não encerrou hoje (não é erro de verdade).
		return OneByOne{}, false, nil
	}
	return o, true, nil
}
