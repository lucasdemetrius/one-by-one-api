// Pacote: internal/convite
// Arquivo: dto.go
// Descrição: DTOs de entrada e saída do módulo de convite.
// Autor: OneByOne API
// Criado em: 2025

package convite

import "time"

// ConviteGeradoDTO é retornado ao gestor quando ele gera um convite.
// O código aparece em texto puro APENAS aqui — para o gestor compartilhar com
// o liderado. Depois disso só fica o hash no banco.
type ConviteGeradoDTO struct {
	// Token é o UUID do convite, usado na URL /convite/{token}
	Token string `json:"token"`
	// Codigo é a contra-senha em texto puro (mostrada só uma vez)
	Codigo string `json:"codigo"`
	// Link é o caminho relativo do convite no frontend
	Link string `json:"link"`
	// ExpiraEm é a validade do convite
	ExpiraEm time.Time `json:"expira_em"`
}

// ConvitePublicoDTO é o que o liderado vê ao abrir o link (sem autenticação).
// Nunca expõe o código.
type ConvitePublicoDTO struct {
	// Token é o UUID do convite
	Token string `json:"token"`
	// Valido indica se o convite ainda pode ser aceito (pendente e não expirado)
	Valido bool `json:"valido"`
	// ColaboradorNome é o nome de quem foi convidado
	ColaboradorNome string `json:"colaborador_nome"`
	// Email é o e-mail do colaborador (já preenchido no aceite)
	Email string `json:"email"`
}

// AceitarConviteDTO contém os dados enviados pelo liderado ao aceitar o convite.
type AceitarConviteDTO struct {
	// Codigo é a contra-senha recebida do gestor (obrigatório)
	Codigo string `json:"codigo" binding:"required"`
	// Senha é a senha de acesso que o liderado define (complexidade via pkg/senha)
	Senha string `json:"senha" binding:"required"`
}
