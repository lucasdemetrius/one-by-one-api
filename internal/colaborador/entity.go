// Pacote: internal/colaborador
// Arquivo: entity.go
// Descrição: Define a entidade Colaborador que representa um membro de equipe,
//            mapeando a tabela tb_colaboradores. O colaborador pode ou não
//            possuir conta de acesso ao sistema (usuario_id é opcional).
// Autor: OneByOne API
// Criado em: 2025

package colaborador

import "time"

// Colaborador representa um membro de uma equipe dentro de uma organização,
// mapeando diretamente os campos da tabela tb_colaboradores.
type Colaborador struct {
	// ID é o identificador único do colaborador no formato UUID v4
	ID string `db:"id"`
	// UsuarioID é o UUID da conta de sistema do colaborador (pode ser nulo — colaborador sem login)
	UsuarioID *string `db:"usuario_id"`
	// OrganizacaoID é o UUID da organização à qual o colaborador pertence
	OrganizacaoID string `db:"organizacao_id"`
	// EquipeID é o UUID da equipe à qual o colaborador pertence
	EquipeID string `db:"equipe_id"`
	// TemplateID é o UUID do template exclusivo deste colaborador (prioridade máxima na herança)
	TemplateID *string `db:"template_id"`
	// Nome é o nome completo do colaborador
	Nome string `db:"nome"`
	// Email é o endereço de e-mail do colaborador (não necessariamente usado para login)
	Email string `db:"email"`
	// Whatsapp é o número de WhatsApp do colaborador com DDD (pode ser nulo)
	Whatsapp *string `db:"whatsapp"`
	// DataNascimento é a data de nascimento do colaborador (pode ser nula)
	DataNascimento *time.Time `db:"data_nascimento"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
	// FotoKey é a chave do objeto no S3 (ex: colaboradores/uuid/foto.jpg); nil quando sem foto
	FotoKey *string `db:"foto_key"`
	// DesligadoEm é a data de desligamento (saída da empresa/equipe). nil = ATIVO.
	// Diferente de DeletadoEm: o registro é preservado para histórico e linha do tempo.
	DesligadoEm *time.Time `db:"desligado_em"`
}
