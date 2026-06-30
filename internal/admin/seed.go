// Pacote: internal/admin
// Arquivo: seed.go
// Descrição: Garante a existência da conta de ADMIN da plataforma no boot da aplicação.
//            É idempotente e defensivo (loga e segue em caso de erro — nunca derruba o
//            boot). Chamado pelo rotas.go logo após conectar no banco.
// Autor: OneByOne API
// Criado em: 2026

package admin

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// GarantirContaAdmin assegura que a conta de ADMIN da plataforma exista e tenha o papel
// ADMIN. Regras (idempotentes):
//
//   - e-mail vazio                 → não faz nada.
//   - conta já existe (qualquer role) → PROMOVE a ADMIN e zera o rh_id (ADMIN é global,
//     não pertence a tenant nenhum).
//   - conta NÃO existe e há senha  → CRIA a conta como ADMIN (hash bcrypt custo 12).
//   - conta NÃO existe e sem senha → não cria (sem senha padrão, por segurança) e loga.
//
// Importante: o papel 'ADMIN' só é aceito pelo banco após a migration 023. Se ela ainda
// não tiver sido aplicada, o UPDATE/INSERT falha — a função apenas LOGA e segue (o app
// sobe normalmente). O `email` deve vir já normalizado (config.AdminEmail já normaliza).
//
// Anti-escalonamento: o e-mail de admin é RESERVADO nas vias públicas de criação/edição de
// conta (usuario.emailReservado), então ninguém consegue "sentar" nesse e-mail antes do
// operador. Assim, o ramo de PROMOÇÃO abaixo só alcança a própria conta de admin (criada
// por este seed ou pré-existente do operador), nunca uma conta de terceiro recém-cadastrada.
func GarantirContaAdmin(db *sqlx.DB, email, senha string) {
	if email == "" {
		return
	}

	// Opt-in EXPLÍCITO: sem ADMIN_SENHA o seed NÃO faz nada — nem cria, nem promove. Isso
	// evita promover automaticamente, num deploy qualquer, uma conta pré-existente que por
	// acaso tenha o ADMIN_EMAIL (que tem default conhecido). Para ligar o admin, o operador
	// define ADMIN_SENHA de propósito. (A reserva de e-mail já barra cadastros NOVOS; isto
	// fecha o caso de dado pré-existente.)
	if senha == "" {
		log.Printf("[admin] ADMIN_SENHA não definido — seed do admin DESLIGADO (nenhuma conta criada ou promovida)")
		return
	}

	var existente struct {
		ID   string `db:"id"`
		Role string `db:"role"`
	}
	err := db.Get(&existente,
		`SELECT id, role FROM tb_usuarios WHERE email = ? AND deletado_em IS NULL`, email)

	switch {
	case err == nil:
		// Conta já existe → garante o papel ADMIN (promove) e desfaz vínculo de tenant.
		if existente.Role == "ADMIN" {
			log.Printf("[admin] conta admin %s já está com papel ADMIN", email)
			return
		}
		if _, e := db.Exec(`UPDATE tb_usuarios SET role = 'ADMIN', rh_id = NULL WHERE id = ?`, existente.ID); e != nil {
			log.Printf("[admin] falha ao promover %s a ADMIN (migration 023 aplicada?): %v", email, e)
			return
		}
		// Aviso PROEMINENTE: estamos promovendo uma conta que JÁ existia. Se você não criou
		// essa conta, alguém pode ter se cadastrado antes com esse e-mail — revise.
		log.Printf("[admin] ATENÇÃO: conta PRÉ-EXISTENTE %s foi PROMOVIDA a ADMIN. Confirme que é a sua conta de administrador (e em produção use um ADMIN_EMAIL não-óbvio).", email)

	case errors.Is(err, sql.ErrNoRows):
		// Conta não existe → cria como ADMIN (ADMIN_SENHA já garantido != "" pelo opt-in acima).
		hash, e := bcrypt.GenerateFromPassword([]byte(senha), 12)
		if e != nil {
			log.Printf("[admin] falha ao gerar hash da senha admin: %v", e)
			return
		}
		_, e = db.Exec(
			`INSERT INTO tb_usuarios (id, nome, email, password, role, criado_em) VALUES (?, ?, ?, ?, 'ADMIN', ?)`,
			uuid.New().String(), "Administrador", email, string(hash), time.Now(),
		)
		if e != nil {
			log.Printf("[admin] falha ao criar conta admin %s (migration 023 aplicada?): %v", email, e)
			return
		}
		log.Printf("[admin] conta admin %s criada com papel ADMIN", email)

	default:
		log.Printf("[admin] erro ao verificar a conta admin %s: %v", email, err)
	}
}
