// Pacote: internal/registroonebyone
// Arquivo: dto.go
// Descrição: Define os DTOs de entrada e saída do módulo de registro de one-on-one.
//            O template utilizado é resolvido automaticamente pelo UseCase
//            seguindo a regra de herança definida no módulo oneaone.
// Autor: OneByOne API
// Criado em: 2025

package registroonebyone

import "time"

// CriarRegistroOneByOneDTO contém os dados necessários para abrir um registro de reunião.
// O template é resolvido automaticamente — não é necessário informá-lo.
type CriarRegistroOneByOneDTO struct {
	// OneByOneID é o UUID da reunião one-on-one que está sendo registrada (obrigatório)
	OneByOneID string `json:"onebyone_id" binding:"required"`
}

// RegistroOneByOneRespostaDTO representa os dados do registro retornados pela API
type RegistroOneByOneRespostaDTO struct {
	// ID é o identificador único do registro
	ID string `json:"id"`
	// OneByOneID é o UUID da reunião one-on-one
	OneByOneID string `json:"onebyone_id"`
	// TemplateID é o UUID do template que foi aplicado automaticamente pela herança
	TemplateID string `json:"template_id"`
	// CriadoEm é a data e hora de criação do registro
	CriadoEm time.Time `json:"criado_em"`
	// AlteradoEm é a data e hora da última modificação
	AlteradoEm *time.Time `json:"alterado_em"`
}
