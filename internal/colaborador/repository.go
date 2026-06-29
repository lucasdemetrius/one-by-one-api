// Pacote: internal/colaborador
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de colaboradores.
//            Toda interação com a tabela tb_colaboradores passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package colaborador

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o contrato de acesso ao banco para a entidade Colaborador
type Repositorio interface {
	// Criar insere um novo colaborador e retorna o registro persistido
	Criar(col Colaborador) (Colaborador, error)
	// BuscarPorId retorna um colaborador ativo pelo UUID
	BuscarPorId(id string) (Colaborador, error)
	// BuscarPorUsuarioID retorna o colaborador ativo mais recente vinculado a uma conta de usuário
	BuscarPorUsuarioID(usuarioID string) (Colaborador, error)
	// ListarPorEquipe retorna todos os colaboradores ativos de uma equipe
	ListarPorEquipe(equipeID string) ([]Colaborador, error)
	// ListarPorOrganizacao retorna todos os colaboradores ativos de uma organização
	ListarPorOrganizacao(organizacaoID string) ([]Colaborador, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(col Colaborador) (Colaborador, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
	// AtualizarFoto persiste a chave S3 da foto no banco de dados
	AtualizarFoto(id string, fotoKey string) error
	// PertenceAoLider diz se o colaborador pertence à estrutura (equipe OU
	// organização) do líder informado. É a checagem de posse da "Cadeia B".
	PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error)
	// ExisteEmailNoLider diz se o líder JÁ tem um liderado ativo com este e-mail
	// (na estrutura dele — Cadeia B). Ignora o colaborador `excetoID` (usado no
	// update para não conflitar consigo mesmo); passe "" ao criar.
	ExisteEmailNoLider(email, usuarioLiderID, excetoID string) (bool, error)
	// EmailEhDoLider diz se o e-mail é o da PRÓPRIA conta do gestor logado. Um
	// liderado não pode usar o e-mail do gestor (sequestraria a conta do líder).
	EmailEhDoLider(email, usuarioLiderID string) (bool, error)
	// EquipePertenceAoLider diz se a equipe é do líder informado.
	EquipePertenceAoLider(equipeID, usuarioLiderID string) (bool, error)
	// OrganizacaoPertenceAoLider diz se a organização é do líder informado.
	OrganizacaoPertenceAoLider(organizacaoID, usuarioLiderID string) (bool, error)
	// VincularUsuario amarra a conta de usuário (liderado) ao colaborador.
	// Usado SÓ pelo fluxo de aceite de convite — não passa pelo Atualizar geral.
	VincularUsuario(colaboradorID, usuarioID string) error
	// DesvincularOutrasContas solta o usuario_id de TODOS os outros colaboradores desta
	// conta (exceto o informado). Garante "uma conta ↔ um colaborador atual": ao trocar de
	// empresa, a pessoa perde o acesso ao 1:1 da anterior (o histórico fica com o gestor).
	DesvincularOutrasContas(usuarioID, excetoColaboradorID string) (int64, error)
	// Desligar marca o colaborador como inativo (data de desligamento).
	Desligar(id string, desligadoEm time.Time) error
	// Reativar limpa a data de desligamento (volta a ativo).
	Reativar(id string) error
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de colaboradores
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// colunas é a lista de colunas usada nas queries de SELECT para evitar repetição
const colunas = `
	id, usuario_id, organizacao_id, equipe_id, template_id,
	nome, email, whatsapp, data_nascimento, foto_key,
	criado_em, alterado_em, deletado_em, deletado_por, desligado_em
`

// Criar insere um novo colaborador na tabela tb_colaboradores e retorna o registro completo
func (r *repositorioMySQL) Criar(col Colaborador) (Colaborador, error) {
	query := `
		INSERT INTO tb_colaboradores
			(id, usuario_id, organizacao_id, equipe_id, template_id, nome, email, whatsapp, data_nascimento, criado_em)
		VALUES
			(:id, :usuario_id, :organizacao_id, :equipe_id, :template_id, :nome, :email, :whatsapp, :data_nascimento, :criado_em)
	`
	if _, err := r.db.NamedExec(query, col); err != nil {
		return Colaborador{}, fmt.Errorf("erro ao inserir colaborador: %w", err)
	}
	return r.BuscarPorId(col.ID)
}

// BuscarPorId retorna um colaborador ativo pelo UUID
func (r *repositorioMySQL) BuscarPorId(id string) (Colaborador, error) {
	var col Colaborador
	query := `SELECT ` + colunas + ` FROM tb_colaboradores WHERE id = ? AND deletado_em IS NULL`
	if err := r.db.Get(&col, query, id); err != nil {
		return Colaborador{}, fmt.Errorf("colaborador não encontrado: %w", err)
	}
	return col, nil
}

// BuscarPorUsuarioID retorna o colaborador ativo mais recente vinculado a uma conta.
// (Um liderado pode ter mais de um vínculo ao longo do tempo — pega o mais novo.)
func (r *repositorioMySQL) BuscarPorUsuarioID(usuarioID string) (Colaborador, error) {
	var col Colaborador
	query := `SELECT ` + colunas + ` FROM tb_colaboradores
	          WHERE usuario_id = ? AND deletado_em IS NULL
	          ORDER BY criado_em DESC LIMIT 1`
	if err := r.db.Get(&col, query, usuarioID); err != nil {
		return Colaborador{}, fmt.Errorf("colaborador não encontrado para este usuário: %w", err)
	}
	return col, nil
}

// ListarPorEquipe retorna todos os colaboradores ativos de uma equipe, ordenados por nome
func (r *repositorioMySQL) ListarPorEquipe(equipeID string) ([]Colaborador, error) {
	var cols []Colaborador
	query := `SELECT ` + colunas + ` FROM tb_colaboradores WHERE equipe_id = ? AND deletado_em IS NULL ORDER BY nome ASC`
	if err := r.db.Select(&cols, query, equipeID); err != nil {
		return nil, fmt.Errorf("erro ao listar colaboradores da equipe: %w", err)
	}
	return cols, nil
}

// ListarPorOrganizacao retorna todos os colaboradores ativos de uma organização
func (r *repositorioMySQL) ListarPorOrganizacao(organizacaoID string) ([]Colaborador, error) {
	var cols []Colaborador
	query := `SELECT ` + colunas + ` FROM tb_colaboradores WHERE organizacao_id = ? AND deletado_em IS NULL ORDER BY nome ASC`
	if err := r.db.Select(&cols, query, organizacaoID); err != nil {
		return nil, fmt.Errorf("erro ao listar colaboradores da organização: %w", err)
	}
	return cols, nil
}

// Atualizar aplica as modificações em um colaborador existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(col Colaborador) (Colaborador, error) {
	agora := time.Now()
	col.AlteradoEm = &agora

	query := `
		UPDATE tb_colaboradores
		SET nome = :nome, email = :email, equipe_id = :equipe_id, usuario_id = :usuario_id,
		    template_id = :template_id, whatsapp = :whatsapp, data_nascimento = :data_nascimento,
		    alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, col); err != nil {
		return Colaborador{}, fmt.Errorf("erro ao atualizar colaborador: %w", err)
	}
	return r.BuscarPorId(col.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_colaboradores
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar colaborador: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("colaborador não encontrado ou já deletado")
	}
	return nil
}

// PertenceAoLider confere a posse da "Cadeia B": o colaborador é acessível se a equipe
// OU a organização dele tiver usuario_id igual ao do ATOR (o líder dono — igualdade, o
// caminho de sempre) OU se o gestor dono pertencer ao tenant do ATOR quando este é um RH
// (g.rh_id = ator). Uma query só (EXISTS), sem N+1.
//
// Igualdade primeiro / self-gating: passamos o id do ator nas DUAS pernas. Para um gestor
// solo ou líder comum, a perna `g.rh_id = ?` NUNCA casa (rh_id só recebe id de RH —
// invariante garantida na escrita), então o comportamento é IDÊNTICO ao anterior. Só um
// RH legítimo, dono daquele gestor, ativa a perna do tenant. Como tudo passa por aqui, o
// RH se propaga a classificacao, pdi, acompanhamento, convite, blocotema/tabuleiro (via
// PodeAcessar) e onebyone.Encerrar sem tocar nesses módulos.
func (r *repositorioMySQL) PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error) {
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM tb_colaboradores c
			JOIN tb_equipes e ON e.id = c.equipe_id
			JOIN tb_usuarios g ON g.id = e.usuario_id
			WHERE c.id = ? AND c.deletado_em IS NULL
			  AND e.deletado_em IS NULL
			  AND (e.usuario_id = ? OR g.rh_id = ?)
			UNION
			SELECT 1
			FROM tb_colaboradores c
			JOIN tb_organizacoes o ON o.id = c.organizacao_id
			JOIN tb_usuarios g ON g.id = o.usuario_id
			WHERE c.id = ? AND c.deletado_em IS NULL
			  AND o.deletado_em IS NULL
			  AND (o.usuario_id = ? OR g.rh_id = ?)
		)
	`
	if err := r.db.Get(&existe, query, colaboradorID, usuarioLiderID, usuarioLiderID, colaboradorID, usuarioLiderID, usuarioLiderID); err != nil {
		return false, fmt.Errorf("erro ao verificar posse do colaborador: %w", err)
	}
	return existe, nil
}

// ExisteEmailNoLider verifica se o líder já possui um liderado ATIVO com este
// e-mail dentro da sua estrutura (equipe OU organização — mesma Cadeia B do
// PertenceAoLider). O `excetoID` exclui um colaborador da checagem (no update,
// para não conflitar consigo mesmo); ao criar, passe "". Liderados deletados
// (deletado_em) são ignorados, então o e-mail pode ser reutilizado depois.
func (r *repositorioMySQL) ExisteEmailNoLider(email, usuarioLiderID, excetoID string) (bool, error) {
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM tb_colaboradores c
			JOIN tb_equipes e ON e.id = c.equipe_id
			WHERE c.email = ? AND c.id <> ? AND c.deletado_em IS NULL
			  AND e.usuario_id = ? AND e.deletado_em IS NULL
			UNION
			SELECT 1
			FROM tb_colaboradores c
			JOIN tb_organizacoes o ON o.id = c.organizacao_id
			WHERE c.email = ? AND c.id <> ? AND c.deletado_em IS NULL
			  AND o.usuario_id = ? AND o.deletado_em IS NULL
		)
	`
	if err := r.db.Get(&existe, query, email, excetoID, usuarioLiderID, email, excetoID, usuarioLiderID); err != nil {
		return false, fmt.Errorf("erro ao verificar e-mail do liderado: %w", err)
	}
	return existe, nil
}

// EmailEhDoLider confere se o e-mail informado é o da própria conta do gestor
// logado (tb_usuarios). Bloquear isso evita que um liderado seja criado com o
// e-mail do líder — o que, no aceite de convite, permitiria sequestrar a conta.
func (r *repositorioMySQL) EmailEhDoLider(email, usuarioLiderID string) (bool, error) {
	var existe bool
	query := `SELECT EXISTS (SELECT 1 FROM tb_usuarios WHERE id = ? AND email = ? AND deletado_em IS NULL)`
	if err := r.db.Get(&existe, query, usuarioLiderID, email); err != nil {
		return false, fmt.Errorf("erro ao verificar e-mail do gestor: %w", err)
	}
	return existe, nil
}

// EquipePertenceAoLider confere se a equipe é do líder dono (usuario_id) OU do tenant do
// ATOR quando este é um RH (rh_id do gestor dono = ator). Self-gating: para não-RH a perna
// rh_id nunca casa, mantendo o comportamento do gestor solo idêntico ao anterior.
func (r *repositorioMySQL) EquipePertenceAoLider(equipeID, usuarioLiderID string) (bool, error) {
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM tb_equipes e
			JOIN tb_usuarios g ON g.id = e.usuario_id
			WHERE e.id = ? AND e.deletado_em IS NULL
			  AND (e.usuario_id = ? OR g.rh_id = ?)
		)
	`
	if err := r.db.Get(&existe, query, equipeID, usuarioLiderID, usuarioLiderID); err != nil {
		return false, fmt.Errorf("erro ao verificar posse da equipe: %w", err)
	}
	return existe, nil
}

