// Pacote: internal/registroonebyone
// Arquivo: mapper.go
// Descrição: Converte entre a entidade RegistroOneByOne e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package registroonebyone

// ParaRespostaDTO converte uma entidade RegistroOneByOne para o DTO de resposta da API
func ParaRespostaDTO(r RegistroOneByOne) RegistroOneByOneRespostaDTO {
	return RegistroOneByOneRespostaDTO{
		ID:         r.ID,
		OneByOneID:  r.OneByOneID,
		TemplateID: r.TemplateID,
		CriadoEm:  r.CriadoEm,
		AlteradoEm: r.AlteradoEm,
	}
}

// ParaListaRespostaDTO converte uma fatia de entidades para uma fatia de DTOs
func ParaListaRespostaDTO(registros []RegistroOneByOne) []RegistroOneByOneRespostaDTO {
	lista := make([]RegistroOneByOneRespostaDTO, 0, len(registros))
	for _, r := range registros {
		lista = append(lista, ParaRespostaDTO(r))
	}
	return lista
}
