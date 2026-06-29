// Pacote: internal/organizacao
// Arquivo: entity.go
// Descrição: Define a entidade Organizacao que representa um registro da
//            tabela tb_organizacoes. Agrupa equipes e colaboradores sob
//            um mesmo contexto gerenciado por um líder.
// Autor: OneByOne API
// Criado em: 2025

package organizacao

import "time"

// Organizacao representa uma organização cadastrada no sistema,
// mapeando diretamente os campos da tabela tb_organizacoes.
type Organizacao struct {
	// ID é o identificador único da organização no formato UUID v4
	ID string `db:"id"`
	// UsuarioID é o UUID do líder proprietário desta organização
	UsuarioID string `db:"usuario_id"`
	// TemplateID é o UUID do template padrão da organização (pode ser nulo)
	TemplateID *string `db:"template_id"`
	// Nome é o nome da organização
	Nome string `db:"nome"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
	// FotoKey é a chave do objeto no S3 (ex: organizacoes/uuid/foto.jpg); nil quando sem foto
	FotoKey *string `db:"foto_key"`
}
