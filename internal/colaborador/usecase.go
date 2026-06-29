// Pacote: internal/colaborador
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de colaborador,
//            intermediando entre o Controller (HTTP) e o Repository (banco).
// Autor: OneByOne API
// Criado em: 2025

package colaborador

import (
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"onebyone-api/pkg/storage"
	"onebyone-api/pkg/texto"
)

// UseCase define o contrato das operações de negócio do módulo de colaborador.
//
// SEGURANÇA: todas as operações sobre um colaborador identificado por ID recebem
// o usuarioID do chamador (extraído do JWT no controller) e validam a POSSE antes
// de ler/escrever. O dono é o LÍDER (via equipe/organização — "Cadeia B"); o
// próprio liderado pode acessar só o seu registro (self). Ver PertenceAoLider.
type UseCase interface {
	// Criar valida e persiste um novo colaborador (na estrutura do líder logado)
	Criar(dto CriarColaboradorDTO, usuarioID string) (ColaboradorRespostaDTO, error)
	// ImportarLote cria vários liderados de uma vez (CSV) numa equipe do líder.
	// Valida linha a linha e agrega os erros — uma linha ruim não derruba o lote.
	ImportarLote(itens []ItemImportacaoDTO, organizacaoID, equipeID, usuarioID string) (ResultadoImportacaoDTO, error)
	// BuscarPorId localiza um colaborador (do líder dono OU do próprio liderado)
	BuscarPorId(id string, usuarioID string) (ColaboradorRespostaDTO, error)
	// BuscarPorUsuarioID localiza o colaborador vinculado a uma conta de usuário (liderado logado)
	BuscarPorUsuarioID(usuarioID string) (ColaboradorRespostaDTO, error)
	// ListarPorEquipe retorna os colaboradores de uma equipe do líder logado
	ListarPorEquipe(equipeID string, usuarioID string) ([]ColaboradorRespostaDTO, error)
	// ListarPorOrganizacao retorna os colaboradores de uma organização do líder logado
	ListarPorOrganizacao(organizacaoID string, usuarioID string) ([]ColaboradorRespostaDTO, error)
	// Atualizar aplica as alterações permitidas (só o líder dono; não mexe em usuario_id)
	Atualizar(id string, usuarioID string, dto AtualizarColaboradorDTO) (ColaboradorRespostaDTO, error)
	// Deletar realiza a exclusão lógica do colaborador (só o líder dono)
	Deletar(id string, usuarioID string) error
	// Desligar marca o liderado como inativo (saída da empresa/equipe). Só o líder dono.
	Desligar(id string, usuarioID string, desligadoEm time.Time) error
	// Reativar volta o liderado a ativo. Só o líder dono.
	Reativar(id string, usuarioID string) error
	// UploadFoto envia a foto para o S3 (líder dono OU o próprio liderado)
	UploadFoto(id string, usuarioID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (ColaboradorRespostaDTO, error)
	// PertenceAoLider expõe a checagem de posse para outros módulos (classificacao,
	// convite, agendamento) reutilizarem a "Cadeia B" (só o líder dono).
	PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error)
	// PodeAcessar é a checagem "líder dono OU o próprio liderado" — para módulos
	// onde o liderado também participa (ex.: conteúdo de tema no 1:1 ao vivo).
	PodeAcessar(colaboradorID, usuarioID string) (bool, error)
	// OrganizacaoPertenceAoLider expõe a posse da organização (reuso por módulos
	// que listam por organização, ex.: classificacao 9-box).
	OrganizacaoPertenceAoLider(organizacaoID, usuarioID string) (bool, error)
	// VincularConta amarra a conta de usuário ao colaborador. SÓ para o aceite de convite.
	VincularConta(colaboradorID, usuarioID string) error
	// BuscarInternoPorId lê o colaborador SEM checar posse. Uso EXCLUSIVO de
	// fluxos internos que têm a própria autorização (ex.: convite público via
	// token+código). NUNCA exponha isto diretamente em rota HTTP.
	BuscarInternoPorId(id string) (ColaboradorRespostaDTO, error)
}

// ErrAcessoNegado indica falta de posse. A mensagem é propositalmente "não
// encontrado" para o controller responder 404 e não revelar a existência do recurso.
var ErrAcessoNegado = errors.New("colaborador não encontrado")

// ErrEmailDuplicado indica que o líder já tem um liderado ativo com este e-mail.
// O controller mapeia para 409 (conflito).
var ErrEmailDuplicado = errors.New("já existe um liderado com este e-mail")

