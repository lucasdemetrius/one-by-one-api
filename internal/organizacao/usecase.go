// Pacote: internal/organizacao
// Arquivo: usecase.go
// Descrição: Contém as regras de negócio do módulo de organização,
//            intermediando entre o Controller (HTTP) e o Repository (banco).
// Autor: OneByOne API
// Criado em: 2025

package organizacao

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"onebyone-api/pkg/storage"
)

// ErrAcessoNegado indica que o solicitante não é o dono (líder) da organização nem o RH do
// tenant dela. Mensagem genérica "não encontrada" → 404 no controller (não revela existência).
var ErrAcessoNegado = errors.New("organização não encontrada")

// UseCase define o contrato das operações de negócio do módulo de organização
type UseCase interface {
	// Criar valida os dados e persiste uma nova organização vinculada ao líder autenticado
	Criar(usuarioID string, dto CriarOrganizacaoDTO) (OrganizacaoRespostaDTO, error)
	// BuscarPorId retorna a organização SE o solicitante for o dono (líder) ou o RH do tenant
	BuscarPorId(id string, usuarioID string) (OrganizacaoRespostaDTO, error)
	// ListarPorUsuario retorna todas as organizações do líder autenticado
	ListarPorUsuario(usuarioID string) ([]OrganizacaoRespostaDTO, error)
	// Atualizar aplica as alterações (dono ou RH do tenant)
	Atualizar(id string, usuarioID string, dto AtualizarOrganizacaoDTO) (OrganizacaoRespostaDTO, error)
	// Deletar realiza a exclusão lógica da organização (dono ou RH do tenant)
	Deletar(id string, deletadoPor string) error
	// UploadFoto envia a foto para o S3 e persiste a chave (dono ou RH do tenant)
	UploadFoto(id string, usuarioID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (OrganizacaoRespostaDTO, error)
}

// useCaseImpl é a implementação concreta do UseCase de organização
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// armazenamento é o serviço S3 para upload e geração de URLs presignadas de fotos
	armazenamento storage.Armazenamento
}

// NovoUseCase cria e retorna uma nova instância do UseCase de organização
func NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento) UseCase {
	return &useCaseImpl{repo: repo, armazenamento: armazenamento}
}

// gerarFotoURL gera uma URL presignada para a foto da organização se a chave S3 estiver preenchida.
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

// Criar gera o UUID, vincula a organização ao líder autenticado e persiste no banco
func (uc *useCaseImpl) Criar(usuarioID string, dto CriarOrganizacaoDTO) (OrganizacaoRespostaDTO, error) {
	novaOrg := Organizacao{
		ID:         uuid.New().String(),
		UsuarioID:  usuarioID,
		TemplateID: dto.TemplateID,
		Nome:       dto.Nome,
		CriadoEm:   time.Now(),
	}

	criada, err := uc.repo.Criar(novaOrg)
	if err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("erro ao criar organização: %w", err)
	}
	return ParaRespostaDTO(criada, nil), nil
}

// BuscarPorId localiza uma organização ativa pelo UUID, validando a posse (dono OU RH do tenant)
func (uc *useCaseImpl) BuscarPorId(id string, usuarioID string) (OrganizacaoRespostaDTO, error) {
	org, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("organização não encontrada: %w", err)
	}
	if !uc.podeAgir(org.UsuarioID, usuarioID) {
		return OrganizacaoRespostaDTO{}, ErrAcessoNegado
	}
	return ParaRespostaDTO(org, uc.gerarFotoURL(org.FotoKey)), nil
}

// podeAgir resume a posse da Cadeia A: o ator é o dono direto (igualdade) OU o RH dono do
// gestor (tenant). Self-gating — para um não-RH o fallback nunca casa.
func (uc *useCaseImpl) podeAgir(donoUsuarioID, usuarioID string) bool {
	if donoUsuarioID == usuarioID {
		return true
	}
	ok, _ := uc.repo.GestorPertenceAoRH(donoUsuarioID, usuarioID)
	return ok
}

// ListarPorUsuario retorna todas as organizações ativas do líder convertidas para DTOs
func (uc *useCaseImpl) ListarPorUsuario(usuarioID string) ([]OrganizacaoRespostaDTO, error) {
	orgs, err := uc.repo.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar organizações: %w", err)
	}
	lista := make([]OrganizacaoRespostaDTO, 0, len(orgs))
	for _, o := range orgs {
		lista = append(lista, ParaRespostaDTO(o, uc.gerarFotoURL(o.FotoKey)))
	}
	return lista, nil
}

// Atualizar aplica apenas os campos informados no DTO preservando os demais
func (uc *useCaseImpl) Atualizar(id string, usuarioID string, dto AtualizarOrganizacaoDTO) (OrganizacaoRespostaDTO, error) {
	// Verifica que a organização existe e está ativa antes de modificar
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("organização não encontrada: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return OrganizacaoRespostaDTO{}, ErrAcessoNegado
	}

	if dto.Nome != "" {
		atual.Nome = dto.Nome
	}
	// TemplateID pode ser nil para remover ou um novo UUID para alterar
	if dto.TemplateID != nil {
		atual.TemplateID = dto.TemplateID
	}

	atualizada, err := uc.repo.Atualizar(atual)
	if err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("erro ao atualizar organização: %w", err)
	}
	return ParaRespostaDTO(atualizada, uc.gerarFotoURL(atualizada.FotoKey)), nil
}

// Deletar verifica a existência e delega a exclusão lógica ao repositório
func (uc *useCaseImpl) Deletar(id string, deletadoPor string) error {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return fmt.Errorf("organização não encontrada: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, deletadoPor) {
		return ErrAcessoNegado
	}
	return uc.repo.DeletarSoft(id, deletadoPor)
}

// UploadFoto envia o arquivo para o S3 e persiste a chave no banco de dados
func (uc *useCaseImpl) UploadFoto(id string, usuarioID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (OrganizacaoRespostaDTO, error) {
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("organização não encontrada: %w", err)
	}
	if !uc.podeAgir(atual.UsuarioID, usuarioID) {
		return OrganizacaoRespostaDTO{}, ErrAcessoNegado
	}

	ext := extensaoPorTipo(tipoConteudo)
	caminho := fmt.Sprintf("organizacoes/%s/foto%s", id, ext)
	chave := uc.armazenamento.ChaveCompleta(caminho)

	if err := uc.armazenamento.Upload(chave, arquivo, tamanho, tipoConteudo); err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("erro ao enviar foto: %w", err)
	}

	if err := uc.repo.AtualizarFoto(id, chave); err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("erro ao salvar chave da foto: %w", err)
	}

	atualizada, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return OrganizacaoRespostaDTO{}, fmt.Errorf("erro ao buscar organização após upload: %w", err)
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
