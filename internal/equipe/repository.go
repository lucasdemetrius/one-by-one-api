// Pacote: internal/equipe
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de equipes.
//            Toda interação com a tabela tb_equipes passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package equipe

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/autorizacao"
)

// Repositorio define o contrato de acesso ao banco para a entidade Equipe
type Repositorio interface {
	// Criar insere uma nova equipe e retorna o registro persistido
	Criar(equipe Equipe) (Equipe, error)
	// BuscarPorId retorna uma equipe ativa pelo UUID
	BuscarPorId(id string) (Equipe, error)
	// ListarPorUsuario retorna todas as equipes ativas de um líder
	ListarPorUsuario(usuarioID string) ([]Equipe, error)
	// ListarPorOrganizacao retorna todas as equipes ativas de uma organização
	ListarPorOrganizacao(organizacaoID string) ([]Equipe, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(equipe Equipe) (Equipe, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
	// AtualizarFoto persiste a chave S3 da foto no banco de dados
	AtualizarFoto(id string, fotoKey string) error
	// ExistePorNome diz se o líder JÁ tem uma equipe ativa com este nome (comparação
	// normalizada — minúsculas e sem espaços nas pontas). `excetoID` exclui uma
	// equipe da checagem (no update, para não conflitar consigo mesma); passe "" ao criar.
	ExistePorNome(usuarioID, nomeNormalizado, excetoID string) (bool, error)
	// GestorPertenceAoRH diz se o gestor dono (usuario_id) pertence ao tenant do RH
	// informado — fallback de posse para o papel RH (Cadeia A).
	GestorPertenceAoRH(gestorID, rhID string) (bool, error)
	// OrganizacaoPertenceAoAtor diz se a organização é do líder dono OU do tenant do RH
	// (usado para autorizar a listagem de equipes de uma organização).
	OrganizacaoPertenceAoAtor(organizacaoID, atorID string) (bool, error)
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de equipes
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// GestorPertenceAoRH delega à primitiva compartilhada: o gestor dono pertence ao tenant
// do RH informado? Fallback de posse para o papel RH (Cadeia A).
func (r *repositorioMySQL) GestorPertenceAoRH(gestorID, rhID string) (bool, error) {
	return autorizacao.GestorPertenceAoRH(r.db, gestorID, rhID)
}

// OrganizacaoPertenceAoAtor confere se a organização é do líder dono (usuario_id) OU do
// tenant do ator quando este é um RH (rh_id do gestor dono = ator). Self-gating.
func (r *repositorioMySQL) OrganizacaoPertenceAoAtor(organizacaoID, atorID string) (bool, error) {
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM tb_organizacoes o
			JOIN tb_usuarios g ON g.id = o.usuario_id
			WHERE o.id = ? AND o.deletado_em IS NULL
			  AND (o.usuario_id = ? OR g.rh_id = ?)
		)
	`
	if err := r.db.Get(&existe, query, organizacaoID, atorID, atorID); err != nil {
		return false, fmt.Errorf("erro ao verificar posse da organização: %w", err)
	}
	return existe, nil
}

// Criar insere uma nova equipe na tabela tb_equipes e retorna o registro completo
func (r *repositorioMySQL) Criar(equipe Equipe) (Equipe, error) {
	query := `
		INSERT INTO tb_equipes (id, usuario_id, organizacao_id, template_id, nome, criado_em)
		VALUES (:id, :usuario_id, :organizacao_id, :template_id, :nome, :criado_em)
	`
	if _, err := r.db.NamedExec(query, equipe); err != nil {
		return Equipe{}, fmt.Errorf("erro ao inserir equipe: %w", err)
	}
	return r.BuscarPorId(equipe.ID)
}

// ExistePorNome verifica se o líder já tem uma equipe ATIVA com o mesmo nome.
// A comparação é case-insensitive e ignora espaços nas pontas: o chamador passa
// o nome já normalizado (LOWER+TRIM em Go) e o SQL aplica LOWER(TRIM(...)) na
// coluna. `excetoID` exclui uma equipe (no update). Equipes deletadas não contam.
func (r *repositorioMySQL) ExistePorNome(usuarioID, nomeNormalizado, excetoID string) (bool, error) {
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM tb_equipes
			WHERE usuario_id = ? AND id <> ? AND deletado_em IS NULL
			  AND LOWER(TRIM(nome)) = ?
		)
	`
	if err := r.db.Get(&existe, query, usuarioID, excetoID, nomeNormalizado); err != nil {
		return false, fmt.Errorf("erro ao verificar nome da equipe: %w", err)
	}
	return existe, nil
}

// BuscarPorId retorna uma equipe ativa pelo UUID; erro se não existir ou estiver deletada
func (r *repositorioMySQL) BuscarPorId(id string) (Equipe, error) {
	var e Equipe
	query := `
		SELECT id, usuario_id, organizacao_id, template_id, nome, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_equipes
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&e, query, id); err != nil {
		return Equipe{}, fmt.Errorf("equipe não encontrada: %w", err)
	}
	return e, nil
}

// ListarPorUsuario retorna todas as equipes ativas pertencentes ao líder informado
func (r *repositorioMySQL) ListarPorUsuario(usuarioID string) ([]Equipe, error) {
	var equipes []Equipe
	query := `
		SELECT id, usuario_id, organizacao_id, template_id, nome, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_equipes
		WHERE usuario_id = ? AND deletado_em IS NULL
		ORDER BY nome ASC
	`
	if err := r.db.Select(&equipes, query, usuarioID); err != nil {
		return nil, fmt.Errorf("erro ao listar equipes: %w", err)
	}
	return equipes, nil
}

// ListarPorOrganizacao retorna todas as equipes ativas de uma organização específica
func (r *repositorioMySQL) ListarPorOrganizacao(organizacaoID string) ([]Equipe, error) {
	var equipes []Equipe
	query := `
		SELECT id, usuario_id, organizacao_id, template_id, nome, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_equipes
		WHERE organizacao_id = ? AND deletado_em IS NULL
		ORDER BY nome ASC
	`
	if err := r.db.Select(&equipes, query, organizacaoID); err != nil {
		return nil, fmt.Errorf("erro ao listar equipes da organização: %w", err)
	}
	return equipes, nil
}

// Atualizar aplica as modificações em uma equipe existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(equipe Equipe) (Equipe, error) {
	agora := time.Now()
	equipe.AlteradoEm = &agora

	query := `
		UPDATE tb_equipes
		SET nome = :nome, template_id = :template_id, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, equipe); err != nil {
		return Equipe{}, fmt.Errorf("erro ao atualizar equipe: %w", err)
	}
	return r.BuscarPorId(equipe.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_equipes
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar equipe: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("equipe não encontrada ou já deletada")
	}
	return nil
}

// AtualizarFoto persiste a chave do objeto S3 na coluna foto_key da equipe informada.
func (r *repositorioMySQL) AtualizarFoto(id string, fotoKey string) error {
	query := `UPDATE tb_equipes SET foto_key = ?, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`

	resultado, err := r.db.Exec(query, fotoKey, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao atualizar foto da equipe '%s': %w", id, err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("equipe não encontrada")
	}
	return nil
}