// ErrEmailDoGestor indica que o e-mail informado é o da conta do próprio gestor.
// Um liderado não pode usar o e-mail do líder. O controller mapeia para 409.
var ErrEmailDoGestor = errors.New("este e-mail é o da sua conta de gestor — use o e-mail do liderado")

// useCaseImpl é a implementação concreta do UseCase de colaborador
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// armazenamento é o serviço S3 para upload e geração de URLs presignadas de fotos
	armazenamento storage.Armazenamento
}

// NovoUseCase cria e retorna uma nova instância do UseCase de colaborador
func NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento) UseCase {
	return &useCaseImpl{repo: repo, armazenamento: armazenamento}
}

// gerarFotoURL gera uma URL presignada para a foto do colaborador se a chave S3 estiver preenchida.
func (uc *useCaseImpl) gerarFotoURL(fotoKey *string) *string {
	if fotoKey == nil || uc.armazenamento == nil {
		return nil
	}
	url, err := uc.armazenamento.GerarURLPresignada(*fotoKey, storage.ExpiracaoURLFoto)
	if err != nil {
		return nil
	}
	return &url
}

// podeAcessar diz se o usuário pode LER o colaborador: ou é o líder dono (Cadeia
// B, via equipe/organização) ou é o próprio liderado (self, colaborador.usuario_id).
func (uc *useCaseImpl) podeAcessar(col Colaborador, usuarioID string) (bool, error) {
	if col.UsuarioID != nil && *col.UsuarioID == usuarioID {
		return true, nil // o próprio liderado acessando seu registro
	}
	return uc.repo.PertenceAoLider(col.ID, usuarioID) // o líder dono
}

