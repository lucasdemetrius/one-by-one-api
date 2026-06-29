// Pacote: internal/equipe
// Arquivo: entity.go
// Descrição: Define a entidade Equipe que representa um subgrupo dentro
//            de uma organização, mapeando a tabela tb_equipes.
// Autor: OneByOne API
// Criado em: 2025

package equipe

import "time"

// Equipe representa um grupo de colaboradores dentro de uma organização,
// mapeando diretamente os campos da tabela tb_equipes.
type Equipe struct {
	// ID é o identificador único da equipe no formato UUID v4
	ID string `db:"id"`
	// UsuarioID é o UUID do líder responsável por esta equipe
	UsuarioID string `db:"usuario_id"`
	// OrganizacaoID é o UUID da organização à qual esta equipe pertence
	OrganizacaoID string `db:"organizacao_id"`
	// TemplateID é o UUID do template padrão da equipe (sobrescreve o da organização se preenchido)
	TemplateID *string `db:"template_id"`
	// Nome é o nome da equipe
	Nome string `db:"nome"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
	// FotoKey é a chave do objeto no S3 (ex: equipes/uuid/foto.jpg); nil quando sem foto
	FotoKey *string `db:"foto_key"`
}
