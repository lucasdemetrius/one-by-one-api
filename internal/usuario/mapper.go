// Pacote: internal/usuario
// Arquivo: mapper.go
// Descrição: Responsável pela conversão entre a entidade Usuario e os DTOs
//            de resposta. Garante que dados sensíveis (senha, soft delete)
//            nunca sejam expostos para fora da camada de repositório.
// Autor: OneByOne API
// Criado em: 2025

package usuario

// ParaRespostaDTO converte uma entidade Usuario para o DTO de resposta da API.
// fotoURL é a URL presignada já gerada pelo UseCase (nil quando sem foto).
// A senha e os campos de soft delete são propositalmente omitidos nesta conversão.
func ParaRespostaDTO(u Usuario, fotoURL *string) UsuarioRespostaDTO {
	return UsuarioRespostaDTO{
		ID:         u.ID,
		Nome:       u.Nome,
		Email:      u.Email,
		Role:       u.Role,
		CriadoEm:   u.CriadoEm,
		AlteradoEm: u.AlteradoEm,
		FotoURL:    fotoURL,
	}
}
