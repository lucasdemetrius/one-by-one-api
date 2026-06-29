// Pacote: internal/template
// Arquivo: entity.go
// Descrição: Define a entidade Template que representa um modelo de formulário
//            criado por um líder para estruturar os registros de one-on-one.
//            Mapeia a tabela tb_template.
// Autor: OneByOne API
// Criado em: 2025

package template

import "time"

// Template representa um modelo de formulário composto por blocos configuráveis,
// mapeando diretamente os campos da tabela tb_template.
type Template struct {
	// ID é o identificador único do template no formato UUID v4
	ID string `db:"id"`
	// UsuarioID é o UUID do líder proprietário deste template
	UsuarioID string `db:"usuario_id"`
	// Nome é o nome descritivo do template (ex.: "Reunião Mensal", "Feedback Trimestral")
	Nome string `db:"nome"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
}