// Criar valida a posse da equipe/organização (devem ser do líder logado), valida
// a data de nascimento, gera UUID e persiste o novo colaborador. O vínculo de
// conta (usuario_id) NÃO é aceito aqui — isso só acontece no aceite de convite.
func (uc *useCaseImpl) Criar(dto CriarColaboradorDTO, usuarioID string) (ColaboradorRespostaDTO, error) {
	// Normaliza o e-mail (minúsculo + sem espaços) para gravar e comparar de forma canônica.
	dto.Email = texto.NormalizarEmail(dto.Email)

	// POSSE: a equipe e a organização informadas têm de ser do líder logado.
	okEquipe, err := uc.repo.EquipePertenceAoLider(dto.EquipeID, usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	okOrg, err := uc.repo.OrganizacaoPertenceAoLider(dto.OrganizacaoID, usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	if !okEquipe || !okOrg {
		return ColaboradorRespostaDTO{}, ErrAcessoNegado
	}

	// O e-mail não pode ser o da conta do próprio gestor (anti-sequestro de conta).
	ehDoGestor, err := uc.repo.EmailEhDoLider(dto.Email, usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	if ehDoGestor {
		return ColaboradorRespostaDTO{}, ErrEmailDoGestor
	}

	// UNICIDADE: o líder não pode ter dois liderados ativos com o mesmo e-mail.
	jaExiste, err := uc.repo.ExisteEmailNoLider(dto.Email, usuarioID, "")
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	if jaExiste {
		return ColaboradorRespostaDTO{}, ErrEmailDuplicado
	}

	// Converte a data de nascimento do formato string "YYYY-MM-DD" para time.Time
	var dataNascimento *time.Time
	if dto.DataNascimento != "" {
		data, err := time.Parse("2006-01-02", dto.DataNascimento)
		if err != nil {
			return ColaboradorRespostaDTO{}, fmt.Errorf("data de nascimento inválida — use o formato YYYY-MM-DD: %w", err)
		}
		dataNascimento = &data
	}

	novoColaborador := Colaborador{
		ID:             uuid.New().String(),
		UsuarioID:      nil, // vínculo de conta só pelo fluxo de convite
		OrganizacaoID:  dto.OrganizacaoID,
		EquipeID:       dto.EquipeID,
		TemplateID:     dto.TemplateID,
		Nome:           dto.Nome,
		Email:          dto.Email,
		Whatsapp:       dto.Whatsapp,
		DataNascimento: dataNascimento,
		CriadoEm:       time.Now(),
	}

	criado, err := uc.repo.Criar(novoColaborador)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("erro ao criar colaborador: %w", err)
	}
	return ParaRespostaDTO(criado, nil), nil
}

// ImportarLote cria vários liderados de uma vez numa equipe do líder (import CSV).
// Cada linha é validada (nome/e-mail) e criada reusando Criar — que já aplica posse,
// unicidade de e-mail por líder e a regra anti-e-mail-do-gestor. Linhas com problema
// (e-mail inválido, duplicado, etc.) viram erros no resultado, sem abortar o lote.
func (uc *useCaseImpl) ImportarLote(itens []ItemImportacaoDTO, organizacaoID, equipeID, usuarioID string) (ResultadoImportacaoDTO, error) {
	// Posse da equipe/organização-alvo: falha cedo e com clareza se não for do líder.
	okEquipe, err := uc.repo.EquipePertenceAoLider(equipeID, usuarioID)
	if err != nil {
		return ResultadoImportacaoDTO{}, err
	}
	okOrg, err := uc.repo.OrganizacaoPertenceAoLider(organizacaoID, usuarioID)
	if err != nil {
		return ResultadoImportacaoDTO{}, err
	}
	if !okEquipe || !okOrg {
		return ResultadoImportacaoDTO{}, ErrAcessoNegado
	}

	res := ResultadoImportacaoDTO{
		Criados: []ColaboradorRespostaDTO{},
		Erros:   []ErroImportacaoDTO{},
	}

	for i, item := range itens {
		linha := i + 1
		nome := strings.TrimSpace(item.Nome)
		email := texto.NormalizarEmail(item.Email)

		// Validação de formato da linha (vem de CSV, não passou pelo binding do Gin).
		if len([]rune(nome)) < 2 {
			res.Erros = append(res.Erros, ErroImportacaoDTO{Linha: linha, Nome: item.Nome, Email: email, Motivo: "nome muito curto (mín. 2 letras)"})
			continue
		}
		if _, err := mail.ParseAddress(email); err != nil {
			res.Erros = append(res.Erros, ErroImportacaoDTO{Linha: linha, Nome: nome, Email: item.Email, Motivo: "e-mail inválido"})
			continue
		}

		// Reusa toda a regra de negócio do Criar (posse + unicidade + anti-gestor).
		criado, err := uc.Criar(CriarColaboradorDTO{
			OrganizacaoID: organizacaoID,
			EquipeID:      equipeID,
			Nome:          nome,
			Email:         email,
		}, usuarioID)
		if err != nil {
			// Erros de NEGÓCIO (e-mail duplicado/do gestor) viram erro de linha e o
			// lote continua. Qualquer outro erro (infra, banco) NÃO pode ser mascarado
			// como "problema do dado" — aborta o lote para o controller responder 500.
			if errors.Is(err, ErrEmailDuplicado) || errors.Is(err, ErrEmailDoGestor) {
				res.Erros = append(res.Erros, ErroImportacaoDTO{Linha: linha, Nome: nome, Email: email, Motivo: motivoImportacao(err)})
				continue
			}
			return ResultadoImportacaoDTO{}, fmt.Errorf("erro ao importar (linha %d): %w", linha, err)
		}
		res.Criados = append(res.Criados, criado)
	}

	return res, nil
}

// motivoImportacao traduz os erros de negócio do Criar para uma mensagem amigável por linha.
func motivoImportacao(err error) string {
	switch {
	case errors.Is(err, ErrEmailDuplicado):
		return "e-mail já cadastrado para um liderado"
	case errors.Is(err, ErrEmailDoGestor):
		return "e-mail é o da sua conta de gestor"
	default:
		return "não foi possível criar"
	}
}

// BuscarPorId localiza um colaborador ativo pelo UUID, validando a posse
// (líder dono OU o próprio liderado). Sem posse → 404 (ErrAcessoNegado).
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (ColaboradorRespostaDTO, error) {
	col, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("colaborador não encontrado: %w", err)
	}
	pode, err := uc.podeAcessar(col, usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	if !pode {
		return ColaboradorRespostaDTO{}, ErrAcessoNegado
	}
	return ParaRespostaDTO(col, uc.gerarFotoURL(col.FotoKey)), nil
}

// BuscarPorUsuarioID localiza o colaborador vinculado à conta do liderado logado.
func (uc *useCaseImpl) BuscarPorUsuarioID(usuarioID string) (ColaboradorRespostaDTO, error) {
	col, err := uc.repo.BuscarPorUsuarioID(usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("colaborador não encontrado: %w", err)
	}
	return ParaRespostaDTO(col, uc.gerarFotoURL(col.FotoKey)), nil
}

// ListarPorEquipe retorna os colaboradores da equipe, desde que a equipe seja do
// líder logado (senão devolve "não encontrado" para não vazar PII de terceiros).
func (uc *useCaseImpl) ListarPorEquipe(equipeID string, usuarioID string) ([]ColaboradorRespostaDTO, error) {
	ok, err := uc.repo.EquipePertenceAoLider(equipeID, usuarioID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAcessoNegado
	}
	cols, err := uc.repo.ListarPorEquipe(equipeID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar colaboradores da equipe: %w", err)
	}
	lista := make([]ColaboradorRespostaDTO, 0, len(cols))
	for _, c := range cols {
		lista = append(lista, ParaRespostaDTO(c, uc.gerarFotoURL(c.FotoKey)))
	}
	return lista, nil
}

// ListarPorOrganizacao retorna os colaboradores da organização, desde que ela
// seja do líder logado (senão "não encontrado", para não vazar PII de terceiros).
func (uc *useCaseImpl) ListarPorOrganizacao(organizacaoID string, usuarioID string) ([]ColaboradorRespostaDTO, error) {
	ok, err := uc.repo.OrganizacaoPertenceAoLider(organizacaoID, usuarioID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAcessoNegado
	}
	cols, err := uc.repo.ListarPorOrganizacao(organizacaoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar colaboradores da organização: %w", err)
	}
	lista := make([]ColaboradorRespostaDTO, 0, len(cols))
	for _, c := range cols {
		lista = append(lista, ParaRespostaDTO(c, uc.gerarFotoURL(c.FotoKey)))
	}
	return lista, nil
}

// Atualizar aplica apenas os campos informados no DTO preservando os demais.
// Só o LÍDER dono pode atualizar. O campo usuario_id NÃO é alterável por aqui
// (vínculo de conta só pelo aceite de convite — evita sequestro de identidade);
// e se a equipe for trocada, a nova equipe também precisa ser do mesmo líder.
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarColaboradorDTO) (ColaboradorRespostaDTO, error) {
	// POSSE: só o líder dono.
	dono, err := uc.repo.PertenceAoLider(id, usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	if !dono {
		return ColaboradorRespostaDTO{}, ErrAcessoNegado
	}

	// Carrega o estado atual para permitir atualização parcial
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("colaborador não encontrado: %w", err)
	}

	if dto.Nome != "" {
		atual.Nome = dto.Nome
	}
	if dto.Email != "" {
		dto.Email = texto.NormalizarEmail(dto.Email)
	}
	if dto.Email != "" && dto.Email != atual.Email {
		// O novo e-mail não pode ser o da conta do gestor (anti-sequestro).
		ehDoGestor, err := uc.repo.EmailEhDoLider(dto.Email, usuarioID)
		if err != nil {
			return ColaboradorRespostaDTO{}, err
		}
		if ehDoGestor {
			return ColaboradorRespostaDTO{}, ErrEmailDoGestor
		}
		// Trocar de e-mail também respeita a unicidade (ignorando o próprio id).
		jaExiste, err := uc.repo.ExisteEmailNoLider(dto.Email, usuarioID, id)
		if err != nil {
			return ColaboradorRespostaDTO{}, err
		}
		if jaExiste {
			return ColaboradorRespostaDTO{}, ErrEmailDuplicado
		}
		atual.Email = dto.Email
	}
	if dto.EquipeID != "" {
		// A nova equipe também precisa pertencer ao líder.
		okEquipe, err := uc.repo.EquipePertenceAoLider(dto.EquipeID, usuarioID)
		if err != nil {
			return ColaboradorRespostaDTO{}, err
		}
		if !okEquipe {
			return ColaboradorRespostaDTO{}, ErrAcessoNegado
		}
		atual.EquipeID = dto.EquipeID
	}
	// dto.UsuarioID é IGNORADO de propósito (segurança — ver comentário acima).
	if dto.TemplateID != nil {
		atual.TemplateID = dto.TemplateID
	}
	if dto.Whatsapp != nil {
		atual.Whatsapp = dto.Whatsapp
	}
	if dto.DataNascimento != "" {
		// Valida e converte a nova data de nascimento
		data, err := time.Parse("2006-01-02", dto.DataNascimento)
		if err != nil {
			return ColaboradorRespostaDTO{}, fmt.Errorf("data de nascimento inválida — use o formato YYYY-MM-DD: %w", err)
		}
		atual.DataNascimento = &data
	}

	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("erro ao atualizar colaborador: %w", err)
	}
	return ParaRespostaDTO(atualizado, uc.gerarFotoURL(atualizado.FotoKey)), nil
}

