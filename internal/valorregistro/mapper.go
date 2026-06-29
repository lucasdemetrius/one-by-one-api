// Pacote: internal/valorregistro
// Arquivo: mapper.go
// Descrição: Converte entre a entidade ValorRegistro e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package valorregistro

import "encoding/json"

// ParaRespostaDTO converte uma entidade ValorRegistro para o DTO de resposta da API
func ParaRespostaDTO(v ValorRegistro) ValorRegistroRespostaDTO {
	// []byte nil (NULL do banco) deserializado para interface{} no DTO
	var valorJSON interface{}
	if len(v.ValorJSON) > 0 {
		_ = json.Unmarshal(v.ValorJSON, &valorJSON)
	}
	return ValorRegistroRespostaDTO{
		ID:         v.ID,
		RegistroID: v.RegistroID,
		BlocoID:    v.BlocoID,
		ValorTexto: v.ValorTexto,
		ValorJSON:  valorJSON,
		CriadoEm:  v.CriadoEm,
		AlteradoEm: v.AlteradoEm,
	}
}

// ParaListaRespostaDTO converte uma fatia de entidades para uma fatia de DTOs
func ParaListaRespostaDTO(valores []ValorRegistro) []ValorRegistroRespostaDTO {
	lista := make([]ValorRegistroRespostaDTO, 0, len(valores))
	for _, v := range valores {
		lista = append(lista, ParaRespostaDTO(v))
	}
	return lista
}
