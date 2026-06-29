// Pacote: internal/blocotema
// Arquivo: controller.go
// Descrição: Controlador HTTP dos blocos de tema. Todas as rotas exigem JWT.
// Autor: OneByOne API
// Criado em: 2025

package blocotema

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz falta de posse (ErrAcessoNegado) em 404 e o resto em 500.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "conteúdo não encontrado")
		return
	}
	response.ErroInterno(ctx, err.Error())
}

// tiposImagemPermitidos lista os Content-Types aceitos no upload.
var tiposImagemPermitidos = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// Controller gerencia os endpoints HTTP dos blocos de tema.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de blocos de tema.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas (todas protegidas por JWT).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	g := router.Group("/colaboradores/:id/blocos")
	g.Use(authMiddleware)
	{
		g.GET("", c.Listar)           // ?tema=...
		g.GET("/tudo", c.ListarTodos) // todo o conteúdo do liderado (para a IA)
		g.POST("", c.Criar)           // texto / link / marco
		g.DELETE("/:blocoId", c.Deletar)
	}
	// Upload de imagem em caminho próprio (registrado fora do grupo /blocos
	// para não virar /blocos/-imagem no path.Join do Gin).
	router.POST("/colaboradores/:id/blocos-imagem", authMiddleware, c.CriarImagem)
}

// Listar devolve os blocos de um tema do colaborador (tema via ?tema=).
func (c *Controller) Listar(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	tema := ctx.Query("tema")
	if tema == "" {
		response.ErroRequisicao(ctx, "informe o tema na query (?tema=...)")
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.Listar(colaboradorID, tema, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

// ListarTodos devolve todo o conteúdo do liderado (todos os temas) — usado pela IA.
func (c *Controller) ListarTodos(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.ListarTodos(colaboradorID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

// Criar adiciona um bloco de texto, link ou marco.
func (c *Controller) Criar(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	var dto CriarBlocoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.Criar(colaboradorID, dto, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, res)
}

// CriarImagem recebe uma imagem (multipart) e cria um bloco do tipo IMAGEM.
func (c *Controller) CriarImagem(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	tema := ctx.PostForm("tema")
	legenda := ctx.PostForm("legenda")
	if tema == "" {
		response.ErroRequisicao(ctx, "informe o tema no formulário")
		return
	}

	// Limita o corpo a 5MB.
	const maxTamanho = 5 << 20
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxTamanho)

	arquivo, header, err := ctx.Request.FormFile("imagem")
	if err != nil {
		response.ErroRequisicao(ctx, "arquivo 'imagem' não encontrado: "+err.Error())
		return
	}
	defer arquivo.Close()

	tipoConteudo := header.Header.Get("Content-Type")
	if !tiposImagemPermitidos[tipoConteudo] {
		response.ErroRequisicao(ctx, "tipo de imagem não suportado; use JPEG, PNG ou WebP")
		return
	}
	// Defesa extra: o Content-Type acima é forjável. Confere o tipo REAL pelos magic bytes
	// e reposiciona o ponteiro para o envio ao S3 ler o arquivo do começo.
	cabecalho := make([]byte, 512)
	lidos, _ := arquivo.Read(cabecalho)
	if _, err := arquivo.Seek(0, 0); err != nil {
		response.ErroRequisicao(ctx, "erro ao processar a imagem")
		return
	}
	if !tiposImagemPermitidos[http.DetectContentType(cabecalho[:lidos])] {
		response.ErroRequisicao(ctx, "o arquivo enviado não é uma imagem válida (JPEG, PNG ou WebP)")
		return
	}

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.CriarImagem(colaboradorID, tema, legenda, arquivo, header.Size, tipoConteudo, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, res)
}

// Deletar remove um bloco pelo UUID (validando posse e vínculo com o colaborador).
func (c *Controller) Deletar(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	blocoID := ctx.Param("blocoId")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	if err := c.uc.Deletar(colaboradorID, blocoID, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "bloco não encontrado")
		return
	}
	response.Sucesso(ctx, "bloco removido")
}
