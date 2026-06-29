// Pacote: internal/valorregistro
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de valor de registro,
//            intermediando entre o Controller (HTTP) e o Repository (banco).
// Autor: OneByOne API
// Criado em: 2025

package valorregistro

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"onebyone-api/internal/registroonebyone"
)

// ErrAcessoNegado: falta de posse (a resposta pertence a um 1:1 de outro líder).
// Mensagem "não encontrada" → controller responde 404.
var ErrAcessoNegado = errors.New("resposta não encontrada")

// UseCase define o contrato das operações de negócio do módulo de valor de registro.
// A posse é herdada da cadeia: valor → registro → reunião (onebyone) → usuario_id.
type UseCase interface {
	// Criar salva a resposta de um bloco dentro de um registro do líder logado
	Criar(dto CriarValorRegistroDTO, usuarioID string) (ValorRegistroRespostaDTO, error)
	// BuscarPorId localiza um valor ativo pelo UUID (só do líder dono)
	BuscarPorId(id string, usuarioID string) (ValorRegistroRespostaDTO, error)
	// ListarPorRegistro retorna as respostas de um registro do líder logado
	ListarPorRegistro(registroID string, usuarioID string) ([]ValorRegistroRespostaDTO, error)
	// Atualizar modifica o conteúdo de uma resposta existente (só do líder dono)
	Atualizar(id string, usuarioID string, dto AtualizarValorRegistroDTO) (ValorRegistroRespostaDTO, error)
	// Deletar realiza a exclusão lógica do valor (só do líder dono)
	Deletar(id string, usuarioID string) error
}

// useCaseImpl é a implementação concreta do UseCase de valor de registro
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// registroUC resolve a posse pela cadeia registro → reunião → usuario_id
	registroUC registroonebyone.UseCase
}

// NovoUseCase cria e retorna uma nova instância do UseCase de valor de registro
func NovoUseCase(repo Repositorio, registroUC registroonebyone.UseCase) UseCase {
	return &useCaseImpl{repo: repo, registroUC: registroUC}
}

// garantirPosseRegistro confere se o registro pertence ao líder logado.
func (uc *useCaseImpl) garantirPosseRegistro(registroID, usuarioID string) error {
	dono, err := uc.registroUC.RegistroPertenceAoUsuario(registroID, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return nil
}

// Criar valida a posse do registro e que ao menos um dos campos de valor foi informado.
func (uc *useCaseImpl) Criar(dto CriarValorRegistroDTO, usuarioID string) (ValorRegistroRespostaDTO, error) {
	if err := uc.garantirPosseRegistro(dto.RegistroID, usuarioID); err != nil {
		return ValorRegistroRespostaDTO{}, err
	}
	if dto.ValorTexto == nil && dto.ValorJSON == nil {
		return ValorRegistroRespostaDTO{}, fmt.Errorf("é necessário informar valor_texto ou valor_json")
	}

	var valorJSONBytes []byte
	if dto.ValorJSON != nil {
		b, err := json.Marshal(dto.ValorJSON)
		if err != nil {
			return ValorRegistroRespostaDTO{}, fmt.Errorf("valor_json inválido: %w", err)
		}
		valorJSONBytes = b
	}

	novoValor := ValorRegistro{
		ID:         uuid.New().String(),
		RegistroID: dto.RegistroID,
		BlocoID:    dto.BlocoID,
		ValorTexto: dto.ValorTexto,
		ValorJSON:  valorJSONBytes,
		CriadoEm:  time.Now(),
	}

	criado, err := uc.repo.Criar(novoValor)
	if err != nil {
		return ValorRegistroRespostaDTO{}, fmt.Errorf("erro ao salvar resposta: %w", err)
	}
	return ParaRespostaDTO(criado), nil
}

// BuscarPorId localiza um valor de registro ativo pelo UUID, validando a posse.
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (ValorRegistroRespostaDTO, error) {
	v, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ValorRegistroRespostaDTO{}, fmt.Errorf("resposta não encontrada: %w", err)
	}
	if err := uc.garantirPosseRegistro(v.RegistroID, usuarioID); err != nil {
		return ValorRegistroRespostaDTO{}, err
	}
	return ParaRespostaDTO(v), nil
}

// ListarPorRegistro retorna as respostas de um registro do líder logado.
func (uc *useCaseImpl) ListarPorRegistro(registroID string, usuarioID string) ([]ValorRegistroRespostaDTO, error) {
	if err := uc.garantirPosseRegistro(registroID, usuarioID); err != nil {
		return nil, err
	}
	valores, err := uc.repo.ListarPorRegistro(registroID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar respostas do registro: %w", err)
	}
	return ParaListaRespostaDTO(valores), nil
}

// Atualizar valida a posse e aplica as modificações preservando os dados não informados.
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarValorRegistroDTO) (ValorRegistroRespostaDTO, error) {
	// Carrega o estado atual para aplicar atualização parcial
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ValorRegistroRespostaDTO{}, fmt.Errorf("resposta não encontrada: %w", err)
	}
	if err := uc.garantirPosseRegistro(atual.RegistroID, usuarioID); err != nil {
		return ValorRegistroRespostaDTO{}, err
	}

	if dto.ValorTexto != nil {
		atual.ValorTexto = dto.ValorTexto
	}
	if dto.ValorJSON != nil {
		b, err := json.Marshal(dto.ValorJSON)
		if err != nil {
			return ValorRegistroRespostaDTO{}, fmt.Errorf("valor_json inválido: %w", err)
		}
		atual.ValorJSON = b
	}

	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return ValorRegistroRespostaDTO{}, fmt.Errorf("erro ao atualizar resposta: %w", err)
	}
	return ParaRespostaDTO(atualizado), nil
}

// Deletar valida a posse e delega a exclusão lógica ao repositório.
func (uc *useCaseImpl) Deletar(id string, usuarioID string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("resposta não encontrada: %w", err)
	}
	if err := uc.garantirPosseRegistro(atual.RegistroID, usuarioID); err != nil {
		return err
	}
	return uc.repo.DeletarSoft(id, usuarioID)
}
