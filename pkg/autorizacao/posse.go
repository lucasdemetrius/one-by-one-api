// Pacote: pkg/autorizacao
// Arquivo: posse.go
// Descrição: Primitiva de autorização compartilhada para o papel RH. Responde se um
//            gestor (LIDER) pertence ao tenant de um RH. É a fronteira que escopa a
//            "gestão completa / visão total" do RH ao seu PRÓPRIO tenant, sem reabrir
//            IDOR. Usada pela Cadeia A (onebyone/template/organizacao/equipe/agendamento),
//            onde a posse é igualdade de usuario_id em Go. A Cadeia B (colaborador) faz o
//            mesmo escopo direto no SQL das suas funções de posse.
// Autor: OneByOne API
// Criado em: 2026

package autorizacao

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// GestorPertenceAoRH retorna true se o gestor cujo usuario_id é gestorID pertence ao
// tenant do RH cujo usuario_id é rhID — ou seja, se o gestor tem rh_id = rhID e a conta
// rhID é de fato um RH. É a fronteira do tenant: um RH só age/enxerga sobre os gestores
// que ele mesmo cadastrou (rh_id apontando para ele).
//
// INVARIANTE de segurança: tb_usuarios.rh_id só recebe o id de uma conta RH, preenchida
// exclusivamente no fluxo autenticado de criação de gestor pelo RH (rh_id é derivado do
// JWT, nunca do corpo). Ainda assim o JOIN exige role='RH' como defesa em profundidade,
// para jamais conceder acesso por um vínculo espúrio.
//
// rhID/gestorID vazios devolvem false de imediato. Por isso o chamador pode passar sempre
// o id do ator como rhID: para um não-RH, nenhum gestor terá rh_id igual ao dele, então
// a função é "self-gating" (só concede algo a um RH legítimo).
func GestorPertenceAoRH(db *sqlx.DB, gestorID, rhID string) (bool, error) {
	if gestorID == "" || rhID == "" {
		return false, nil
	}
	var existe bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM tb_usuarios g
			JOIN tb_usuarios r ON r.id = g.rh_id
			WHERE g.id = ? AND g.rh_id = ?
			  AND r.role = 'RH' AND r.deletado_em IS NULL
			  AND g.deletado_em IS NULL
		)
	`
	if err := db.Get(&existe, query, gestorID, rhID); err != nil {
		return false, fmt.Errorf("erro ao verificar tenant do RH: %w", err)
	}
	return existe, nil
}
