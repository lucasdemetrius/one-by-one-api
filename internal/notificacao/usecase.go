// Pacote: internal/notificacao
// Arquivo: usecase.go
// Descrição: Regras de negócio das notificações in-app (listar, marcar lida,
//            contar não-lidas) e das preferências (ler/salvar).
// Autor: OneByOne API
// Criado em: 2026

package notificacao

// UseCase define as operações de notificação.
type UseCase interface {
	Listar(usuarioID string) ([]NotificacaoRespostaDTO, error)
	ContarNaoLidas(usuarioID string) (int, error)
	MarcarLida(id, usuarioID string) error
	MarcarTodasLidas(usuarioID string) error
	ObterPref(usuarioID string) (PrefDTO, error)
	SalvarPref(usuarioID string, dto PrefDTO) error
}

type useCaseImpl struct {
	repo Repositorio
}

// NovoUseCase cria o UseCase de notificações.
func NovoUseCase(repo Repositorio) UseCase {
	return &useCaseImpl{repo: repo}
}

func paraDTO(n Notificacao) NotificacaoRespostaDTO {
	return NotificacaoRespostaDTO{
		ID:       n.ID,
		Tipo:     n.Tipo,
		Titulo:   n.Titulo,
		Mensagem: n.Mensagem,
		Link:     n.Link,
		Lida:     n.Lida,
		CriadoEm: n.CriadoEm,
	}
}

func (uc *useCaseImpl) Listar(usuarioID string) ([]NotificacaoRespostaDTO, error) {
	itens, err := uc.repo.ListarPorUsuario(usuarioID, 30)
	if err != nil {
		return nil, err
	}
	lista := make([]NotificacaoRespostaDTO, 0, len(itens))
	for _, n := range itens {
		lista = append(lista, paraDTO(n))
	}
	return lista, nil
}

func (uc *useCaseImpl) ContarNaoLidas(usuarioID string) (int, error) {
	return uc.repo.ContarNaoLidas(usuarioID)
}

func (uc *useCaseImpl) MarcarLida(id, usuarioID string) error {
	return uc.repo.MarcarLida(id, usuarioID)
}

func (uc *useCaseImpl) MarcarTodasLidas(usuarioID string) error {
	return uc.repo.MarcarTodasLidas(usuarioID)
}

func (uc *useCaseImpl) ObterPref(usuarioID string) (PrefDTO, error) {
	p, err := uc.repo.ObterPref(usuarioID)
	if err != nil {
		return PrefDTO{}, err
	}
	return PrefDTO{Agenda1Dia: p.Agenda1Dia, AgendaHoje: p.AgendaHoje, Agenda1H: p.Agenda1H}, nil
}

func (uc *useCaseImpl) SalvarPref(usuarioID string, dto PrefDTO) error {
	return uc.repo.SalvarPref(Pref{
		UsuarioID:  usuarioID,
		Agenda1Dia: dto.Agenda1Dia,
		AgendaHoje: dto.AgendaHoje,
		Agenda1H:   dto.Agenda1H,
	})
}
