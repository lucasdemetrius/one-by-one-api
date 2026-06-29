// Pacote: internal/acompanhamento
// Arquivo: usecase.go
// Descrição: Regras de negócio do acompanhamento. Toda operação valida a POSSE do
//            liderado (gestor dono, via colaborador.PertenceAoLider). Recurso
//            alheio → 404. SENTIMENTO exige valor (1-5); os demais exigem título.
// Autor: OneByOne API
// Criado em: 2026

package acompanhamento

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrAcessoNegado: sem posse. Mensagem "não encontrado" → 404 no controller.
var ErrAcessoNegado = errors.New("acompanhamento não encontrado")

// Erros de NEGÓCIO (validação). São mensagens amigáveis que DEVEM chegar ao
// usuário (→ 400). O controller os reconhece via errors.Is e mostra a mensagem;
// qualquer outro erro é tratado como técnico/inesperado (→ 500 genérico).
var (
	// ErrDataInvalida: a data informada não está no formato esperado.
	ErrDataInvalida = errors.New("data inválida — use AAAA-MM-DD")
	// ErrHumorObrigatorio: SENTIMENTO precisa de uma nota de 1 a 5.
	ErrHumorObrigatorio = errors.New("informe o humor (de 1 a 5)")
	// ErrTituloObrigatorio: os demais tipos precisam de um título.
	ErrTituloObrigatorio = errors.New("informe um título")
)

// PosseColaborador é o pedaço do colaborador.UseCase de que precisamos (posse).
type PosseColaborador interface {
	PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error)
}

// UseCase define as operações de acompanhamento.
type UseCase interface {
	Listar(colaboradorID, usuarioID, tipo string) ([]AcompanhamentoRespostaDTO, error)
	Criar(colaboradorID, usuarioID string, dto CriarAcompanhamentoDTO) (AcompanhamentoRespostaDTO, error)
	Atualizar(id, usuarioID string, dto AtualizarAcompanhamentoDTO) (AcompanhamentoRespostaDTO, error)
	Deletar(id, usuarioID string) error
}

type useCaseImpl struct {
	repo    Repositorio
	colabUC PosseColaborador
}

// NovoUseCase cria o UseCase de acompanhamento.
func NovoUseCase(repo Repositorio, colabUC PosseColaborador) UseCase {
	return &useCaseImpl{repo: repo, colabUC: colabUC}
}

func parsearData(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, ErrDataInvalida
	}
	return t.Add(12 * time.Hour), nil // meio-dia evita pulo de fuso
}

func paraDTO(a Acompanhamento) AcompanhamentoRespostaDTO {
	return AcompanhamentoRespostaDTO{
		ID:            a.ID,
		ColaboradorID: a.ColaboradorID,
		Tipo:          a.Tipo,
		Titulo:        a.Titulo,
		Detalhe:       a.Detalhe,
		Valor:         a.Valor,
		DataRef:       a.DataRef.Format("2006-01-02"),
		CriadoEm:      a.CriadoEm,
	}
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

func (uc *useCaseImpl) Listar(colaboradorID, usuarioID, tipo string) ([]AcompanhamentoRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return nil, err
	}
	itens, err := uc.repo.ListarPorColaborador(colaboradorID, tipo)
	if err != nil {
		return nil, err
	}
	lista := make([]AcompanhamentoRespostaDTO, 0, len(itens))
	for _, a := range itens {
		lista = append(lista, paraDTO(a))
	}
	return lista, nil
}

func (uc *useCaseImpl) Criar(colaboradorID, usuarioID string, dto CriarAcompanhamentoDTO) (AcompanhamentoRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return AcompanhamentoRespostaDTO{}, err
	}
	// Regra por tipo: humor exige nota (1-5); os demais exigem um título.
	titulo := strings.TrimSpace(dto.Titulo)
	if dto.Tipo == TipoSentimento {
		if dto.Valor == nil {
			return AcompanhamentoRespostaDTO{}, ErrHumorObrigatorio
		}
	} else if titulo == "" {
		return AcompanhamentoRespostaDTO{}, ErrTituloObrigatorio
	}

	dataRef, err := parsearData(dto.DataRef)
	if err != nil {
		return AcompanhamentoRespostaDTO{}, err
	}
	var detalhe *string
	if dto.Detalhe != "" {
		detalhe = &dto.Detalhe
	}
	a := Acompanhamento{
		ID:            uuid.New().String(),
		ColaboradorID: colaboradorID,
		Tipo:          dto.Tipo,
		Titulo:        titulo,
		Detalhe:       detalhe,
		Valor:         dto.Valor,
		DataRef:       dataRef,
		CriadoEm:      time.Now(),
	}
	criado, err := uc.repo.Criar(a)
	if err != nil {
		return AcompanhamentoRespostaDTO{}, err
	}
	return paraDTO(criado), nil
}

func (uc *useCaseImpl) Atualizar(id, usuarioID string, dto AtualizarAcompanhamentoDTO) (AcompanhamentoRespostaDTO, error) {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return AcompanhamentoRespostaDTO{}, ErrAcessoNegado // inexistente/alheio → 404
	}
	if err := uc.garantirPosse(atual.ColaboradorID, usuarioID); err != nil {
		return AcompanhamentoRespostaDTO{}, err
	}
	if t := strings.TrimSpace(dto.Titulo); t != "" {
		atual.Titulo = t
	}
	if dto.Detalhe != nil {
		atual.Detalhe = dto.Detalhe
	}
	if dto.Valor != nil {
		atual.Valor = dto.Valor
	}
	if dto.DataRef != "" {
		dataRef, err := parsearData(dto.DataRef)
		if err != nil {
			return AcompanhamentoRespostaDTO{}, err
		}
		atual.DataRef = dataRef
	}
	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return AcompanhamentoRespostaDTO{}, err
	}
	return paraDTO(atualizado), nil
}

func (uc *useCaseImpl) Deletar(id, usuarioID string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ErrAcessoNegado // inexistente/alheio → 404
	}
	if err := uc.garantirPosse(atual.ColaboradorID, usuarioID); err != nil {
		return err
	}
	return uc.repo.DeletarSoft(id)
}
