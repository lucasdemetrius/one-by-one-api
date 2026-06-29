// Pacote: internal/classificacao
// Arquivo: usecase.go
// Descrição: Regras de negócio da classificação 9-box. Valida o colaborador e
//            persiste/lista o posicionamento desempenho × potencial.
// Autor: OneByOne API
// Criado em: 2025

package classificacao

import (
	"errors"
	"time"

	"onebyone-api/internal/colaborador"
)

// ErrAcessoNegado: falta de posse. Mensagem "não encontrado" → controller responde 404.
var ErrAcessoNegado = errors.New("colaborador não encontrado")

// UseCase define as operações de negócio da classificação. Recebem o usuarioID
// do líder logado e validam a posse (avaliação de RH é exclusiva do gestor dono).
type UseCase interface {
	Definir(colaboradorID string, dto DefinirClassificacaoDTO, usuarioID string) (ClassificacaoRespostaDTO, error)
	ListarPorOrganizacao(organizacaoID string, usuarioID string) ([]ClassificacaoRespostaDTO, error)
}

type useCaseImpl struct {
	repo          Repositorio
	colaboradorUC colaborador.UseCase
}

// NovoUseCase cria o UseCase de classificação.
func NovoUseCase(repo Repositorio, colaboradorUC colaborador.UseCase) UseCase {
	return &useCaseImpl{repo: repo, colaboradorUC: colaboradorUC}
}

func (uc *useCaseImpl) Definir(colaboradorID string, dto DefinirClassificacaoDTO, usuarioID string) (ClassificacaoRespostaDTO, error) {
	// POSSE: só o líder dono do liderado pode definir a nota 9-box.
	dono, err := uc.colaboradorUC.PertenceAoLider(colaboradorID, usuarioID)
	if err != nil {
		return ClassificacaoRespostaDTO{}, err
	}
	if !dono {
		return ClassificacaoRespostaDTO{}, ErrAcessoNegado
	}

	c := Classificacao{
		ColaboradorID: colaboradorID,
		Desempenho:    dto.Desempenho,
		Potencial:     dto.Potencial,
		AtualizadoEm:  time.Now(),
	}
	if err := uc.repo.Definir(c); err != nil {
		return ClassificacaoRespostaDTO{}, err
	}

	return ClassificacaoRespostaDTO{
		ColaboradorID: c.ColaboradorID,
		Desempenho:    c.Desempenho,
		Potencial:     c.Potencial,
	}, nil
}

func (uc *useCaseImpl) ListarPorOrganizacao(organizacaoID string, usuarioID string) ([]ClassificacaoRespostaDTO, error) {
	// POSSE: a organização tem de ser do líder logado (senão vaza avaliação alheia).
	dono, err := uc.colaboradorUC.OrganizacaoPertenceAoLider(organizacaoID, usuarioID)
	if err != nil {
		return nil, err
	}
	if !dono {
		return nil, ErrAcessoNegado
	}
	lista, err := uc.repo.ListarPorOrganizacao(organizacaoID)
	if err != nil {
		return nil, err
	}
	resp := make([]ClassificacaoRespostaDTO, 0, len(lista))
	for _, c := range lista {
		resp = append(resp, ClassificacaoRespostaDTO{
			ColaboradorID: c.ColaboradorID,
			Desempenho:    c.Desempenho,
			Potencial:     c.Potencial,
		})
	}
	return resp, nil
}
