// Pacote: internal/colaborador
// Arquivo: mapper.go
// Descrição: Converte entre a entidade Colaborador e o DTO de resposta,
//            isolando o modelo de banco da camada de apresentação HTTP.
// Autor: OneByOne API
// Criado em: 2025

package colaborador

// ParaRespostaDTO converte uma entidade Colaborador para o DTO de resposta da API.
// fotoURL é a URL presignada já gerada pelo UseCase (nil quando sem foto).
func ParaRespostaDTO(c Colaborador, fotoURL *string) ColaboradorRespostaDTO {
	return ColaboradorRespostaDTO{
		ID:             c.ID,
		UsuarioID:      c.UsuarioID,
		OrganizacaoID:  c.OrganizacaoID,
		EquipeID:       c.EquipeID,
		TemplateID:     c.TemplateID,
		Nome:           c.Nome,
		Email:          c.Email,
		Whatsapp:       c.Whatsapp,
		DataNascimento: c.DataNascimento,
		CriadoEm:       c.CriadoEm,
		AlteradoEm:     c.AlteradoEm,
		FotoURL:        fotoURL,
		DesligadoEm:    c.DesligadoEm,
		Ativo:          c.DesligadoEm == nil,
	}
}
