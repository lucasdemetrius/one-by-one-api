// Pacote: internal/template
// Arquivo: mapper.go
// Descrição: Converte entre a entidade Template e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package template

// ParaRespostaDTO converte uma entidade Template para o DTO de resposta da API
func ParaRespostaDTO(t Template) TemplateRespostaDTO {
	return TemplateRespostaDTO{
		ID:         t.ID,
		UsuarioID:  t.UsuarioID,
		Nome:       t.Nome,
		CriadoEm:  t.CriadoEm,
		AlteradoEm: t.AlteradoEm,
	}
}

// ParaListaRespostaDTO converte uma fatia de entidades para uma fatia de DTOs
func ParaListaRespostaDTO(templates []Template) []TemplateRespostaDTO {
	lista := make([]TemplateRespostaDTO, 0, len(templates))
	for _, t := range templates {
		lista = append(lista, ParaRespostaDTO(t))
	}
	return lista
}
