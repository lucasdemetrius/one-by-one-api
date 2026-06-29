// Pacote: internal/registroonebyone
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de registro de one-on-one.
//            Ao criar um registro, chama oneaone.UseCase.ResolverTemplate para
//            determinar automaticamente qual template aplicar, seguindo a regra
//            de herança documentada em oneaone/usecase.go.
// Autor: OneByOne API
// Criado em: 2025

package registroonebyone

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"onebyone-api/internal/onebyone"
)

// ErrAcessoNegado: falta de posse (o registro/reunião não é do líder logado).
// Mensagem "não encontrado" → controller responde 404.
var ErrAcessoNegado = errors.New("registro não encontrado")

// UseCase define o contrato das operações de negócio do módulo de registro de one-on-one.
// A posse é herdada da reunião pai (onebyone → usuario_id), via onebyoneUC.
type UseCase interface {
	// Criar abre um novo registro para uma reunião do líder logado
	Criar(dto CriarRegistroOneByOneDTO, usuarioID string) (RegistroOneByOneRespostaDTO, error)
	// BuscarPorId localiza um registro ativo pelo UUID (só do líder dono da reunião)
	BuscarPorId(id string, usuarioID string) (RegistroOneByOneRespostaDTO, error)
	// ListarPorOneByOne retorna os registros de uma reunião do líder logado
	ListarPorOneByOne(oneaoneID string, usuarioID string) ([]RegistroOneByOneRespostaDTO, error)
	// Deletar realiza a exclusão lógica do registro (só do líder dono)
	Deletar(id string, usuarioID string) error
	// RegistroPertenceAoUsuario expõe a posse (registro → reunião → usuario_id)
	// para reuso por valorregistro.
	RegistroPertenceAoUsuario(registroID, usuarioID string) (bool, error)
}

// useCaseImpl é a implementação concreta do UseCase de registro de one-on-one
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// onebyoneUC é o UseCase do módulo onebyone, usado para resolver o template
	// seguindo a regra de herança: colaborador → equipe → organização → padrão do líder
	onebyoneUC onebyone.UseCase
}

// NovoUseCase cria e retorna uma nova instância do UseCase de registro de one-on-one
func NovoUseCase(repo Repositorio, onebyoneUC onebyone.UseCase) UseCase {
	return &useCaseImpl{repo: repo, onebyoneUC: onebyoneUC}
}

// Criar abre um novo registro para uma reunião, resolvendo o template automaticamente
// pela regra de herança definida em oneaone/usecase.go antes de persistir.
func (uc *useCaseImpl) Criar(dto CriarRegistroOneByOneDTO, usuarioID string) (RegistroOneByOneRespostaDTO, error) {
	// POSSE: a reunião precisa ser do líder logado.
	dono, err := uc.onebyoneUC.PertenceAoUsuario(dto.OneByOneID, usuarioID)
	if err != nil {
		return RegistroOneByOneRespostaDTO{}, err
	}
	if !dono {
		return RegistroOneByOneRespostaDTO{}, ErrAcessoNegado
	}

	// Resolve o template a ser usado seguindo a regra de prioridade:
	// colaborador.template_id → equipe.template_id → organizacao.template_id → padrão do líder
	templateID, err := uc.onebyoneUC.ResolverTemplate(dto.OneByOneID)
	if err != nil {
		return RegistroOneByOneRespostaDTO{}, fmt.Errorf("não foi possível abrir a reunião: %w", err)
	}

	novoRegistro := RegistroOneByOne{
		ID:         uuid.New().String(),
		OneByOneID:  dto.OneByOneID,
		TemplateID: templateID,
		CriadoEm:  time.Now(),
	}

	criado, err := uc.repo.Criar(novoRegistro)
	if err != nil {
		return RegistroOneByOneRespostaDTO{}, fmt.Errorf("erro ao criar registro de one-on-one: %w", err)
	}
	return ParaRespostaDTO(criado), nil
}

// garantirPosse confere se a reunião pai do registro/oneaone é do líder logado.
func (uc *useCaseImpl) garantirPosseOneByOne(oneaoneID, usuarioID string) error {
	dono, err := uc.onebyoneUC.PertenceAoUsuario(oneaoneID, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return nil
}

// BuscarPorId localiza um registro ativo pelo UUID, validando a posse via a reunião pai.
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (RegistroOneByOneRespostaDTO, error) {
	reg, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return RegistroOneByOneRespostaDTO{}, fmt.Errorf("registro não encontrado: %w", err)
	}
	if err := uc.garantirPosseOneByOne(reg.OneByOneID, usuarioID); err != nil {
		return RegistroOneByOneRespostaDTO{}, err
	}
	return ParaRespostaDTO(reg), nil
}

// ListarPorOneByOne retorna os registros da reunião (do líder dono).
func (uc *useCaseImpl) ListarPorOneByOne(oneaoneID string, usuarioID string) ([]RegistroOneByOneRespostaDTO, error) {
	if err := uc.garantirPosseOneByOne(oneaoneID, usuarioID); err != nil {
		return nil, err
	}
	registros, err := uc.repo.ListarPorOneByOne(oneaoneID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar registros do one-on-one: %w", err)
	}
	return ParaListaRespostaDTO(registros), nil
}

// Deletar valida a posse (via reunião pai) e delega a exclusão lógica.
func (uc *useCaseImpl) Deletar(id string, usuarioID string) error {
	reg, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("registro não encontrado: %w", err)
	}
	if err := uc.garantirPosseOneByOne(reg.OneByOneID, usuarioID); err != nil {
		return err
	}
	return uc.repo.DeletarSoft(id, usuarioID)
}

// RegistroPertenceAoUsuario resolve registro → reunião → usuario_id (para valorregistro).
func (uc *useCaseImpl) RegistroPertenceAoUsuario(registroID, usuarioID string) (bool, error) {
	reg, err := uc.repo.BuscarPorId(registroID)
	if err != nil {
		return false, nil
	}
	return uc.onebyoneUC.PertenceAoUsuario(reg.OneByOneID, usuarioID)
}
