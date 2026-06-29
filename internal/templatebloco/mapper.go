// Pacote: internal/templatebloco
// Arquivo: mapper.go
// Descrição: Converte entre a entidade TemplateBloco e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package templatebloco

// ParaRespostaDTO converte uma entidade TemplateBloco para o DTO de resposta da API
func ParaRespostaDTO(b TemplateBloco) TemplateBlocoRespostaDTO {
	return TemplateBlocoRespostaDTO{
		ID:         b.ID,
		TemplateID: b.TemplateID,
		Tipo:       b.Tipo,
		Posicao:    b.Posicao,
		Rotulo:     b.Rotulo,
		CriadoEm:  b.CriadoEm,
		AlteradoEm: b.AlteradoEm,
	}
}

// ParaListaRespostaDTO converte uma fatia de entidades para uma fatia de DTOs
func ParaListaRespostaDTO(blocos []TemplateBloco) []TemplateBlocoRespostaDTO {
	lista := make([]TemplateBlocoRespostaDTO, 0, len(blocos))
	for _, b := range blocos {
		lista = append(lista, ParaRespostaDTO(b))
	}
	return lista
}
