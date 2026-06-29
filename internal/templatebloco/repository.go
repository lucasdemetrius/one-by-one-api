// Pacote: internal/templatebloco
// Arquivo: repository.go
// Descrição: Define a interface e a implementação MySQL do repositório de blocos de template.
//            Toda interação com a tabela tb_template_blocos passa por aqui.
// Autor: OneByOne API
// Criado em: 2025

package templatebloco

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o contrato de acesso ao banco para a entidade TemplateBloco
type Repositorio interface {
	// Criar insere um novo bloco e retorna o registro persistido
	Criar(bloco TemplateBloco) (TemplateBloco, error)
	// BuscarPorId retorna um bloco ativo pelo UUID
	BuscarPorId(id string) (TemplateBloco, error)
	// ListarPorTemplate retorna todos os blocos ativos de um template, ordenados por posicao
	ListarPorTemplate(templateID string) ([]TemplateBloco, error)
	// Atualizar aplica as modificações e retorna o registro atualizado
	Atualizar(bloco TemplateBloco) (TemplateBloco, error)
	// DeletarSoft realiza a exclusão lógica preenchendo deletado_em e deletado_por
	DeletarSoft(id string, deletadoPor string) error
}

// repositorioMySQL é a implementação concreta do Repositorio para MySQL
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de blocos de template
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// Criar insere um novo bloco na tabela tb_template_blocos e retorna o registro completo
func (r *repositorioMySQL) Criar(bloco TemplateBloco) (TemplateBloco, error) {
	query := `
		INSERT INTO tb_template_blocos (id, template_id, tipo, posicao, rotulo, criado_em)
		VALUES (:id, :template_id, :tipo, :posicao, :rotulo, :criado_em)
	`
	if _, err := r.db.NamedExec(query, bloco); err != nil {
		return TemplateBloco{}, fmt.Errorf("erro ao inserir bloco de template: %w", err)
	}
	return r.BuscarPorId(bloco.ID)
}

// BuscarPorId retorna um bloco ativo pelo UUID; erro se não existir ou estiver deletado
func (r *repositorioMySQL) BuscarPorId(id string) (TemplateBloco, error) {
	var bloco TemplateBloco
	query := `
		SELECT id, template_id, tipo, posicao, rotulo, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_template_blocos
		WHERE id = ? AND deletado_em IS NULL
	`
	if err := r.db.Get(&bloco, query, id); err != nil {
		return TemplateBloco{}, fmt.Errorf("bloco de template não encontrado: %w", err)
	}
	return bloco, nil
}

// ListarPorTemplate retorna todos os blocos ativos de um template, ordenados por posicao ASC.
// Essa ordenação garante que os blocos sejam exibidos na ordem correta no formulário.
func (r *repositorioMySQL) ListarPorTemplate(templateID string) ([]TemplateBloco, error) {
	var blocos []TemplateBloco
	query := `
		SELECT id, template_id, tipo, posicao, rotulo, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_template_blocos
		WHERE template_id = ? AND deletado_em IS NULL
		ORDER BY posicao ASC
	`
	if err := r.db.Select(&blocos, query, templateID); err != nil {
		return nil, fmt.Errorf("erro ao listar blocos do template: %w", err)
	}
	return blocos, nil
}

// Atualizar aplica as modificações em um bloco existente e retorna o registro atualizado
func (r *repositorioMySQL) Atualizar(bloco TemplateBloco) (TemplateBloco, error) {
	agora := time.Now()
	bloco.AlteradoEm = &agora

	query := `
		UPDATE tb_template_blocos
		SET tipo = :tipo, posicao = :posicao, rotulo = :rotulo, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`
	if _, err := r.db.NamedExec(query, bloco); err != nil {
		return TemplateBloco{}, fmt.Errorf("erro ao atualizar bloco de template: %w", err)
	}
	return r.BuscarPorId(bloco.ID)
}

// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()
	query := `
		UPDATE tb_template_blocos
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`
	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar bloco de template: %w", err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("bloco de template não encontrado ou já deletado")
	}
	return nil
}
