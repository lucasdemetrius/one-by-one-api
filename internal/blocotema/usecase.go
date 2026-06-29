// Pacote: internal/blocotema
// Arquivo: usecase.go
// Descrição: Regras de negócio dos blocos de tema. Cria blocos de texto/link/marco,
//            faz upload de imagens para o S3 e devolve os blocos com URLs
//            presignadas. Valida o colaborador via o módulo colaborador.
// Autor: OneByOne API
// Criado em: 2025

package blocotema

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"onebyone-api/internal/colaborador"
	"onebyone-api/pkg/storage"
)

// ErrAcessoNegado: falta de posse. Mensagem "não encontrado" → controller responde 404.
var ErrAcessoNegado = errors.New("conteúdo não encontrado")

// UseCase define as operações de negócio dos blocos de tema. Todas recebem o
// usuarioID do chamador e validam a posse (líder dono OU o próprio liderado,
// pois no 1:1 ao vivo os dois editam o conteúdo do tema).
type UseCase interface {
	Listar(colaboradorID, tema, usuarioID string) ([]BlocoRespostaDTO, error)
	ListarTodos(colaboradorID, usuarioID string) ([]BlocoRespostaDTO, error)
	Criar(colaboradorID string, dto CriarBlocoDTO, usuarioID string) (BlocoRespostaDTO, error)
	CriarImagem(colaboradorID, tema, legenda string, arquivo io.Reader, tamanho int64, tipoConteudo, usuarioID string) (BlocoRespostaDTO, error)
	Deletar(colaboradorID, blocoID, usuarioID string) error
}

type useCaseImpl struct {
	repo          Repositorio
	armazenamento storage.Armazenamento
	colaboradorUC colaborador.UseCase
}

// NovoUseCase cria o UseCase de blocos de tema com as dependências injetadas.
func NovoUseCase(repo Repositorio, armazenamento storage.Armazenamento, colaboradorUC colaborador.UseCase) UseCase {
	return &useCaseImpl{repo: repo, armazenamento: armazenamento, colaboradorUC: colaboradorUC}
}

// formatarData converte *time.Time para "YYYY-MM-DD" (ou nil).
func formatarData(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

// parsearData converte "YYYY-MM-DD" para *time.Time (nil se vazio/ inválido).
func parsearData(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	// Guarda ao meio-dia para que conversões de fuso (±h) não virem o dia.
	t = t.Add(12 * time.Hour)
	return &t
}

// gerarImagemURL produz a URL presignada da imagem de um bloco (nil se sem imagem).
func (uc *useCaseImpl) gerarImagemURL(imagemKey *string) *string {
	if imagemKey == nil || uc.armazenamento == nil {
		return nil
	}
	url, err := uc.armazenamento.GerarURLPresignada(*imagemKey, storage.ExpiracaoURLFoto)
	if err != nil {
		return nil
	}
	return &url
}

// paraDTO converte a entidade em DTO de resposta, já com a URL da imagem.
func (uc *useCaseImpl) paraDTO(b BlocoTema) BlocoRespostaDTO {
	return BlocoRespostaDTO{
		ID:         b.ID,
		Tema:       b.Tema,
		Tipo:       b.Tipo,
		Texto:      b.Texto,
		URL:        b.URL,
		ImagemURL:  uc.gerarImagemURL(b.ImagemKey),
		DataInicio: formatarData(b.DataInicio),
		DataFim:    formatarData(b.DataFim),
		Ordem:      b.Ordem,
	}
}

// garantirPosse confere se o usuário pode acessar o conteúdo do colaborador.
func (uc *useCaseImpl) garantirPosse(colaboradorID, usuarioID string) error {
	pode, err := uc.colaboradorUC.PodeAcessar(colaboradorID, usuarioID)
	if err != nil {
		return err
	}
	if !pode {
		return ErrAcessoNegado
	}
	return nil
}

func (uc *useCaseImpl) ListarTodos(colaboradorID, usuarioID string) ([]BlocoRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return nil, err
	}
	blocos, err := uc.repo.ListarTodos(colaboradorID)
	if err != nil {
		return nil, err
	}
	lista := make([]BlocoRespostaDTO, 0, len(blocos))
	for _, b := range blocos {
		lista = append(lista, uc.paraDTO(b))
	}
	return lista, nil
}

