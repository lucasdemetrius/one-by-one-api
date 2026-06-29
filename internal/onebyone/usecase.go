// Pacote: internal/onebyone
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de one-on-one, incluindo
//            a documentação completa da regra de herança de template e o método
//            ResolverTemplate utilizado ao abrir o formulário de uma reunião.
// Autor: OneByOne API
// Criado em: 2025

package onebyone

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"onebyone-api/internal/colaborador"
)

// ErrAcessoNegado: falta de posse (a reunião não é do líder logado). Mensagem
// "não encontrado" → o controller responde 404 e não revela a existência.
var ErrAcessoNegado = errors.New("one-on-one não encontrado")

// PosseColaborador é o pedaço do colaborador.UseCase de que o encerrar precisa:
// checar a posse do gestor (PertenceAoLider) e obter a estrutura (organização/equipe)
// do liderado para preencher os FKs ao gravar o 1:1 realizado. Reusa a Cadeia B de
// posse em vez de duplicar SQL (ver CLAUDE.md §7.1).
type PosseColaborador interface {
	PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error)
	BuscarPorId(id string, usuarioID string) (colaborador.ColaboradorRespostaDTO, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// REGRA DE NEGÓCIO: HERANÇA DE TEMPLATE
// ═══════════════════════════════════════════════════════════════════════════════
//
// Quando um one-on-one é "aberto" (ou seja, quando o usuário inicia o preenchimento
// e o módulo registrooneaone cria um novo registro), o sistema precisa determinar
// qual template de formulário será utilizado.
//
// A resolução segue esta ordem de prioridade (do mais específico ao mais genérico):
//
//   1. tb_colaboradores.template_id
//      → Se o colaborador tiver um template exclusivo configurado, usa esse.
//        Representa a configuração mais específica — o líder personalizou o formulário
//        individualmente para aquele colaborador.
//
//   2. tb_equipes.template_id
//      → Se o colaborador NÃO tiver template próprio, mas a sua equipe tiver um
//        template configurado, usa o da equipe.
//        Permite que times com necessidades específicas tenham um padrão diferente
//        do restante da organização.
//
//   3. tb_organizacoes.template_id
//      → Se nem o colaborador nem a equipe tiverem template, usa o da organização.
//        Esse é o padrão mais comum — um formulário único para toda a empresa.
//
//   4. Template padrão do líder (fallback final)
//      → Se nenhum dos níveis acima estiver configurado, o sistema busca o primeiro
//        template criado pelo líder (SELECT ... ORDER BY criado_em ASC LIMIT 1).
//        Esse é o template "default" implícito — o líder não precisa marcá-lo
//        explicitamente como padrão; basta ser o mais antigo.
//
// A implementação SQL desta lógica está em oneaone/repository.go → ResolverTemplateID(),
// que usa COALESCE para aplicar a prioridade em uma única query eficiente.
//
// Se nenhum template for encontrado em nenhum nível, retorna erro orientando o líder
// a configurar pelo menos um template antes de abrir reuniões.
// ═══════════════════════════════════════════════════════════════════════════════

// UseCase define o contrato das operações de negócio do módulo de one-on-one
type UseCase interface {
	// Criar valida os dados e persiste uma nova reunião one-on-one agendada
	Criar(usuarioID string, dto CriarOneByOneDTO) (OneByOneRespostaDTO, error)
	// Encerrar registra um 1:1 como REALIZADO (cria a linha no livro-razão). Idempotente
	// por dia. Só o gestor dono do colaborador (Cadeia B de posse).
	Encerrar(usuarioID string, dto EncerrarOneByOneDTO) (OneByOneRespostaDTO, error)
	// BuscarPorId localiza uma reunião ativa pelo UUID (só do líder dono)
	BuscarPorId(id string, usuarioID string) (OneByOneRespostaDTO, error)
	// ListarPorUsuario retorna todas as reuniões do líder autenticado
	ListarPorUsuario(usuarioID string) ([]OneByOneRespostaDTO, error)
	// Atualizar aplica as alterações permitidas em uma reunião (só do líder dono)
	Atualizar(id string, usuarioID string, dto AtualizarOneByOneDTO) (OneByOneRespostaDTO, error)
	// Deletar realiza a exclusão lógica da reunião (só do líder dono)
	Deletar(id string, usuarioID string) error
	// ResolverTemplate determina qual template usar ao abrir esta reunião,
	// seguindo a regra de herança documentada acima neste arquivo.
	ResolverTemplate(oneaoneID string) (string, error)
	// PertenceAoUsuario diz se a reunião é do líder informado (Cadeia A). Reuso
	// por registroonebyone/valorregistro para checar posse via o onebyone pai.
	PertenceAoUsuario(oneaoneID, usuarioID string) (bool, error)
}

// useCaseImpl é a implementação concreta do UseCase de one-on-one
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// colabUC resolve posse e estrutura (organização/equipe) do liderado ao encerrar
	colabUC PosseColaborador
}

