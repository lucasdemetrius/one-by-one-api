// Pacote: internal/templatebloco
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de bloco de template,
//            intermediando entre o Controller (HTTP) e o Repository (banco).
// Autor: OneByOne API
// Criado em: 2025

package templatebloco

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"onebyone-api/internal/template"
)

// ErrAcessoNegado: falta de posse (o bloco/template não é do líder logado).
// Mensagem "não encontrado" → controller responde 404.
var ErrAcessoNegado = errors.New("bloco de template não encontrado")

// UseCase define o contrato das operações de negócio do módulo de bloco de template.
// A posse é herdada do template pai (template → usuario_id), via templateUC.
type UseCase interface {
	// Criar valida e persiste um novo bloco dentro de um template do líder logado
	Criar(dto CriarTemplateBlocoDTO, usuarioID string) (TemplateBlocoRespostaDTO, error)
	// BuscarPorId localiza um bloco ativo pelo UUID (só do líder dono do template)
	BuscarPorId(id string, usuarioID string) (TemplateBlocoRespostaDTO, error)
	// ListarPorTemplate retorna os blocos de um template do líder logado
	ListarPorTemplate(templateID string, usuarioID string) ([]TemplateBlocoRespostaDTO, error)
	// Atualizar aplica as alterações permitidas (só do líder dono do template)
	Atualizar(id string, usuarioID string, dto AtualizarTemplateBlocoDTO) (TemplateBlocoRespostaDTO, error)
	// Deletar realiza a exclusão lógica do bloco (só do líder dono do template)
	Deletar(id string, usuarioID string) error
}

// useCaseImpl é a implementação concreta do UseCase de bloco de template
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// templateUC resolve a posse pela cadeia bloco → template → usuario_id
	templateUC template.UseCase
}

// NovoUseCase cria e retorna uma nova instância do UseCase de bloco de template
func NovoUseCase(repo Repositorio, templateUC template.UseCase) UseCase {
	return &useCaseImpl{repo: repo, templateUC: templateUC}
}

// garantirPosseTemplate confere se o template pertence ao líder logado.
func (uc *useCaseImpl) garantirPosseTemplate(templateID, usuarioID string) error {
	dono, err := uc.templateUC.PertenceAoUsuario(templateID, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return nil
}

// Criar valida a posse do template e persiste o novo bloco
func (uc *useCaseImpl) Criar(dto CriarTemplateBlocoDTO, usuarioID string) (TemplateBlocoRespostaDTO, error) {
	if err := uc.garantirPosseTemplate(dto.TemplateID, usuarioID); err != nil {
		return TemplateBlocoRespostaDTO{}, err
	}
	novoBloco := TemplateBloco{
		ID:         uuid.New().String(),
		TemplateID: dto.TemplateID,
		Tipo:       dto.Tipo,
		Posicao:    dto.Posicao,
		Rotulo:     dto.Rotulo,
		CriadoEm:  time.Now(),
	}

	criado, err := uc.repo.Criar(novoBloco)
	if err != nil {
		return TemplateBlocoRespostaDTO{}, fmt.Errorf("erro ao criar bloco de template: %w", err)
	}
	return ParaRespostaDTO(criado), nil
}

// BuscarPorId localiza um bloco ativo pelo UUID, validando a posse via o template pai.
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (TemplateBlocoRespostaDTO, error) {
	bloco, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return TemplateBlocoRespostaDTO{}, fmt.Errorf("bloco de template não encontrado: %w", err)
	}
	if err := uc.garantirPosseTemplate(bloco.TemplateID, usuarioID); err != nil {
		return TemplateBlocoRespostaDTO{}, err
	}
	return ParaRespostaDTO(bloco), nil
}

// ListarPorTemplate retorna os blocos de um template do líder logado.
func (uc *useCaseImpl) ListarPorTemplate(templateID string, usuarioID string) ([]TemplateBlocoRespostaDTO, error) {
	if err := uc.garantirPosseTemplate(templateID, usuarioID); err != nil {
		return nil, err
	}
	blocos, err := uc.repo.ListarPorTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar blocos do template: %w", err)
	}
	return ParaListaRespostaDTO(blocos), nil
}

// Atualizar valida a posse e aplica apenas os campos informados, preservando os demais.
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarTemplateBlocoDTO) (TemplateBlocoRespostaDTO, error) {
	// Carrega o estado atual para permitir atualização parcial
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return TemplateBlocoRespostaDTO{}, fmt.Errorf("bloco de template não encontrado: %w", err)
	}
	if err := uc.garantirPosseTemplate(atual.TemplateID, usuarioID); err != nil {
		return TemplateBlocoRespostaDTO{}, err
	}

	if dto.Tipo != "" {
		atual.Tipo = dto.Tipo
	}
	if dto.Posicao != nil {
		atual.Posicao = *dto.Posicao
	}
	if dto.Rotulo != "" {
		atual.Rotulo = dto.Rotulo
	}

	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return TemplateBlocoRespostaDTO{}, fmt.Errorf("erro ao atualizar bloco de template: %w", err)
	}
	return ParaRespostaDTO(atualizado), nil
}

// Deletar valida a posse (via template pai) e delega a exclusão lógica.
func (uc *useCaseImpl) Deletar(id string, usuarioID string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("bloco de template não encontrado: %w", err)
	}
	if err := uc.garantirPosseTemplate(atual.TemplateID, usuarioID); err != nil {
		return err
	}
	return uc.repo.DeletarSoft(id, usuarioID)
}
