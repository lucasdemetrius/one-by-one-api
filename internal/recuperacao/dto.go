// Pacote: internal/recuperacao
// Arquivo: dto.go
// Descrição: Contratos HTTP do fluxo de recuperação de senha.
// Autor: OneByOne API
// Criado em: 2026

package recuperacao

// SolicitarDTO é o corpo de POST /auth/recuperar-senha (a pessoa informa o e-mail).
type SolicitarDTO struct {
	Email string `json:"email" binding:"required,email"`
}

// RedefinirDTO é o corpo de POST /recuperacoes/:token/redefinir.
type RedefinirDTO struct {
	Codigo    string `json:"codigo" binding:"required"`
	NovaSenha string `json:"nova_senha" binding:"required"`
}

// StatusTokenDTO informa ao front se o link ainda é válido (GET /recuperacoes/:token).
type StatusTokenDTO struct {
	Valido bool `json:"valido"`
}