func (uc *useCaseImpl) Listar(colaboradorID, tema, usuarioID string) ([]BlocoRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return nil, err
	}
	blocos, err := uc.repo.Listar(colaboradorID, tema)
	if err != nil {
		return nil, err
	}
	lista := make([]BlocoRespostaDTO, 0, len(blocos))
	for _, b := range blocos {
		lista = append(lista, uc.paraDTO(b))
	}
	return lista, nil
}

func (uc *useCaseImpl) Criar(colaboradorID string, dto CriarBlocoDTO, usuarioID string) (BlocoRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return BlocoRespostaDTO{}, err
	}

	ordem, _ := uc.repo.ProximaOrdem(colaboradorID, dto.Tema)

	bloco := BlocoTema{
		ID:            uuid.New().String(),
		ColaboradorID: colaboradorID,
		Tema:          dto.Tema,
		Tipo:          dto.Tipo,
		Texto:         dto.Texto,
		URL:           dto.URL,
		DataInicio:    parsearData(dto.DataInicio),
		DataFim:       parsearData(dto.DataFim),
		Ordem:         ordem,
		CriadoEm:      time.Now(),
	}

	criado, err := uc.repo.Criar(bloco)
	if err != nil {
		return BlocoRespostaDTO{}, err
	}
	return uc.paraDTO(criado), nil
}

func (uc *useCaseImpl) CriarImagem(colaboradorID, tema, legenda string, arquivo io.Reader, tamanho int64, tipoConteudo, usuarioID string) (BlocoRespostaDTO, error) {
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return BlocoRespostaDTO{}, err
	}

	id := uuid.New().String()
	ext := extensaoPorTipo(tipoConteudo)
	caminho := fmt.Sprintf("temas/%s/%s%s", colaboradorID, id, ext)
	chave := uc.armazenamento.ChaveCompleta(caminho)

	if err := uc.armazenamento.Upload(chave, arquivo, tamanho, tipoConteudo); err != nil {
		return BlocoRespostaDTO{}, fmt.Errorf("erro ao enviar imagem: %w", err)
	}

	ordem, _ := uc.repo.ProximaOrdem(colaboradorID, tema)
	var texto *string
	if legenda != "" {
		texto = &legenda
	}

	bloco := BlocoTema{
		ID:            id,
		ColaboradorID: colaboradorID,
		Tema:          tema,
		Tipo:          TipoImagem,
		Texto:         texto,
		ImagemKey:     &chave,
		Ordem:         ordem,
		CriadoEm:      time.Now(),
	}

	criado, err := uc.repo.Criar(bloco)
	if err != nil {
		return BlocoRespostaDTO{}, err
	}
	return uc.paraDTO(criado), nil
}

func (uc *useCaseImpl) Deletar(colaboradorID, blocoID, usuarioID string) error {
	// Posse do colaborador (líder dono OU o próprio liderado).
	if err := uc.garantirPosse(colaboradorID, usuarioID); err != nil {
		return err
	}
	// O bloco precisa existir E pertencer ao colaborador da rota (evita apagar
	// bloco de outro colaborador passando um colaboradorID que você possui).
	bloco, err := uc.repo.BuscarPorId(blocoID)
	if err != nil {
		return fmt.Errorf("bloco não encontrado")
	}
	if bloco.ColaboradorID != colaboradorID {
		return ErrAcessoNegado
	}
	return uc.repo.Deletar(blocoID)
}

// extensaoPorTipo deriva a extensão do arquivo a partir do Content-Type.
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
