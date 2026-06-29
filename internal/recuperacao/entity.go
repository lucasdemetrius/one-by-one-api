// Pacote: internal/recuperacao
// Arquivo: entity.go
// Descrição: Entidade Recuperacao — um pedido de "esqueci minha senha". Liga um
//            usuário a um link (UUID = token) + código (contra-senha). Mapeia a
//            tabela tb_recuperacoes_senha. Mesmo padrão dos convites.
// Autor: OneByOne API
// Criado em: 2026

package recuperacao

import "time"

// Status possíveis de uma recuperação.
const (
	StatusPendente = "PENDENTE"
	StatusUsado    = "USADO"
)

// Recuperacao liga um usuário a um token (link) + código, com validade e uso único.
type Recuperacao struct {
	ID         string     `db:"id"`          // UUID = token usado no link /redefinir-senha/{id}
	UsuarioID  string     `db:"usuario_id"`  // dono da conta
	CodigoHash string     `db:"codigo_hash"` // hash bcrypt do código (contra-senha)
	Status     string     `db:"status"`      // PENDENTE | USADO
	Tentativas int        `db:"tentativas"`  // códigos errados (token é invalidado ao atingir o limite)
	ExpiraEm   time.Time  `db:"expira_em"`   // validade (1 hora)
	CriadoEm   time.Time  `db:"criado_em"`
	UsadoEm    *time.Time `db:"usado_em"`
}