// NovoUseCase cria e retorna uma nova instância do UseCase de one-on-one
func NovoUseCase(repo Repositorio, colabUC PosseColaborador) UseCase {
	return &useCaseImpl{repo: repo, colabUC: colabUC}
}

// Criar valida a data agendada, gera UUID e persiste a nova reunião com status AGENDADO
func (uc *useCaseImpl) Criar(usuarioID string, dto CriarOneByOneDTO) (OneByOneRespostaDTO, error) {
	// Converte a data agendada do formato string "YYYY-MM-DD" para time.Time
	dataAgendada, err := time.Parse("2006-01-02", dto.DataAgendada)
	if err != nil {
		return OneByOneRespostaDTO{}, fmt.Errorf("data agendada inválida — use o formato YYYY-MM-DD: %w", err)
	}

	// Define NENHUMA como recorrência padrão caso não seja informada
	recorrencia := dto.Recorrencia
	if recorrencia == "" {
		recorrencia = "NENHUMA"
	}

	novaReuniao := OneByOne{
		ID:            uuid.New().String(),
		UsuarioID:     usuarioID,
		OrganizacaoID: dto.OrganizacaoID,
		EquipeID:      dto.EquipeID,
		ColaborID:     dto.ColaborID,
		Recorrencia:   recorrencia,
		Status:        "AGENDADO", // toda reunião começa com status AGENDADO
		DataAgendada:  dataAgendada,
		CriadoEm:      time.Now(),
	}

	criada, err := uc.repo.Criar(novaReuniao)
	if err != nil {
		return OneByOneRespostaDTO{}, fmt.Errorf("erro ao criar one-on-one: %w", err)
	}
	return ParaRespostaDTO(criada), nil
}

// Encerrar registra um 1:1 como REALIZADO no livro-razão (tb_onebyone). Como o 1:1 ao
// vivo é por colaborador (não há reunião pré-criada), aqui CRIAMOS a linha já realizada.
// Posse: só o gestor dono do liderado (Cadeia B). Idempotente: se já existe um realizado
// do mesmo colaborador hoje, devolve o existente (não duplica a métrica).
func (uc *useCaseImpl) Encerrar(usuarioID string, dto EncerrarOneByOneDTO) (OneByOneRespostaDTO, error) {
	// 1) Posse do gestor sobre o liderado (Cadeia B). Recurso alheio → 404.
	dono, err := uc.colabUC.PertenceAoLider(dto.ColaborID, usuarioID)
	if err != nil || !dono {
		return OneByOneRespostaDTO{}, ErrAcessoNegado
	}

	// 2) Estrutura (organização/equipe) do liderado para preencher os FKs NOT NULL.
	col, err := uc.colabUC.BuscarPorId(dto.ColaborID, usuarioID)
	if err != nil {
		return OneByOneRespostaDTO{}, ErrAcessoNegado
	}

	agora := time.Now()

	// 3) Idempotência por dia: reaproveita o realizado de hoje, se houver.
	if existente, ok, _ := uc.repo.BuscarRealizadoNoDia(dto.ColaborID, agora); ok {
		return ParaRespostaDTO(existente), nil
	}

	// 4) Grava a linha já como REALIZADO, datada de agora.
	reuniao := OneByOne{
		ID:            uuid.New().String(),
		UsuarioID:     usuarioID,
		OrganizacaoID: col.OrganizacaoID,
		EquipeID:      col.EquipeID,
		ColaborID:     dto.ColaborID,
		Recorrencia:   "NENHUMA",
		Status:        "REALIZADO",
		RealizadoEm:   &agora,
		DataAgendada:  agora, // coluna DATE — guarda só a data de hoje
		CriadoEm:      agora,
	}
	criada, err := uc.repo.Criar(reuniao)
	if err != nil {
		return OneByOneRespostaDTO{}, fmt.Errorf("erro ao encerrar one-on-one: %w", err)
	}
	return ParaRespostaDTO(criada), nil
}

