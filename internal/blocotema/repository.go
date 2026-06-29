// Pacote: internal/blocotema
// Arquivo: repository.go
// Descrição: Interface Repositorio e implementação MySQL (sqlx) para a tabela
//            tb_blocos_tema. Apenas I/O de banco.
// Autor: OneByOne API
// Criado em: 2025

package blocotema

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repositorio define as operações de persistência dos blocos de tema.
type Repositorio interface {
	// Listar retorna os blocos de um tema de um colaborador, em ordem
	Listar(colaboradorID, tema string) ([]BlocoTema, error)
	// ListarTodos retorna TODOS os blocos do colaborador (todos os temas) — para a IA
	ListarTodos(colaboradorID string) ([]BlocoTema, error)
	// Criar insere um bloco e retorna o registro persistido
	Criar(b BlocoTema) (BlocoTema, error)
	// BuscarPorId localiza um bloco pelo UUID
	BuscarPorId(id string) (BlocoTema, error)
	// Deletar remove um bloco definitivamente
	Deletar(id string) error
	// ProximaOrdem devolve a próxima posição (MAX(ordem)+1) para o tema
	ProximaOrdem(colaboradorID, tema string) (int, error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de blocos de tema.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) Listar(colaboradorID, tema string) ([]BlocoTema, error) {
	var blocos []BlocoTema
	query := `SELECT id, colaborador_id, tema, tipo, texto, url, imagem_key,
	                 data_inicio, data_fim, ordem, criado_em
	          FROM tb_blocos_tema
	          WHERE colaborador_id = ? AND tema = ?
	          ORDER BY ordem ASC, criado_em ASC`
	if err := r.db.Select(&blocos, query, colaboradorID, tema); err != nil {
		return nil, fmt.Errorf("erro ao listar blocos: %w", err)
	}
	return blocos, nil
}

// ListarTodos traz todos os blocos do colaborador (todos os temas), agrupáveis por tema.
func (r *repositorioMySQL) ListarTodos(colaboradorID string) ([]BlocoTema, error) {
	var blocos []BlocoTema
	query := `SELECT id, colaborador_id, tema, tipo, texto, url, imagem_key,
	                 data_inicio, data_fim, ordem, criado_em
	          FROM tb_blocos_tema
	          WHERE colaborador_id = ?
	          ORDER BY tema ASC, ordem ASC, criado_em ASC`
	if err := r.db.Select(&blocos, query, colaboradorID); err != nil {
		return nil, fmt.Errorf("erro ao listar conteúdo do colaborador: %w", err)
	}
	return blocos, nil
}

func (r *repositorioMySQL) Criar(b BlocoTema) (BlocoTema, error) {
	query := `INSERT INTO tb_blocos_tema
	          (id, colaborador_id, tema, tipo, texto, url, imagem_key, data_inicio, data_fim, ordem, criado_em)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.Exec(query, b.ID, b.ColaboradorID, b.Tema, b.Tipo, b.Texto, b.URL,
		b.ImagemKey, b.DataInicio, b.DataFim, b.Ordem, b.CriadoEm); err != nil {
		return BlocoTema{}, fmt.Errorf("erro ao inserir bloco: %w", err)
	}
	return b, nil
}

func (r *repositorioMySQL) BuscarPorId(id string) (BlocoTema, error) {
	var b BlocoTema
	query := `SELECT id, colaborador_id, tema, tipo, texto, url, imagem_key,
	                 data_inicio, data_fim, ordem, criado_em
	          FROM tb_blocos_tema WHERE id = ?`
	if err := r.db.Get(&b, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BlocoTema{}, fmt.Errorf("bloco não encontrado")
		}
		return BlocoTema{}, fmt.Errorf("erro ao buscar bloco: %w", err)
	}
	return b, nil
}

func (r *repositorioMySQL) Deletar(id string) error {
	if _, err := r.db.Exec(`DELETE FROM tb_blocos_tema WHERE id = ?`, id); err != nil {
		return fmt.Errorf("erro ao deletar bloco: %w", err)
	}
	return nil
}

func (r *repositorioMySQL) ProximaOrdem(colaboradorID, tema string) (int, error) {
	var max sql.NullInt64
	query := `SELECT MAX(ordem) FROM tb_blocos_tema WHERE colaborador_id = ? AND tema = ?`
	if err := r.db.Get(&max, query, colaboradorID, tema); err != nil {
		return 0, fmt.Errorf("erro ao calcular ordem: %w", err)
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64) + 1, nil
}
