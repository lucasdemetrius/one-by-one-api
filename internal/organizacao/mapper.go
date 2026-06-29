// Pacote: internal/organizacao
// Arquivo: mapper.go
// Descrição: Converte entre a entidade Organizacao e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package organizacao

// ParaRespostaDTO converte uma entidade Organizacao para o DTO de resposta da API.
// fotoURL é a URL presignada já gerada pelo UseCase (nil quando sem foto).
func ParaRespostaDTO(o Organizacao, fotoURL *string) OrganizacaoRespostaDTO {
	return OrganizacaoRespostaDTO{
		ID:         o.ID,
		UsuarioID:  o.UsuarioID,
		TemplateID: o.TemplateID,
		Nome:       o.Nome,
		CriadoEm:  o.CriadoEm,
		AlteradoEm: o.AlteradoEm,
		FotoURL:    fotoURL,
	}
}
