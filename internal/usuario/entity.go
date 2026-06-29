// Pacote: internal/usuario
// Arquivo: entity.go
// Descrição: Define a entidade Usuario que representa um registro da tabela
//            tb_usuarios. As tags `db` mapeiam os campos para as colunas do banco.
// Autor: OneByOne API
// Criado em: 2025

package usuario

import "time"

// Usuario representa um usuário cadastrado no sistema, seja ele um líder
// ou colaborador. Mapeia diretamente a tabela tb_usuarios do banco de dados.
type Usuario struct {
	// ID é o identificador único do usuário no formato UUID v4
	ID string `db:"id"`
	// Nome é o nome completo do usuário
	Nome string `db:"nome"`
	// Email é o endereço de e-mail único utilizado para login
	Email string `db:"email"`
	// Password é a senha armazenada como hash bcrypt (nunca em texto puro)
	Password string `db:"password"`
	// TentativasLogin conta as falhas de login consecutivas (lockout após N falhas)
	TentativasLogin int `db:"tentativas_login"`
	// BloqueadoAte: se preenchido e no futuro, a conta está temporariamente bloqueada
	BloqueadoAte *time.Time `db:"bloqueado_ate"`
	// TokenVersion: versão atual do token do usuário (revogação ao trocar senha/excluir conta)
	TokenVersion int `db:"token_version"`
	// Role define o papel do usuário no sistema: LIDER, COLABORADOR ou RH
	Role string `db:"role"`
	// RhID, em um GESTOR (LIDER), aponta para o usuario_id do RH dono dele — é o
	// vínculo que define "quais gestores um RH enxerga". Nil significa: gestor solo
	// (sem RH acima, igual ao comportamento de hoje) OU o próprio RH (que é raiz).
	// As queries que leem/escrevem esta coluna dependem da migration 017.
	RhID *string `db:"rh_id"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
	// FotoKey é a chave do objeto no S3 (ex: usuarios/uuid/foto.jpg); nil quando sem foto
	FotoKey *string `db:"foto_key"`
}
