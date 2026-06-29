// Pacote: internal/tabuleiro
// Arquivo: usecase.go
// Descrição: Regras de negócio do tabuleiro. A POSSE usa PodeAcessar (líder dono
//            OU o próprio liderado), pois o board do 1:1 é COLABORATIVO — os dois
//            arrastam temas ao vivo, então ambos podem ler e salvar. Recurso
//            alheio → 404.
// Autor: OneByOne API
// Criado em: 2026

package tabuleiro

import (
	"encoding/json"
	"errors"
)

// ErrAcessoNegado: sem posse. Mensagem "não encontrado" → 404 no controller.
var ErrAcessoNegado = errors.New("tabuleiro não encontrado")

// PosseColaborador é o pedaço do colaborador.UseCase de que precisamos. Usamos
// PodeAcessar (líder OU o próprio liderado) porque o tabuleiro é colaborativo.
type PosseColaborador interface {
	PodeAcessar(colaboradorID, usuarioID string) (bool, error)
}

// UseCase define as operações do tabuleiro.
type UseCase interface {
	Obter(colaboradorID, usuarioID string) (TabuleiroRespostaDTO, error)
	Salvar(colaboradorID, usuarioID string, dto SalvarTabuleiroDTO) error
}

type useCaseImpl struct {
	repo    Repositorio
	colabUC PosseColaborador
}

// NovoUseCase cria o UseCase de tabuleiro.
func NovoUseCase(repo Repositorio, colabUC PosseColaborador) UseCase {
	return &useCaseImpl{repo: repo, colabUC: colabUC}
}

func (uc *useCaseImpl) garantirPosse(colaboradorID, usuarioID string) error {
	pode, err := uc.colabUC.PodeAcessar(colaboradorID, usuarioID)
	if err != nil {
		return err
	}
	if !pode {
		return ErrAcessoNegado
	}
	return nil
}

func (uc *useCaseImpl) Obter(colaboradorID, usuarioID string) (TabuleiroRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return TabuleiroRespostaDTO{}, err
	}
	estado, err := uc.repo.Buscar(colaboradorID)
	if err != nil {
		if errors.Is(err, ErrSemTabuleiro) {
			return TabuleiroRespostaDTO{Estado: nil}, nil // ainda não há board salvo
		}
		return TabuleiroRespostaDTO{}, err
	}
	return TabuleiroRespostaDTO{Estado: json.RawMessage(estado)}, nil
}

func (uc *useCaseImpl) Salvar(colaboradorID, usuarioID string, dto SalvarTabuleiroDTO) error {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return err
	}
	return uc.repo.Salvar(colaboradorID, string(dto.Estado))
}
