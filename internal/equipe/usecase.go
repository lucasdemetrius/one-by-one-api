// Pacote: internal/equipe
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de equipe,
//            intermediando entre o Controller (HTTP) e o Repository (banco).
// Autor: OneByOne API
// Criado em: 2025

package equipe

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"onebyone-api/pkg/storage"
)

// ErrAcessoNegado indica que o solicitante não é o dono (líder) da equipe/organização nem o
// RH do tenant. Mensagem genérica "não encontrada" → 404 no controller (não revela existência).
var ErrAcessoNegado = errors.New("equipe não encontrada")

// normalizarNome deixa o nome comparável para a regra de unicidade: minúsculas
// e sem espaços nas pontas (assim "Vendas", "vendas" e " VENDAS " colidem).
func normalizarNome(nome string) string {
	return strings.ToLower(strings.TrimSpace(nome))
}

// UseCase define o contrato das operações de negócio do módulo de equipe
type UseCase interface {
	// Criar valida os dados e persiste uma nova equipe vinculada ao líder autenticado
	Criar(usuarioID string, dto CriarEquipeDTO) (EquipeRespostaDTO, error)
	// BuscarPorId retorna a equipe SE o solicitante for o dono (líder) ou o RH do tenant
	BuscarPorId(id string, usuarioID string) (EquipeRespostaDTO, error)
	// ListarPorUsuario retorna todas as equipes do líder autenticado
	ListarPorUsuario(usuarioID string) ([]EquipeRespostaDTO, error)
	// ListarPorOrganizacao retorna as equipes de uma organização (se for do ator ou do RH do tenant)
	ListarPorOrganizacao(organizacaoID string, usuarioID string) ([]EquipeRespostaDTO, error)
	// Atualizar aplica as alterações (dono ou RH do tenant)
	Atualizar(id string, usuarioID string, dto AtualizarEquipeDTO) (EquipeRespostaDTO, error)
	// Deletar realiza a exclusão lógica da equipe (dono ou RH do tenant)
	Deletar(id string, deletadoPor string) error
	// UploadFoto envia a foto para o S3 e persiste a chave (dono ou RH do tenant)
	UploadFoto(id string, usuarioID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (EquipeRespostaDTO, error)
}

// useCaseImpl é a implementação concreta do UseCase de equipe
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// armazenamento é o serviço S3 para upload e geração de URLs presignadas de fotos
	armazenamento storage.Armazenamento
}

// NovoUseCase cria e retorna uma nova instância do UseCase de equipe
func NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento) UseCase {
	return &useCaseImpl{repo: repo, armazenamento: armazenamento}
}

// gerarFotoURL gera uma URL presignada para a foto da equipe se a chave S3 estiver preenchida.
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

// Criar gera o UUID, vincula a equipe ao líder e à organização informada e persiste no banco
func (uc *useCaseImpl) Criar(usuarioID string, dto CriarEquipeDTO) (EquipeRespostaDTO, error) {
	// UNICIDADE: o líder não pode ter duas equipes com o mesmo nome (case-insensitive).
	jaExiste, err := uc.repo.ExistePorNome(usuarioID, normalizarNome(dto.Nome), "")
	if err != nil {
		return EquipeRespostaDTO{}, err
	}
	if jaExiste {
		return EquipeRespostaDTO{}, fmt.Errorf("já existe uma equipe com este nome")
	}

	novaEquipe := Equipe{
		ID:            uuid.New().String(),
		UsuarioID:     usuarioID,
		OrganizacaoID: dto.OrganizacaoID,
		TemplateID:    dto.TemplateID,
		Nome:          strings.TrimSpace(dto.Nome),
		CriadoEm:      time.Now(),
	}

	criada, err := uc.repo.Criar(novaEquipe)
	if err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("erro ao criar equipe: %w", err)
	}
	return ParaRespostaDTO(criada, nil), nil
}

// BuscarPorId localiza uma equipe ativa pelo UUID, validando a posse (dono OU RH do tenant)
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (EquipeRespostaDTO, error) {
	e, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("equipe não encontrada: %w", err)
	}
	if !uc.podeAgir(e.UsuarioID, usuarioID) {
		return EquipeRespostaDTO{}, ErrAcessoNegado
	}
	return ParaRespostaDTO(e, uc.gerarFotoURL(e.FotoKey)), nil
}

