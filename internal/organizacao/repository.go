// Pacote: internal/organizacao
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de organizações.
//            Toda interação com a tabela tb_organizacoes passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package organizacao

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/autorizacao"
)

// Repositorio define o contrato de acesso ao banco para a entidade Organizacao
type Repositorio interface {
	// Criar insere uma nova organização e retorna o registro persistido
	Criar(org Organizacao) (Organizacao, error)
	// BuscarPorId retorna uma organização ativa pelo UUID
	BuscarPorId(id string) (Organizacao, error)
	// ListarPorUsuario retorna todas as organizações ativas de um líder
	ListarPorUsuario(usuarioID string) ([]Organizacao, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(org Organizacao) (Organizacao, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
	// AtualizarFoto persiste a chave S3 da foto no banco de dados
	AtualizarFoto(id string, fotoKey string) error
	// GestorPertenceAoRH diz se o gestor dono (usuario_id) pertence ao tenant do RH
	// informado — fallback de posse para o papel RH (Cadeia A).
	GestorPertenceAoRH(gestorID, rhID string) (bool, error)
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de organizações
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// GestorPertenceAoRH delega à primitiva compartilhada: o gestor dono pertence ao tenant
// do RH informado? Fallback de posse para o papel RH (Cadeia A).
func (r *repositorioMySQL) GestorPertenceAoRH(gestorID, rhID string) (bool, error) {
	return autorizacao.GestorPertenceAoRH(r.db, gestorID, rhID)
}

// Criar insere uma nova organização na tabela tb_organizacoes e retorna o registro completo
func (r *repositorioMySQL) Criar(org Organizacao) (Organizacao, error) {
	query := `
		INSERT INTO tb_organizacoes (id, usuario_id, template_id, nome, criado_em)
		VALUES (:id, :usuario_id, :template_id, :nome, :criado_em)
	`
	if _, err := r.db.NamedExec(query, org); err != nil {
		return Organizacao{}, fmt.Errorf("erro ao inserir organização: %w", err)
	}
	return r.BuscarPorId(org.ID)
}

// BuscarPorId retorna uma organização ativa pelo UUID; erro se não existir ou estiver deletada
func (r *repositorioMySQL) BuscarPorId(id string) (Organizacao, error) {
	var org Organizacao
	query := `
		SELECT id, usuario_id, template_id, nome, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_organizacoes
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&org, query, id); err != nil {
		return Organizacao{}, fmt.Errorf("organização não encontrada: %w", err)
	}
	return org, nil
}

// ListarPorUsuario retorna todas as organizações ativas pertencentes ao líder informado
func (r *repositorioMySQL) ListarPorUsuario(usuarioID string) ([]Organizacao, error) {
	var orgs []Organizacao
	query := `
		SELECT id, usuario_id, template_id, nome, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_organizacoes
		WHERE usuario_id = ? AND deletado_em IS NULL
		ORDER BY nome ASC
	`
	if err := r.db.Select(&orgs, query, usuarioID); err != nil {
		return nil, fmt.Errorf("erro ao listar organizações: %w", err)
	}
	return orgs, nil
}

// Atualizar aplica as modificações em uma organização existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(org Organizacao) (Organizacao, error) {
	agora := time.Now()
	org.AlteradoEm = &agora

	query := `
		UPDATE tb_organizacoes
		SET nome = :nome, template_id = :template_id, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, org); err != nil {
		return Organizacao{}, fmt.Errorf("erro ao atualizar organização: %w", err)
	}
	return r.BuscarPorId(org.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_organizacoes
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar organização: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("organização não encontrada ou já deletada")
	}
	return nil
}

// AtualizarFoto persiste a chave do objeto S3 na coluna foto_key da organização informada.
func (r *repositorioMySQL) AtualizarFoto(id string, fotoKey string) error {
	query := `UPDATE tb_organizacoes SET foto_key = ?, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`

	resultado, err := r.db.Exec(query, fotoKey, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao atualizar foto da organização '%s': %w", id, err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("organização não encontrada")
	}
	return nil
}
