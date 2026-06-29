// Pacote: internal/templatebloco
// Arquivo: entity.go
// Descrição: Define a entidade TemplateBloco que representa um campo/seção
//            dentro de um template de formulário. Mapeia tb_template_blocos.
//            Tipos suportados: TEXT, IMAGE, LIST, HIGHLIGHT.
// Autor: OneByOne API
// Criado em: 2025

package templatebloco

import "time"

// TemplateBloco representa um campo individual de um template de formulário,
// mapeando diretamente os campos da tabela tb_template_blocos.
type TemplateBloco struct {
	// ID é o identificador único do bloco no formato UUID v4
	ID string `db:"id"`
	// TemplateID é o UUID do template ao qual este bloco pertence
	TemplateID string `db:"template_id"`
	// Tipo define o tipo de conteúdo do bloco: TEXT, IMAGE, LIST ou HIGHLIGHT
	Tipo string `db:"tipo"`
	// Posicao define a ordem de exibição do bloco dentro do template (menor = primeiro)
	Posicao int `db:"posicao"`
	// Rotulo é o label/título exibido para o usuário ao preencher este campo
	Rotulo string `db:"rotulo"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
}
