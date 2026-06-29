// Pacote: internal/onebyone
// Arquivo: mapper.go
// Descrição: Converte entre a entidade OneByOne e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package onebyone

// ParaRespostaDTO converte uma entidade OneByOne para o DTO de resposta da API
func ParaRespostaDTO(o OneByOne) OneByOneRespostaDTO {
	return OneByOneRespostaDTO{
		ID:            o.ID,
		UsuarioID:     o.UsuarioID,
		OrganizacaoID: o.OrganizacaoID,
		EquipeID:      o.EquipeID,
		ColaborID:     o.ColaborID,
		Recorrencia:   o.Recorrencia,
		Status:        o.Status,
		RealizadoEm:   o.RealizadoEm,
		DataAgendada:  o.DataAgendada,
		CriadoEm:      o.CriadoEm,
		AlteradoEm:    o.AlteradoEm,
	}
}

// ParaListaRespostaDTO converte uma fatia de entidades para uma fatia de DTOs
func ParaListaRespostaDTO(reunioes []OneByOne) []OneByOneRespostaDTO {
	lista := make([]OneByOneRespostaDTO, 0, len(reunioes))
	for _, o := range reunioes {
		lista = append(lista, ParaRespostaDTO(o))
	}
	return lista
}