// BuscarPorId localiza uma reunião ativa pelo UUID, validando a posse (só o
// líder dono). Sem posse → ErrAcessoNegado (404 no controller).
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (OneByOneRespostaDTO, error) {
	reuniao, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return OneByOneRespostaDTO{}, fmt.Errorf("one-on-one não encontrado: %w", err)
	}
	if !uc.podeAgir(reuniao.UsuarioID, usuarioID) {
		return OneByOneRespostaDTO{}, ErrAcessoNegado
	}
	return ParaRespostaDTO(reuniao), nil
}

// PertenceAoUsuario confere se a reunião é acessível pelo ator: dono direto (Cadeia A —
// usuario_id) OU RH dono do gestor da reunião (escopo do tenant). Como registroonebyone e
// valorregistro herdam a posse por aqui, eles passam a aceitar o RH automaticamente.
func (uc *useCaseImpl) PertenceAoUsuario(oneaoneID, usuarioID string) (bool, error) {
	reuniao, err := uc.repo.BuscarPorId(oneaoneID)
	if err != nil {
		return false, nil
	}
	if reuniao.UsuarioID == usuarioID {
		return true, nil
	}
	return uc.repo.GestorPertenceAoRH(reuniao.UsuarioID, usuarioID)
}

// podeAgir resume a posse da Cadeia A: o ator é o dono direto (igualdade) OU o RH dono do
// gestor (tenant). Igualdade primeiro; o fallback RH só consulta o banco quando necessário
// e é self-gating — para um não-RH nenhum gestor tem rh_id igual ao dele.
func (uc *useCaseImpl) podeAgir(donoUsuarioID, usuarioID string) bool {
	if donoUsuarioID == usuarioID {
		return true
	}
	ok, _ := uc.repo.GestorPertenceAoRH(donoUsuarioID, usuarioID)
	return ok
}

// ListarPorUsuario retorna todas as reuniões ativas do líder convertidas para DTOs
func (uc *useCaseImpl) ListarPorUsuario(usuarioID string) ([]OneByOneRespostaDTO, error) {
	reunioes, err := uc.repo.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar one-on-ones: %w", err)
	}
	return ParaListaRespostaDTO(reunioes), nil
}

// Atualizar aplica apenas os campos informados no DTO (status, recorrência, data)
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarOneByOneDTO) (OneByOneRespostaDTO, error) {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return OneByOneRespostaDTO{}, fmt.Errorf("one-on-one não encontrado: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return OneByOneRespostaDTO{}, ErrAcessoNegado
	}

	if dto.Status != "" {
		atual.Status = dto.Status
		// Mantém realizado_em coerente com o status: ao virar REALIZADO carimba a data
		// (se ainda não tinha); ao sair de REALIZADO, limpa.
		if dto.Status == "REALIZADO" {
			if atual.RealizadoEm == nil {
				agora := time.Now()
				atual.RealizadoEm = &agora
			}
		} else {
			atual.RealizadoEm = nil
		}
	}
	if dto.Recorrencia != "" {
		atual.Recorrencia = dto.Recorrencia
	}
	if dto.DataAgendada != "" {
		// Valida e converte a nova data agendada
		novaData, err := time.Parse("2006-01-02", dto.DataAgendada)
		if err != nil {
			return OneByOneRespostaDTO{}, fmt.Errorf("data agendada inválida — use o formato YYYY-MM-DD: %w", err)
		}
		atual.DataAgendada = novaData
	}

	atualizada, err := uc.repo.Atualizar(atual)
	if err != nil {
		return OneByOneRespostaDTO{}, fmt.Errorf("erro ao atualizar one-on-one: %w", err)
	}
	return ParaRespostaDTO(atualizada), nil
}

// Deletar valida a posse (só o líder dono) e delega a exclusão lógica.
func (uc *useCaseImpl) Deletar(id string, usuarioID string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("one-on-one não encontrado: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return ErrAcessoNegado
	}
	return uc.repo.DeletarSoft(id, usuarioID)
}

// ResolverTemplate determina qual template aplicar ao abrir esta reunião.
// Delega a lógica de COALESCE SQL ao repositório e retorna o UUID do template resolvido.
// Ver documentação completa da regra de herança no bloco de comentário no topo deste arquivo.
func (uc *useCaseImpl) ResolverTemplate(oneaoneID string) (string, error) {
	templateID, err := uc.repo.ResolverTemplateID(oneaoneID)
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o template da reunião: %w", err)
	}
	return templateID, nil
}
