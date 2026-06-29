// Pacote: internal/pdi
// Arquivo: usecase.go
// Descrição: Regras de negócio do PDI. Toda operação valida a POSSE do liderado
//            (o gestor dono, via colaborador.PertenceAoLider). Recurso alheio → 404.
// Autor: OneByOne API
// Criado em: 2026

package pdi

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrAcessoNegado: sem posse. Mensagem "não encontrado" → 404 no controller.
var ErrAcessoNegado = errors.New("item de PDI não encontrado")

// PosseColaborador é o pedaço do colaborador.UseCase de que precisamos (posse do líder).
type PosseColaborador interface {
	PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error)
}

// UseCase define as operações do PDI.
type UseCase interface {
	Listar(colaboradorID, usuarioID string) ([]ItemPDIRespostaDTO, error)
	Criar(colaboradorID, usuarioID string, dto CriarItemPDIDTO) (ItemPDIRespostaDTO, error)
	Atualizar(itemID, usuarioID string, dto AtualizarItemPDIDTO) (ItemPDIRespostaDTO, error)
	Deletar(itemID, usuarioID string) error
}

type useCaseImpl struct {
	repo    Repositorio
	colabUC PosseColaborador
}

// NovoUseCase cria o UseCase de PDI.
func NovoUseCase(repo Repositorio, colabUC PosseColaborador) UseCase {
	return &useCaseImpl{repo: repo, colabUC: colabUC}
}

func parsearPrazo(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, fmt.Errorf("prazo inválido — use AAAA-MM-DD")
	}
	t = t.Add(12 * time.Hour) // meio-dia evita pulo de fuso
	return &t, nil
}

func paraDTO(i ItemPDI) ItemPDIRespostaDTO {
	dto := ItemPDIRespostaDTO{
		ID:            i.ID,
		ColaboradorID: i.ColaboradorID,
		Titulo:        i.Titulo,
		Descricao:     i.Descricao,
		Concluido:     i.Concluido,
		CriadoEm:      i.CriadoEm,
	}
	if i.Prazo != nil {
		p := i.Prazo.Format("2006-01-02")
		dto.Prazo = &p
	}
	if i.ConcluidoEm != nil {
		c := i.ConcluidoEm.Format("2006-01-02")
		dto.ConcluidoEm = &c
	}
	return dto
}

func (uc *useCaseImpl) garantirPosse(colaboradorID, usuarioID string) error {
	dono, err := uc.colabUC.PertenceAoLider(colaboradorID, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return nil
}

func (uc *useCaseImpl) Listar(colaboradorID, usuarioID string) ([]ItemPDIRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return nil, err
	}
	itens, err := uc.repo.ListarPorColaborador(colaboradorID)
	if err != nil {
		return nil, err
	}
	lista := make([]ItemPDIRespostaDTO, 0, len(itens))
	for _, i := range itens {
		lista = append(lista, paraDTO(i))
	}
	return lista, nil
}

func (uc *useCaseImpl) Criar(colaboradorID, usuarioID string, dto CriarItemPDIDTO) (ItemPDIRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return ItemPDIRespostaDTO{}, err
	}
	prazo, err := parsearPrazo(dto.Prazo)
	if err != nil {
		return ItemPDIRespostaDTO{}, err
	}
	var desc *string
	if dto.Descricao != "" {
		desc = &dto.Descricao
	}
	item := ItemPDI{
		ID:            uuid.New().String(),
		ColaboradorID: colaboradorID,
		Titulo:        dto.Titulo,
		Descricao:     desc,
		Prazo:         prazo,
		Concluido:     false,
		CriadoEm:      time.Now(),
	}
	criado, err := uc.repo.Criar(item)
	if err != nil {
		return ItemPDIRespostaDTO{}, err
	}
	return paraDTO(criado), nil
}

func (uc *useCaseImpl) Atualizar(itemID, usuarioID string, dto AtualizarItemPDIDTO) (ItemPDIRespostaDTO, error) {
	atual, err := uc.repo.BuscarPorId(itemID)
	if err != nil {
		return ItemPDIRespostaDTO{}, fmt.Errorf("item de PDI não encontrado")
	}
	if err := uc.garantirPosse(atual.ColaboradorID, usuarioID); err != nil {
		return ItemPDIRespostaDTO{}, err
	}
	if dto.Titulo != "" {
		atual.Titulo = dto.Titulo
	}
	if dto.Prazo != "" {
		prazo, err := parsearPrazo(dto.Prazo)
		if err != nil {
			return ItemPDIRespostaDTO{}, err
		}
		atual.Prazo = prazo
	}
	if dto.Concluido != nil && *dto.Concluido != atual.Concluido {
		atual.Concluido = *dto.Concluido
		// Carimba quando foi concluído (para a evolução do PDR); limpa ao reabrir.
		if atual.Concluido {
			agora := time.Now()
			atual.ConcluidoEm = &agora
		} else {
			atual.ConcluidoEm = nil
		}
	}
	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return ItemPDIRespostaDTO{}, err
	}
	return paraDTO(atualizado), nil
}

func (uc *useCaseImpl) Deletar(itemID, usuarioID string) error {
	atual, err := uc.repo.BuscarPorId(itemID)
	if err != nil {
		return fmt.Errorf("item de PDI não encontrado")
	}
	if err := uc.garantirPosse(atual.ColaboradorID, usuarioID); err != nil {
		return err
	}
	return uc.repo.DeletarSoft(itemID)
}
