// Pacote: internal/ia
// Arquivo: dto.go
// Descrição: Contratos HTTP do módulo de IA. A chave de API NUNCA volta para o
//            cliente — só o provedor e um indicador de "tem chave".
// Autor: OneByOne API
// Criado em: 2026

package ia

// ConfigIARespostaDTO é o que o cliente vê sobre a IA do gestor (sem a chave).
type ConfigIARespostaDTO struct {
	Provedor string `json:"provedor"` // "" se não configurado
	TemChave bool   `json:"tem_chave"`
	// HerdadaDoRH=true quando o usuário não tem IA própria, mas usa a que o RH configurou.
	HerdadaDoRH bool `json:"herdada_do_rh"`
}

// SalvarConfigDTO recebe o provedor e, opcionalmente, a chave (vazia = manter atual).
type SalvarConfigDTO struct {
	Provedor string `json:"provedor" binding:"required"`
	Chave    string `json:"chave" binding:"omitempty"`
}

// ChatDTO é uma pergunta do gestor ao assistente.
type ChatDTO struct {
	Mensagem string `json:"mensagem" binding:"required"`
}
