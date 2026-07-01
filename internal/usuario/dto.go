// Pacote: internal/usuario
// Arquivo: dto.go
// Descrição: Define os DTOs (Data Transfer Objects) de entrada e saída para
//            as operações do módulo de usuário. Separa o modelo de banco
//            do contrato da API HTTP.
// Autor: OneByOne API
// Criado em: 2025

package usuario

import "time"

// CriarUsuarioDTO contém os dados enviados pelo cliente para criar um novo usuário
type CriarUsuarioDTO struct {
	// Nome é o nome completo do usuário (obrigatório, 2 a 100 caracteres)
	Nome string `json:"nome" binding:"required,min=2,max=100"`
	// Email é o endereço de e-mail do usuário (obrigatório, formato válido)
	Email string `json:"email" binding:"required,email,max=150"`
	// Password é a senha em texto puro (complexidade validada por pkg/senha: ≥ 8, com
	// maiúscula, minúscula e número). max=100 é só um teto de sanidade.
	Password string `json:"password" binding:"required,max=100"`
	// Role define o papel do usuário: LIDER, COLABORADOR ou RH (padrão: COLABORADOR).
	// Observação de segurança: NÃO existe campo rh_id aqui de propósito. O vínculo
	// gestor→RH nunca vem do corpo da requisição — é sempre derivado no servidor
	// (RH que se auto-cadastra nasce como raiz, rh_id nulo). Isso evita que alguém
	// se atrele a um tenant existente e veja dados alheios (escalonamento de privilégio).
	Role string `json:"role" binding:"omitempty,oneof=LIDER COLABORADOR RH"`
}

// AtualizarUsuarioDTO contém os campos que podem ser alterados em um usuário existente.
// Todos os campos são opcionais — apenas os informados serão atualizados.
// Observação de segurança: NÃO existe campo Role aqui de propósito. O papel do usuário
// não é editável por auto-serviço — senão um PUT permitiria escalonamento
// COLABORADOR→LIDER/RH. Mudança de papel só por fluxos controlados (cadastro/convite/RH).
type AtualizarUsuarioDTO struct {
	// Nome é o novo nome completo do usuário (opcional)
	Nome string `json:"nome" binding:"omitempty,min=2,max=100"`
	// Email é o novo endereço de e-mail do usuário (opcional)
	Email string `json:"email" binding:"omitempty,email,max=150"`
}

// LoginDTO contém as credenciais enviadas pelo cliente para autenticação
type LoginDTO struct {
	// Email é o e-mail do usuário cadastrado (obrigatório)
	Email string `json:"email" binding:"required,email"`
	// Password é a senha em texto puro para validação contra o hash armazenado (obrigatório)
	Password string `json:"password" binding:"required"`
}

// LoginGoogleDTO recebe o "credential" (ID token JWT) devolvido pelo Google Identity
// Services no front. O backend valida esse token contra o GOOGLE_CLIENT_ID e faz o
// login (conta existente) ou o cadastro (conta nova de Gestor) por trás.
type LoginGoogleDTO struct {
	// Credential é o ID token JWT emitido pelo Google (campo "credential" do GIS).
	Credential string `json:"credential" binding:"required"`
}

// UsuarioRespostaDTO representa os dados do usuário retornados pela API.
// Nunca expõe campos sensíveis como a senha ou metadados de soft delete.
type UsuarioRespostaDTO struct {
	// ID é o identificador único do usuário
	ID string `json:"id"`
	// Nome é o nome completo do usuário
	Nome string `json:"nome"`
	// Email é o endereço de e-mail do usuário
	Email string `json:"email"`
	// Role é o papel do usuário no sistema: LIDER ou COLABORADOR
	Role string `json:"role"`
	// CriadoEm é a data e hora de criação do registro
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação (null se nunca alterado)
	AlteradoEm *time.Time `json:"alterado_em"`
	// FotoURL é a URL presignada temporária da foto de perfil (null se sem foto; expira em 2h)
	FotoURL *string `json:"foto_url"`
}

// LoginRespostaDTO é retornado após uma autenticação bem-sucedida.
// Contém o token JWT e os dados básicos do usuário autenticado.
type LoginRespostaDTO struct {
	// Token é o JWT assinado para uso nas próximas requisições protegidas
	Token string `json:"token"`
	// Usuario contém os dados do usuário autenticado
	Usuario UsuarioRespostaDTO `json:"usuario"`
}