// UploadFoto envia o arquivo para o S3 e persiste a chave no banco de dados.
// Permitido ao líder dono OU ao próprio liderado (self).
func (uc *useCaseImpl) UploadFoto(id string, usuarioID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (ColaboradorRespostaDTO, error) {
	col, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("colaborador não encontrado: %w", err)
	}
	pode, err := uc.podeAcessar(col, usuarioID)
	if err != nil {
		return ColaboradorRespostaDTO{}, err
	}
	if !pode {
		return ColaboradorRespostaDTO{}, ErrAcessoNegado
	}

	ext := extensaoPorTipo(tipoConteudo)
	caminho := fmt.Sprintf("colaboradores/%s/foto%s", id, ext)
	chave := uc.armazenamento.ChaveCompleta(caminho)

	if err := uc.armazenamento.Upload(chave, arquivo, tamanho, tipoConteudo); err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("erro ao enviar foto: %w", err)
	}

	if err := uc.repo.AtualizarFoto(id, chave); err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("erro ao salvar chave da foto: %w", err)
	}

	atualizado, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("erro ao buscar colaborador após upload: %w", err)
	}

	return ParaRespostaDTO(atualizado, uc.gerarFotoURL(atualizado.FotoKey)), nil
}

func extensaoPorTipo(tipoConteudo string) string {
	switch tipoConteudo {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// Deletar valida a posse (só o líder dono) e delega a exclusão lógica. O próprio
// usuarioID do líder é gravado em deletado_por.
func (uc *useCaseImpl) Deletar(id string, usuarioID string) error {
	dono, err := uc.repo.PertenceAoLider(id, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return uc.repo.DeletarSoft(id, usuarioID)
}

// Desligar valida a posse (só o líder dono) e marca o liderado como inativo.
func (uc *useCaseImpl) Desligar(id string, usuarioID string, desligadoEm time.Time) error {
	dono, err := uc.repo.PertenceAoLider(id, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return uc.repo.Desligar(id, desligadoEm)
}

// Reativar valida a posse (só o líder dono) e reativa o liderado.
func (uc *useCaseImpl) Reativar(id string, usuarioID string) error {
	dono, err := uc.repo.PertenceAoLider(id, usuarioID)
	if err != nil {
		return err
	}
	if !dono {
		return ErrAcessoNegado
	}
	return uc.repo.Reativar(id)
}

// PertenceAoLider expõe a checagem de posse da Cadeia B para outros módulos.
func (uc *useCaseImpl) PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error) {
	return uc.repo.PertenceAoLider(colaboradorID, usuarioLiderID)
}

// OrganizacaoPertenceAoLider expõe a posse da organização para reuso por módulos.
func (uc *useCaseImpl) OrganizacaoPertenceAoLider(organizacaoID, usuarioID string) (bool, error) {
	return uc.repo.OrganizacaoPertenceAoLider(organizacaoID, usuarioID)
}

// VincularConta amarra a conta de usuário ao colaborador. SÓ para o aceite de
// convite — o aceite já provou a posse via token + código, então não revalida.
func (uc *useCaseImpl) VincularConta(colaboradorID, usuarioID string) error {
	// Troca de empresa: solta os vínculos antigos desta conta ANTES de amarrar o novo, para a
	// conta pertencer só a ESTE colaborador. Quem muda de empresa perde o acesso ao 1:1 da
	// anterior (o histórico fica com o gestor de lá) — isolamento/privacidade entre empresas.
	if _, err := uc.repo.DesvincularOutrasContas(usuarioID, colaboradorID); err != nil {
		return err
	}
	return uc.repo.VincularUsuario(colaboradorID, usuarioID)
}

// PodeAcessar diz se o usuário pode acessar o colaborador (líder dono OU o
// próprio liderado). Usado por módulos da Cadeia B onde o liderado também
// participa (ex.: conteúdo de tema do 1:1 ao vivo). Não existe → sem acesso.
func (uc *useCaseImpl) PodeAcessar(colaboradorID, usuarioID string) (bool, error) {
	col, err := uc.repo.BuscarPorId(colaboradorID)
	if err != nil {
		return false, nil
	}
	return uc.podeAcessar(col, usuarioID)
}

// BuscarInternoPorId lê o colaborador sem checar posse (uso interno confiável).
func (uc *useCaseImpl) BuscarInternoPorId(id string) (ColaboradorRespostaDTO, error) {
	col, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return ColaboradorRespostaDTO{}, fmt.Errorf("colaborador não encontrado: %w", err)
	}
	return ParaRespostaDTO(col, uc.gerarFotoURL(col.FotoKey)), nil
}
