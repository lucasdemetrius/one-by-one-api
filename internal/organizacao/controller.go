// Pacote: internal/organizacao
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de organização. Recebe as requisições,
//            valida os dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package organizacao

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

var tiposImagemPermitidos = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// Controller gerencia os endpoints HTTP do módulo de organização
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de organização
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de organização
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	orgs := router.Group("/organizacoes")
	orgs.Use(authMiddleware)
	{
		// Gestão restrita a LIDER/RH (defesa em profundidade); a posse fina (dono OU RH do
		// tenant) é checada no usecase. GET tem posse no usecase e devolve 404 a quem não é dono.
		orgs.POST("", middleware.PermitirGestaoOuRH(), c.Criar)
		orgs.GET("", c.Listar)
		orgs.GET("/:id", c.BuscarPorId)
		orgs.PUT("/:id", middleware.PermitirGestaoOuRH(), c.Atualizar)
		orgs.DELETE("/:id", middleware.PermitirGestaoOuRH(), c.Deletar)
		orgs.POST("/:id/foto", middleware.PermitirGestaoOuRH(), c.UploadFoto)
	}
}

// Criar cadastra uma nova organização vinculada ao líder autenticado
// @Summary      Criar organização
// @Description  Cria uma nova organização vinculada ao líder autenticado
// @Tags         Organizações
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarOrganizacaoDTO                                         true  "Dados da organização"
// @Success      201   {object}  response.RespostaPadrao{dados=OrganizacaoRespostaDTO}        "Organização criada"
// @Failure      400   {object}  response.ErroPadrao                                          "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                          "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                          "Erro interno"
// @Router       /organizacoes [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarOrganizacaoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	// Extrai o ID do usuário autenticado para vincular a organização ao líder correto
	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.Criar(usuarioID, dto)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, resultado)
}

// BuscarPorId retorna os dados de uma organização pelo UUID
// @Summary      Buscar organização por ID
// @Description  Retorna os dados de uma organização ativa pelo UUID
// @Tags         Organizações
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                        true  "UUID da organização"
// @Success      200  {object}  response.RespostaPadrao{dados=OrganizacaoRespostaDTO}         "Organização encontrada"
// @Failure      401  {object}  response.ErroPadrao                                           "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                           "Não encontrada"
// @Router       /organizacoes/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "organização não encontrada")
		return
	}
	response.Sucesso(ctx, resultado)
}

// Listar retorna todas as organizações do líder autenticado
// @Summary      Listar organizações
// @Description  Retorna todas as organizações ativas do líder autenticado
// @Tags         Organizações
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=[]OrganizacaoRespostaDTO}       "Lista de organizações"
// @Failure      401  {object}  response.ErroPadrao                                           "Não autenticado"
// @Failure      500  {object}  response.ErroPadrao                                           "Erro interno"
// @Router       /organizacoes [get]
func (c *Controller) Listar(ctx *gin.Context) {
	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.ListarPorUsuario(usuarioID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// Atualizar modifica os dados de uma organização existente
// @Summary      Atualizar organização
// @Description  Atualiza parcialmente os dados de uma organização pelo UUID
// @Tags         Organizações
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                        true  "UUID da organização"
// @Param        body  body      AtualizarOrganizacaoDTO                                       true  "Campos a atualizar"
// @Success      200   {object}  response.RespostaPadrao{dados=OrganizacaoRespostaDTO}         "Organização atualizada"
// @Failure      400   {object}  response.ErroPadrao                                           "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                           "Não autenticado"
// @Failure      404   {object}  response.ErroPadrao                                           "Não encontrada"
// @Router       /organizacoes/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	var dto AtualizarOrganizacaoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	resultado, err := c.uc.Atualizar(id, usuarioID, dto)
	if err != nil {
		if errors.Is(err, ErrAcessoNegado) {
			response.ErroNaoEncontrado(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// UploadFoto faz o upload da foto da organização para o S3 e retorna a URL presignada
// @Summary      Upload de foto da organização
// @Description  Recebe um arquivo de imagem (JPEG, PNG ou WebP, máx. 5MB) via multipart/form-data
// @Tags         Organizações
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                        true  "UUID da organização"
// @Param        foto  formData  file                                                          true  "Arquivo de imagem"
// @Success      200   {object}  response.RespostaPadrao{dados=OrganizacaoRespostaDTO}         "Foto enviada"
// @Failure      400   {object}  response.ErroPadrao                                           "Arquivo inválido"
// @Failure      404   {object}  response.ErroPadrao                                           "Não encontrada"
// @Router       /organizacoes/{id}/foto [post]
func (c *Controller) UploadFoto(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	const maxTamanho = 5 << 20
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxTamanho)

	arquivo, header, err := ctx.Request.FormFile("foto")
	if err != nil {
		response.ErroRequisicao(ctx, "arquivo 'foto' inválido ou ausente")
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

	resultado, err := c.uc.UploadFoto(id, usuarioID, arquivo, header.Size, tipoConteudo)
	if err != nil {
		if errors.Is(err, ErrAcessoNegado) {
			response.ErroNaoEncontrado(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}

	response.Sucesso(ctx, resultado)
}

// Deletar realiza a exclusão lógica de uma organização
// @Summary      Deletar organização
// @Description  Realiza o soft delete de uma organização pelo UUID
// @Tags         Organizações
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                true  "UUID da organização"
// @Success      200  {object}  response.RespostaPadrao{dados=string}                "Deletada com sucesso"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                  "Não encontrada"
// @Router       /organizacoes/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	deletadoPorInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	deletadoPor, _ := deletadoPorInterface.(string)

	if err := c.uc.Deletar(id, deletadoPor); err != nil {
		response.ErroNaoEncontrado(ctx, "organização não encontrada")
		return
	}
	response.Sucesso(ctx, "organização deletada com sucesso")
}
