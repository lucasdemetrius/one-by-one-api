// Pacote: internal/template
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de template,
//            intermediando entre o Controller (HTTP) e o Repository (banco).
// Autor: OneByOne API
// Criado em: 2025

package template

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrAcessoNegado: falta de posse (o template não é do líder logado). Mensagem
// "não encontrado" → controller responde 404.
var ErrAcessoNegado = errors.New("template não encontrado")

// UseCase define o contrato das operações de negócio do módulo de template
type UseCase interface {
	// Criar valida e persiste um novo template vinculado ao líder autenticado
	Criar(usuarioID string, dto CriarTemplateDTO) (TemplateRespostaDTO, error)
	// BuscarPorId localiza um template ativo pelo UUID (só do líder dono)
	BuscarPorId(id string, usuarioID string) (TemplateRespostaDTO, error)
	// ListarPorUsuario retorna todos os templates do líder autenticado
	ListarPorUsuario(usuarioID string) ([]TemplateRespostaDTO, error)
	// Atualizar renomeia um template existente do líder
	Atualizar(id string, usuarioID string, dto AtualizarTemplateDTO) (TemplateRespostaDTO, error)
	// Deletar realiza a exclusão lógica do template
	Deletar(id string, usuarioID string, deletadoPor string) error
	// PertenceAoUsuario diz se o template é do líder (reuso por templatebloco).
	PertenceAoUsuario(templateID, usuarioID string) (bool, error)
}

// useCaseImpl é a implementação concreta do UseCase de template
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
}

// NovoUseCase cria e retorna uma nova instância do UseCase de template
func NovoUseCase(repo Repositorio) UseCase {
	return &useCaseImpl{repo: repo}
}

// Criar gera o UUID, vincula o template ao líder autenticado e persiste no banco
func (uc *useCaseImpl) Criar(usuarioID string, dto CriarTemplateDTO) (TemplateRespostaDTO, error) {
	novoTemplate := Template{
		ID:        uuid.New().String(),
		UsuarioID: usuarioID,
		Nome:      dto.Nome,
		CriadoEm:  time.Now(),
	}

	criado, err := uc.repo.Criar(novoTemplate)
	if err != nil {
		return TemplateRespostaDTO{}, fmt.Errorf("erro ao criar template: %w", err)
	}
	return ParaRespostaDTO(criado), nil
}

// BuscarPorId localiza um template ativo pelo UUID, validando a posse (só o dono).
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (TemplateRespostaDTO, error) {
	t, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return TemplateRespostaDTO{}, fmt.Errorf("template não encontrado: %w", err)
	}
	if !uc.podeAgir(t.UsuarioID, usuarioID) {
		return TemplateRespostaDTO{}, ErrAcessoNegado
	}
	return ParaRespostaDTO(t), nil
}

// PertenceAoUsuario confere se o template é acessível pelo ator: dono direto OU RH do tenant
// do gestor dono. templatebloco herda a posse por aqui, ganhando o RH automaticamente.
func (uc *useCaseImpl) PertenceAoUsuario(templateID, usuarioID string) (bool, error) {
	t, err := uc.repo.BuscarPorId(templateID)
	if err != nil {
		return false, nil
	}
	if t.UsuarioID == usuarioID {
		return true, nil
	}
	return uc.repo.GestorPertenceAoRH(t.UsuarioID, usuarioID)
}

// podeAgir resume a posse: dono direto (igualdade) OU RH dono do gestor (tenant).
// Self-gating — para um não-RH o fallback nunca casa.
func (uc *useCaseImpl) podeAgir(donoUsuarioID, usuarioID string) bool {
	if donoUsuarioID == usuarioID {
		return true
	}
	ok, _ := uc.repo.GestorPertenceAoRH(donoUsuarioID, usuarioID)
	return ok
}

// ListarPorUsuario retorna todos os templates ativos do líder convertidos para DTOs
func (uc *useCaseImpl) ListarPorUsuario(usuarioID string) ([]TemplateRespostaDTO, error) {
	templates, err := uc.repo.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar templates: %w", err)
	}
	return ParaListaRespostaDTO(templates), nil
}

// Atualizar renomeia um template, verificando que o usuário é o proprietário
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarTemplateDTO) (TemplateRespostaDTO, error) {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return TemplateRespostaDTO{}, fmt.Errorf("template não encontrado: %w", err)
	}

	// Apenas o dono do template OU o RH do tenant pode alterá-lo
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return TemplateRespostaDTO{}, fmt.Errorf("você não tem permissão para alterar este template")
	}

	atual.Nome = dto.Nome

	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return TemplateRespostaDTO{}, fmt.Errorf("erro ao atualizar template: %w", err)
	}
	return ParaRespostaDTO(atualizado), nil
}

// Deletar verifica a propriedade e delega a exclusão lógica ao repositório
func (uc *useCaseImpl) Deletar(id string, usuarioID string, deletadoPor string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("template não encontrado: %w", err)
	}

	// Apenas o dono do template OU o RH do tenant pode excluí-lo
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return fmt.Errorf("você não tem permissão para excluir este template")
	}

	return uc.repo.DeletarSoft(id, deletadoPor)
}
