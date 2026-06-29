// Pacote: internal/ia
// Arquivo: repository.go
// Descrição: Persistência da configuração de IA do usuário (provedor + chave
//            cifrada) na própria tb_usuarios. Só I/O de banco.
// Autor: OneByOne API
// Criado em: 2026

package ia

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o acesso à config de IA do usuário.
type Repositorio interface {
	// ObterConfig devolve o provedor e a chave CIFRADA (ambos podem ser nil).
	ObterConfig(usuarioID string) (provedor *string, chaveCifrada *string, err error)
	// SalvarConfig grava o provedor e a chave cifrada (chave nil = não altera a chave).
	SalvarConfig(usuarioID, provedor string, chaveCifrada *string) error
	// ObterConfigEfetiva devolve a config EM VIGOR: a própria do usuário se completa;
	// senão a do RH dono (rh_id) — é assim que os gestores herdam a IA do RH.
	// herdadaDoRH=true quando caiu na config do RH.
	ObterConfigEfetiva(usuarioID string) (provedor *string, chaveCifrada *string, herdadaDoRH bool, err error)
}

type repositorioMySQL struct {
	db *sqlx.DB
}

// NovoRepositorio cria o repositório de config de IA.
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

func (r *repositorioMySQL) ObterConfig(usuarioID string) (*string, *string, error) {
	var linha struct {
		Provedor *string `db:"ia_provedor"`
		Chave    *string `db:"ia_chave_cifrada"`
	}
	query := `SELECT ia_provedor, ia_chave_cifrada FROM tb_usuarios WHERE id = ? AND deletado_em IS NULL`
	if err := r.db.Get(&linha, query, usuarioID); err != nil {
		return nil, nil, fmt.Errorf("erro ao obter config de IA: %w", err)
	}
	return linha.Provedor, linha.Chave, nil
}

func (r *repositorioMySQL) ObterConfigEfetiva(usuarioID string) (*string, *string, bool, error) {
	var linha struct {
		UProv  *string `db:"u_prov"`
		UChave *string `db:"u_chave"`
		RProv  *string `db:"rh_prov"`
		RChave *string `db:"rh_chave"`
	}
	query := `
		SELECT u.ia_provedor AS u_prov, u.ia_chave_cifrada AS u_chave,
		       rh.ia_provedor AS rh_prov, rh.ia_chave_cifrada AS rh_chave
		FROM tb_usuarios u
		LEFT JOIN tb_usuarios rh ON rh.id = u.rh_id AND rh.deletado_em IS NULL
		WHERE u.id = ? AND u.deletado_em IS NULL`
	if err := r.db.Get(&linha, query, usuarioID); err != nil {
		return nil, nil, false, fmt.Errorf("erro ao obter config de IA: %w", err)
	}
	// Config própria tem prioridade (precisa de provedor E chave completos).
	if linha.UProv != nil && *linha.UProv != "" && linha.UChave != nil && *linha.UChave != "" {
		return linha.UProv, linha.UChave, false, nil
	}
	// Senão, herda a do RH (se houver e estiver completa).
	if linha.RProv != nil && *linha.RProv != "" && linha.RChave != nil && *linha.RChave != "" {
		return linha.RProv, linha.RChave, true, nil
	}
	// Nenhuma config completa em vigor.
	return linha.UProv, linha.UChave, false, nil
}

func (r *repositorioMySQL) SalvarConfig(usuarioID, provedor string, chaveCifrada *string) error {
	// Se chaveCifrada for nil, mantém a chave atual (o gestor só trocou o provedor).
	if chaveCifrada != nil {
		_, err := r.db.Exec(
			`UPDATE tb_usuarios SET ia_provedor = ?, ia_chave_cifrada = ? WHERE id = ?`,
			provedor, *chaveCifrada, usuarioID,
		)
		if err != nil {
			return fmt.Errorf("erro ao salvar config de IA: %w", err)
		}
		return nil
	}
	_, err := r.db.Exec(`UPDATE tb_usuarios SET ia_provedor = ? WHERE id = ?`, provedor, usuarioID)
	if err != nil {
		return fmt.Errorf("erro ao salvar provedor de IA: %w", err)
	}
	return nil
}
