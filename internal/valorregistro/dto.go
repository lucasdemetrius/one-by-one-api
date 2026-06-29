package valorregistro

import "time"

// CriarValorRegistroDTO contém os dados enviados pelo cliente para registrar
// a resposta de um bloco de formulário
type CriarValorRegistroDTO struct {
	// RegistroID é o UUID do registro de one-on-one ao qual este valor pertence (obrigatório)
	RegistroID string `json:"registro_id" binding:"required"`
	// BlocoID é o UUID do bloco do template que está sendo respondido (obrigatório)
	BlocoID string `json:"bloco_id" binding:"required"`
	// ValorTexto é a resposta em texto puro (usar para blocos TEXT e HIGHLIGHT)
	ValorTexto *string `json:"valor_texto" binding:"omitempty"`
	// ValorJSON é a resposta em formato JSON estruturado (usar para blocos LIST e IMAGE)
	ValorJSON interface{} `json:"valor_json" binding:"omitempty"`
}

// AtualizarValorRegistroDTO contém os campos alteráveis de um valor de registro
type AtualizarValorRegistroDTO struct {
	// ValorTexto é o novo conteúdo textual (opcional)
	ValorTexto *string `json:"valor_texto" binding:"omitempty"`
	// ValorJSON é o novo conteúdo JSON estruturado (opcional)
	ValorJSON interface{} `json:"valor_json" binding:"omitempty"`
}

// ValorRegistroRespostaDTO representa os dados do valor retornados pela API
type ValorRegistroRespostaDTO struct {
	// ID é o identificador único do valor
	ID string `json:"id"`
	// RegistroID é o UUID do registro de one-on-one
	RegistroID string `json:"registro_id"`
	// BlocoID é o UUID do bloco que foi respondido
	BlocoID string `json:"bloco_id"`
	// ValorTexto é o conteúdo textual da resposta (null para blocos JSON)
	ValorTexto *string `json:"valor_texto"`
	// ValorJSON é o conteúdo JSON da resposta (null para blocos de texto)
	ValorJSON interface{} `json:"valor_json"`
	// CriadoEm é a data e hora de criação
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação
	AlteradoEm *time.Time `json:"alterado_em"`
}
