// Pacote: internal/valorregistro
// Arquivo: entity.go
// Descrição: Define a entidade ValorRegistro que representa a resposta
//            de um bloco específico dentro de um registro de one-on-one.
//            Mapeia a tabela tb_valores_registro.
// Autor: OneByOne API
// Criado em: 2025

package valorregistro

import "time"

// ValorRegistro representa a resposta preenchida para um bloco de formulário
// dentro de um registro de one-on-one. Suporta conteúdo textual e JSON estruturado.
// Mapeia diretamente os campos da tabela tb_valores_registro.
type ValorRegistro struct {
	// ID é o identificador único do valor no formato UUID v4
	ID string `db:"id"`
	// RegistroID é o UUID do registro de one-on-one ao qual este valor pertence
	RegistroID string `db:"registro_id"`
	// BlocoID é o UUID do bloco de template que foi respondido
	BlocoID string `db:"bloco_id"`
	// ValorTexto armazena a resposta em texto puro (usado em blocos TEXT e HIGHLIGHT)
	ValorTexto *string `db:"valor_texto"`
	// ValorJSON armazena a resposta em formato JSON estruturado (usado em blocos LIST e IMAGE)
	ValorJSON []byte `db:"valor_json"`
	// CriadoEm é o timestamp de criação do registro
	CriadoEm time.Time `db:"criado_em"`
	// AlteradoEm é o timestamp da última modificação (nil se nunca alterado)
	AlteradoEm *time.Time `db:"alterado_em"`
	// DeletadoEm é o timestamp da exclusão lógica (nil significa registro ativo)
	DeletadoEm *time.Time `db:"deletado_em"`
	// DeletadoPor armazena o ID do usuário responsável pela exclusão lógica
	DeletadoPor *string `db:"deletado_por"`
}
