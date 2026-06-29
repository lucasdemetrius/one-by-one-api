// Pacote: internal/templatebloco
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de bloco de template. Recebe requisições,
//            valida dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package templatebloco

import (
	"errors"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz falta de posse (ErrAcessoNegado) em 404; o resto em 500.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "bloco de template não encontrado")
		return
	}
	response.ErroInterno(ctx, err.Error())
}

// Controller gerencia os endpoints HTTP do módulo de bloco de template
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de bloco de template
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de bloco de template
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Blocos de template são configuração do gestor — só LÍDER (a posse pelo
	// template pai ainda é checada no usecase).
	blocos := router.Group("/template-blocos")
	blocos.Use(authMiddleware, middleware.ApenasLider())
	{
		blocos.POST("", c.Criar)
		blocos.GET("/:id", c.BuscarPorId)
		blocos.PUT("/:id", c.Atualizar)
		blocos.DELETE("/:id", c.Deletar)
	}

	// Rota aninhada: listar blocos de um template específico
	// Usa prefixo próprio para evitar conflito de wildcard com GET /templates/:id
	aninhado := router.Group("/templates/:id/blocos")
	aninhado.Use(authMiddleware, middleware.ApenasLider())
	aninhado.GET("", c.ListarPorTemplate)
}

// Criar cadastra um novo bloco dentro de um template
// @Summary      Criar bloco de template
// @Description  Cria um novo bloco (campo de formulário) dentro de um template existente
// @Tags         Blocos de Template
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarTemplateBlocoDTO                                          true  "Dados do bloco"
// @Success      201   {object}  response.RespostaPadrao{dados=TemplateBlocoRespostaDTO}        "Bloco criado"
// @Failure      400   {object}  response.ErroPadrao                                            "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                            "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                            "Erro interno"
// @Router       /template-blocos [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarTemplateBlocoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.Criar(dto, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, resultado)
}

// BuscarPorId retorna os dados de um bloco pelo UUID
// @Summary      Buscar bloco por ID
// @Description  Retorna os dados de um bloco de template ativo pelo UUID
// @Tags         Blocos de Template
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                       true  "UUID do bloco"
// @Success      200  {object}  response.RespostaPadrao{dados=TemplateBlocoRespostaDTO}      "Bloco encontrado"
// @Failure      404  {object}  response.ErroPadrao                                          "Não encontrado"
// @Router       /template-blocos/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "bloco de template não encontrado")
		return
	}
	response.Sucesso(ctx, resultado)
}

// ListarPorTemplate retorna todos os blocos de um template ordenados por posicao
// @Summary      Listar blocos por template
// @Description  Retorna todos os blocos ativos de um template, ordenados pela posição de exibição
// @Tags         Blocos de Template
// @Produce      json
// @Security     BearerAuth
// @Param        templateId  path      string                                                          true  "UUID do template"
// @Success      200         {object}  response.RespostaPadrao{dados=[]TemplateBlocoRespostaDTO}       "Lista de blocos"
// @Failure      401         {object}  response.ErroPadrao                                            "Não autenticado"
// @Router       /templates/{templateId}/blocos [get]
func (c *Controller) ListarPorTemplate(ctx *gin.Context) {
	templateID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarPorTemplate(templateID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// Atualizar modifica os dados de um bloco existente
// @Summary      Atualizar bloco
// @Description  Atualiza parcialmente os dados de um bloco de template pelo UUID
// @Tags         Blocos de Template
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                       true  "UUID do bloco"
// @Param        body  body      AtualizarTemplateBlocoDTO                                    true  "Campos a atualizar"
// @Success      200   {object}  response.RespostaPadrao{dados=TemplateBlocoRespostaDTO}      "Bloco atualizado"
// @Failure      400   {object}  response.ErroPadrao                                          "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                          "Não encontrado"
// @Router       /template-blocos/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	var dto AtualizarTemplateBlocoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.Atualizar(id, usuarioID, dto)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// Deletar realiza a exclusão lógica de um bloco de template
// @Summary      Deletar bloco
// @Description  Realiza o soft delete de um bloco de template pelo UUID
// @Tags         Blocos de Template
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                               true  "UUID do bloco"
// @Success      200  {object}  response.RespostaPadrao{dados=string}                "Deletado com sucesso"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                  "Não encontrado"
// @Router       /template-blocos/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	if err := c.uc.Deletar(id, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "bloco de template não encontrado")
		return
	}
	response.Sucesso(ctx, "bloco de template deletado com sucesso")
}
