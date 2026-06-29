package auditoria

import (
	"time"

	"github.com/google/uuid"
)

// UseCase define o contrato de operações de auditoria
type UseCase interface {
	// Registrar grava um evento de auditoria de forma assíncrona (não bloqueia a requisição)
	Registrar(usuarioID *string, acao, entidade string, entidadeID *string, ip, userAgent string)
	// RegistrarEvento grava um evento enviado diretamente pelo frontend
	RegistrarEvento(usuarioID string, dto EventoDTO, ip, userAgent string)
	// ListarPorUsuario retorna os últimos N eventos de um usuário
	ListarPorUsuario(usuarioID string, limite int) ([]AuditoriaRespostaDTO, error)
	// ListarPorEntidade retorna a linha do tempo de uma entidade (ex.: colaborador)
	ListarPorEntidade(entidadeID string, limite int) ([]AuditoriaRespostaDTO, error)
}

type useCaseImpl struct {
	repo Repositorio
}

func NovoUseCase(repo Repositorio) UseCase {
	return &useCaseImpl{repo: repo}
}

func (uc *useCaseImpl) Registrar(usuarioID *string, acao, entidade string, entidadeID *string, ip, userAgent string) {
	a := Auditoria{
		ID:         uuid.New().String(),
		UsuarioID:  usuarioID,
		Acao:       acao,
		Entidade:   entidade,
		EntidadeID: entidadeID,
		IP:         strPtr(ip),
		UserAgent:  strPtr(userAgent),
		CriadoEm:  time.Now(),
	}
	// Grava em goroutine separada para não bloquear a resposta HTTP
	go func() { _ = uc.repo.Gravar(a) }()
}

func (uc *useCaseImpl) RegistrarEvento(usuarioID string, dto EventoDTO, ip, userAgent string) {
	uid := &usuarioID
	uc.Registrar(uid, dto.Acao, dto.Entidade, dto.EntidadeID, ip, userAgent)
}

func (uc *useCaseImpl) ListarPorUsuario(usuarioID string, limite int) ([]AuditoriaRespostaDTO, error) {
	if limite <= 0 || limite > 200 {
		limite = 50
	}
	registros, err := uc.repo.ListarPorUsuario(usuarioID, limite)
	if err != nil {
		return nil, err
	}
	return paraDTOs(registros), nil
}

func (uc *useCaseImpl) ListarPorEntidade(entidadeID string, limite int) ([]AuditoriaRespostaDTO, error) {
	if limite <= 0 || limite > 200 {
		limite = 50
	}
	registros, err := uc.repo.ListarPorEntidade(entidadeID, limite)
	if err != nil {
		return nil, err
	}
	return paraDTOs(registros), nil
}

func paraDTOs(registros []Auditoria) []AuditoriaRespostaDTO {
	lista := make([]AuditoriaRespostaDTO, 0, len(registros))
	for _, r := range registros {
		lista = append(lista, AuditoriaRespostaDTO{
			ID:         r.ID,
			UsuarioID:  r.UsuarioID,
			Acao:       r.Acao,
			Entidade:   r.Entidade,
			EntidadeID: r.EntidadeID,
			IP:         r.IP,
			UserAgent:  r.UserAgent,
			CriadoEm:  r.CriadoEm,
		})
	}
	return lista
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
