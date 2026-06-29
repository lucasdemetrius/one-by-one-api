// Pacote: internal/registroonebyone
// Arquivo: entity.go
// Descrição: Define a entidade RegistroOneByOne que representa o formulário
//            preenchido durante uma reunião one-on-one. Mapeia tb_registros_onebyone.
// Autor: OneByOne API
// Criado em: 2025

package registroonebyone

import "time"

// RegistroOneByOne representa o formulário preenchido durante uma reunião one-on-one,
// vinculando-a ao template utilizado no momento do preenchimento.
// Mapeia diretamente os campos da tabela tb_registros_onebyone.
type RegistroOneByOne struct {
	// ID é o identificador único do registro no formato UUID v4
	ID string `db:"id"`
	// OneByOneID é o UUID da reunião one-on-one à qual este registro pertence
	OneByOneID string `db:"oneaone_id"`
	// TemplateID é o UUID do template que foi utilizado para estruturar este registro
	TemplateID string `db:"template_id"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
}