// OrganizacaoPertenceAoLider confere se a organização é do líder dono (usuario_id) OU do
// tenant do ATOR quando este é um RH (rh_id do gestor dono = ator). Self-gating, igual à
// EquipePertenceAoLider.
func (r *repositorioMySQL) OrganizacaoPertenceAoLider(organizacaoID, usuarioLiderID string) (bool, error) {
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM tb_organizacoes o
			JOIN tb_usuarios g ON g.id = o.usuario_id
			WHERE o.id = ? AND o.deletado_em IS NULL
			  AND (o.usuario_id = ? OR g.rh_id = ?)
		)
	`
	if err := r.db.Get(&existe, query, organizacaoID, usuarioLiderID, usuarioLiderID); err != nil {
		return false, fmt.Errorf("erro ao verificar posse da organização: %w", err)
	}
	return existe, nil
}

// VincularUsuario amarra a conta de usuário ao colaborador. Método dedicado ao
// fluxo de aceite de convite (o Atualizar geral NÃO mexe mais em usuario_id).
func (r *repositorioMySQL) VincularUsuario(colaboradorID, usuarioID string) error {
	query := `UPDATE tb_colaboradores SET usuario_id = ?, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`
	resultado, err := r.db.Exec(query, usuarioID, time.Now(), colaboradorID)
	if err != nil {
		return fmt.Errorf("erro ao vincular usuário ao colaborador: %w", err)
	}
	linhas, _ := resultado.RowsAffected()
	if linhas == 0 {
		return fmt.Errorf("colaborador não encontrado")
	}
	return nil
}

// DesvincularOutrasContas zera o usuario_id de todos os colaboradores ligados a esta conta,
// menos o atual. É o que garante o isolamento entre empresas: ao entrar numa nova empresa,
// o liderado deixa de ter acesso ao 1:1 das anteriores (o registro/histórico continua lá,
// só sem o vínculo de login da pessoa).
func (r *repositorioMySQL) DesvincularOutrasContas(usuarioID, excetoColaboradorID string) (int64, error) {
	query := `UPDATE tb_colaboradores SET usuario_id = NULL, alterado_em = ?
	          WHERE usuario_id = ? AND id != ?`
	resultado, err := r.db.Exec(query, time.Now(), usuarioID, excetoColaboradorID)
	if err != nil {
		return 0, fmt.Errorf("erro ao desvincular contas antigas: %w", err)
	}
	n, _ := resultado.RowsAffected()
	return n, nil
}

// Desligar marca o colaborador como inativo preenchendo desligado_em (preserva o registro).
func (r *repositorioMySQL) Desligar(id string, desligadoEm time.Time) error {
	query := `UPDATE tb_colaboradores SET desligado_em = ?, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`
	resultado, err := r.db.Exec(query, desligadoEm, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao desligar colaborador: %w", err)
	}
	linhas, _ := resultado.RowsAffected()
	if linhas == 0 {
		return fmt.Errorf("colaborador não encontrado")
	}
	return nil
}

// Reativar limpa desligado_em, voltando o colaborador a ativo.
func (r *repositorioMySQL) Reativar(id string) error {
	query := `UPDATE tb_colaboradores SET desligado_em = NULL, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`
	resultado, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao reativar colaborador: %w", err)
	}
	linhas, _ := resultado.RowsAffected()
	if linhas == 0 {
		return fmt.Errorf("colaborador não encontrado")
	}
	return nil
}

// AtualizarFoto persiste a chave do objeto S3 na coluna foto_key do colaborador informado.
func (r *repositorioMySQL) AtualizarFoto(id string, fotoKey string) error {
	query := `UPDATE tb_colaboradores SET foto_key = ?, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`

	resultado, err := r.db.Exec(query, fotoKey, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao atualizar foto do colaborador '%s': %w", id, err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("colaborador não encontrado")
	}
	return nil
}
