// Pacote: internal/equipe
// Arquivo: mapper.go
// Descrição: Converte entre a entidade Equipe e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package equipe

// ParaRespostaDTO converte uma entidade Equipe para o DTO de resposta da API.
// fotoURL é a URL presignada já gerada pelo UseCase (nil quando sem foto).
func ParaRespostaDTO(e Equipe, fotoURL *string) EquipeRespostaDTO {
	return EquipeRespostaDTO{
		ID:            e.ID,
		UsuarioID:     e.UsuarioID,
		OrganizacaoID: e.OrganizacaoID,
		TemplateID:    e.TemplateID,
		Nome:          e.Nome,
		CriadoEm:     e.CriadoEm,
		AlteradoEm:   e.AlteradoEm,
		FotoURL:       fotoURL,
	}
}
