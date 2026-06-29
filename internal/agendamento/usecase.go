// Pacote: internal/agendamento
// Arquivo: usecase.go
// Descrição: Regras de negócio dos agendamentos de 1:1. Valida o liderado, parseia
//            a data/hora no fuso local e persiste/lista.
// Autor: OneByOne API
// Criado em: 2025

package agendamento

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"onebyone-api/internal/colaborador"
	"onebyone-api/pkg/email"
)

// UseCase define as operações de negócio dos agendamentos.
type UseCase interface {
	Criar(usuarioID string, dto CriarAgendamentoDTO) (AgendamentoRespostaDTO, error)
	ListarPorUsuario(usuarioID string) ([]AgendamentoRespostaDTO, error)
	Deletar(id, usuarioID string) error
	// Reagendar muda a data/hora de um 1:1 do gestor (arrastar no calendário)
	Reagendar(id, usuarioID, dataHora string) error
	// CancelarDoColaborador remove todos os 1:1 de um liderado de uma vez (ex.: ele saiu)
	CancelarDoColaborador(colaboradorID, usuarioID string) (int64, error)
}

type useCaseImpl struct {
	repo          Repositorio
	colaboradorUC colaborador.UseCase
	emailSvc      email.Servico
}

// NovoUseCase cria o UseCase de agendamentos.
func NovoUseCase(repo Repositorio, colaboradorUC colaborador.UseCase, emailSvc email.Servico) UseCase {
	return &useCaseImpl{repo: repo, colaboradorUC: colaboradorUC, emailSvc: emailSvc}
}

// avisarLiderado envia (best-effort, assíncrono) um e-mail ao liderado de um agendamento.
// Dormente se o SMTP não está configurado. `montar` recebe o nome do liderado e devolve
// (assunto, html).
func (uc *useCaseImpl) avisarLiderado(colaboradorID string, montar func(nomeLiderado string) (string, string)) {
	if uc.emailSvc == nil {
		return
	}
	col, err := uc.colaboradorUC.BuscarInternoPorId(colaboradorID)
	if err != nil || col.Email == "" {
		return
	}
	assunto, html := montar(col.Nome)
	go func() { _ = uc.emailSvc.EnviarHTML([]string{col.Email}, assunto, html) }()
}

// formato de saída/entrada amigável ao <input type=datetime-local>.
const formatoCurto = "2006-01-02T15:04"

// parsearDataHora aceita alguns formatos; interpreta no fuso LOCAL (loc=Local no DSN).
func parsearDataHora(s string) (time.Time, error) {
	for _, f := range []string{formatoCurto, "2006-01-02T15:04:05", time.RFC3339} {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("data/hora inválida (use AAAA-MM-DDTHH:MM)")
}

func (uc *useCaseImpl) Criar(usuarioID string, dto CriarAgendamentoDTO) (AgendamentoRespostaDTO, error) {
	// BuscarPorId valida a POSSE (o liderado tem de ser da estrutura do líder
	// logado) e já devolve o nome para a resposta. Sem posse → "não encontrado".
	col, err := uc.colaboradorUC.BuscarPorId(dto.ColaboradorID, usuarioID)
	if err != nil {
		return AgendamentoRespostaDTO{}, fmt.Errorf("liderado não encontrado")
	}

	dh, err := parsearDataHora(dto.DataHora)
	if err != nil {
		return AgendamentoRespostaDTO{}, err
	}

	rec := dto.Recorrencia
	if rec == "" {
		rec = RecNenhuma
	}

	// Fim da recorrência (opcional, "YYYY-MM-DD"). Só faz sentido para 1:1 recorrente.
	var repeteAte *time.Time
	if rec != RecNenhuma && dto.RepeteAte != "" {
		t, errData := time.ParseInLocation("2006-01-02", dto.RepeteAte, time.Local)
		if errData != nil {
			return AgendamentoRespostaDTO{}, fmt.Errorf("data de término inválida (use AAAA-MM-DD)")
		}
		repeteAte = &t
	}

	a := Agendamento{
		ID:            uuid.New().String(),
		UsuarioID:     usuarioID,
		ColaboradorID: dto.ColaboradorID,
		DataHora:      dh,
		Recorrencia:   rec,
		RepeteAte:     repeteAte,
		Ativo:         true,
		CriadoEm:      time.Now(),
	}
	if err := uc.repo.Criar(a); err != nil {
		return AgendamentoRespostaDTO{}, err
	}

	return AgendamentoRespostaDTO{
		ID:            a.ID,
		ColaboradorID: a.ColaboradorID,
		LideradoNome:  col.Nome,
		DataHora:      dh.Format(formatoCurto),
		Recorrencia:   rec,
		RepeteAte:     fmtRepeteAte(repeteAte),
	}, nil
}

// fmtRepeteAte formata o fim da recorrência como "YYYY-MM-DD" (vazio se for "para sempre").
func fmtRepeteAte(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func (uc *useCaseImpl) ListarPorUsuario(usuarioID string) ([]AgendamentoRespostaDTO, error) {
	lista, err := uc.repo.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, err
	}
	resp := make([]AgendamentoRespostaDTO, 0, len(lista))
	for _, a := range lista {
		resp = append(resp, AgendamentoRespostaDTO{
			ID:            a.ID,
			ColaboradorID: a.ColaboradorID,
			LideradoNome:  a.LideradoNome,
			DataHora:      a.DataHora.Format(formatoCurto),
			Recorrencia:   a.Recorrencia,
			RepeteAte:     fmtRepeteAte(a.RepeteAte),
		})
	}
	return resp, nil
}

func (uc *useCaseImpl) Deletar(id, usuarioID string) error {
	// Pega os dados ANTES de apagar (para avisar o liderado por e-mail).
	ag, ok, _ := uc.repo.BuscarPorId(id, usuarioID)
	if err := uc.repo.Deletar(id, usuarioID); err != nil {
		return err
	}
	if ok {
		uc.avisarLiderado(ag.ColaboradorID, email.TemplateAgendaCancelada)
	}
	return nil
}

// Reagendar valida a data/hora e muda o agendamento (apenas se for do gestor). Avisa o
// liderado da nova data por e-mail.
func (uc *useCaseImpl) Reagendar(id, usuarioID, dataHora string) error {
	dh, err := parsearDataHora(dataHora)
	if err != nil {
		return err
	}
	ok, err := uc.repo.Reagendar(id, usuarioID, dh)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("agendamento não encontrado")
	}
	if ag, found, _ := uc.repo.BuscarPorId(id, usuarioID); found {
		uc.avisarLiderado(ag.ColaboradorID, func(nome string) (string, string) {
			return email.TemplateAgendaRemarcada(nome, dh.Format("02/01 às 15:04"))
		})
	}
	return nil
}

// CancelarDoColaborador remove de uma vez todos os 1:1 agendados de um liderado — útil
// quando ele sai da empresa (em vez de cancelar um por um). Escopado ao gestor dono.
func (uc *useCaseImpl) CancelarDoColaborador(colaboradorID, usuarioID string) (int64, error) {
	return uc.repo.DeletarPorColaborador(colaboradorID, usuarioID)
}