// podeAgir resume a posse da Cadeia A: dono direto (igualdade) OU RH dono do gestor (tenant).
// Self-gating — para um não-RH o fallback nunca casa.
func (uc *useCaseImpl) podeAgir(donoUsuarioID, usuarioID string) bool {
	if donoUsuarioID == usuarioID {
		return true
	}
	ok, _ := uc.repo.GestorPertenceAoRH(donoUsuarioID, usuarioID)
	return ok
}

// ListarPorUsuario retorna todas as equipes ativas do líder convertidas para DTOs
func (uc *useCaseImpl) ListarPorUsuario(usuarioID string) ([]EquipeRespostaDTO, error) {
	equipes, err := uc.repo.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar equipes: %w", err)
	}
	lista := make([]EquipeRespostaDTO, 0, len(equipes))
	for _, e := range equipes {
		lista = append(lista, ParaRespostaDTO(e, uc.gerarFotoURL(e.FotoKey)))
	}
	return lista, nil
}

// ListarPorOrganizacao retorna todas as equipes ativas de uma organização
func (uc *useCaseImpl) ListarPorOrganizacao(organizacaoID string, usuarioID string) ([]EquipeRespostaDTO, error) {
	// Posse: a organização tem de ser do ator (líder dono) OU do tenant do RH. Sem isso,
	// qualquer logado listaria as equipes de qualquer empresa.
	ok, err := uc.repo.OrganizacaoPertenceAoAtor(organizacaoID, usuarioID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAcessoNegado
	}
	equipes, err := uc.repo.ListarPorOrganizacao(organizacaoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar equipes da organização: %w", err)
	}
	lista := make([]EquipeRespostaDTO, 0, len(equipes))
	for _, e := range equipes {
		lista = append(lista, ParaRespostaDTO(e, uc.gerarFotoURL(e.FotoKey)))
	}
	return lista, nil
}

// Atualizar aplica apenas os campos informados no DTO preservando os demais
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarEquipeDTO) (EquipeRespostaDTO, error) {
	// Verifica que a equipe existe e está ativa antes de modificar
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("equipe não encontrada: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return EquipeRespostaDTO{}, ErrAcessoNegado
	}

	// Renomear: ajuste só de caixa/espaço é permitido (e re-persistido); mudança
	// real de nome também passa pela unicidade por líder (ignorando a própria equipe).
	if novoNome := strings.TrimSpace(dto.Nome); novoNome != "" && novoNome != atual.Nome {
		if normalizarNome(novoNome) != normalizarNome(atual.Nome) {
			jaExiste, err := uc.repo.ExistePorNome(atual.UsuarioID, normalizarNome(novoNome), id)
			if err != nil {
				return EquipeRespostaDTO{}, err
			}
			if jaExiste {
				return EquipeRespostaDTO{}, fmt.Errorf("já existe uma equipe com este nome")
			}
		}
		atual.Nome = novoNome
	}
	if dto.TemplateID != nil {
		atual.TemplateID = dto.TemplateID
	}

	atualizada, err := uc.repo.Atualizar(atual)
	if err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("erro ao atualizar equipe: %w", err)
	}
	return ParaRespostaDTO(atualizada, uc.gerarFotoURL(atualizada.FotoKey)), nil
}

// Deletar verifica a existência e delega a exclusão lógica ao repositório
func (uc *useCaseImpl) Deletar(id string, deletadoPor string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("equipe não encontrada: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, deletadoPor) {
		return ErrAcessoNegado
	}
	return uc.repo.DeletarSoft(id, deletadoPor)
}

// UploadFoto envia o arquivo para o S3 e persiste a chave no banco de dados
func (uc *useCaseImpl) UploadFoto(id string, usuarioID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (EquipeRespostaDTO, error) {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("equipe não encontrada: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return EquipeRespostaDTO{}, ErrAcessoNegado
	}

	ext := extensaoPorTipo(tipoConteudo)
	caminho := fmt.Sprintf("equipes/%s/foto%s", id, ext)
	chave := uc.armazenamento.ChaveCompleta(caminho)

	if err := uc.armazenamento.Upload(chave, arquivo, tamanho, tipoConteudo); err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("erro ao enviar foto: %w", err)
	}

	if err := uc.repo.AtualizarFoto(id, chave); err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("erro ao salvar chave da foto: %w", err)
	}

	atualizada, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return EquipeRespostaDTO{}, fmt.Errorf("erro ao buscar equipe após upload: %w", err)
	}

	return ParaRespostaDTO(atualizada, uc.gerarFotoURL(atualizada.FotoKey)), nil
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
