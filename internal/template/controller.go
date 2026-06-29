// Pacote: internal/template
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de template. Recebe as requisições,
//            valida os dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package template

import (
	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints HTTP do módulo de template
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de template
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de template
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Templates são configuração do gestor — todas as rotas são só LÍDER.
	// (O liderado nunca chama /templates; ao abrir um 1:1 o template é resolvido
	// no servidor.) A posse por usuario_id ainda é checada no usecase.
	templates := router.Group("/templates")
	templates.Use(authMiddleware, middleware.ApenasLider())
	{
		templates.POST("", c.Criar)
		templates.GET("", c.Listar)
		templates.GET("/:id", c.BuscarPorId)
		templates.PUT("/:id", c.Atualizar)
		templates.DELETE("/:id", c.Deletar)
	}
}

// Criar cadastra um novo template vinculado ao líder autenticado
// @Summary      Criar template
// @Description  Cria um novo template de formulário para o líder autenticado
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarTemplateDTO                                        true  "Dados do template"
// @Success      201   {object}  response.RespostaPadrao{dados=TemplateRespostaDTO}      "Template criado"
// @Failure      400   {object}  response.ErroPadrao                                     "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                     "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                     "Erro interno"
// @Router       /templates [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarTemplateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.Criar(usuarioID, dto)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, resultado)
}

// BuscarPorId retorna os dados de um template pelo UUID
// @Summary      Buscar template por ID
// @Description  Retorna os dados de um template ativo pelo UUID
// @Tags         Templates
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                true  "UUID do template"
// @Success      200  {object}  response.RespostaPadrao{dados=TemplateRespostaDTO}   "Template encontrado"
// @Failure      404  {object}  response.ErroPadrao                                  "Não encontrado"
// @Router       /templates/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "template não encontrado")
		return
	}
	response.Sucesso(ctx, resultado)
}

// Listar retorna todos os templates do líder autenticado
// @Summary      Listar templates
// @Description  Retorna todos os templates ativos do líder autenticado, do mais antigo ao mais novo
// @Tags         Templates
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=[]TemplateRespostaDTO}  "Lista de templates"
// @Failure      401  {object}  response.ErroPadrao                                   "Não autenticado"
// @Router       /templates [get]
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

// Atualizar renomeia um template existente
// @Summary      Atualizar template
// @Description  Atualiza o nome de um template pelo UUID. Apenas o dono pode alterar.
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                true  "UUID do template"
// @Param        body  body      AtualizarTemplateDTO                                  true  "Novo nome"
// @Success      200   {object}  response.RespostaPadrao{dados=TemplateRespostaDTO}   "Template atualizado"
// @Failure      400   {object}  response.ErroPadrao                                  "Dados inválidos"
// @Failure      403   {object}  response.ErroPadrao                                  "Sem permissão"
// @Failure      404   {object}  response.ErroPadrao                                  "Não encontrado"
// @Router       /templates/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	var dto AtualizarTemplateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.Atualizar(id, usuarioID, dto)
	if err != nil {
		if err.Error() == "você não tem permissão para alterar este template" {
			response.ErroProibido(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// Deletar realiza a exclusão lógica de um template
// @Summary      Deletar template
// @Description  Realiza o soft delete de um template pelo UUID. Apenas o dono pode excluir.
// @Tags         Templates
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                           true  "UUID do template"
// @Success      200  {object}  response.RespostaPadrao{dados=string}            "Deletado com sucesso"
// @Failure      403  {object}  response.ErroPadrao                              "Sem permissão"
// @Failure      404  {object}  response.ErroPadrao                              "Não encontrado"
// @Router       /templates/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	if err := c.uc.Deletar(id, usuarioID, usuarioID); err != nil {
		if err.Error() == "você não tem permissão para excluir este template" {
			response.ErroProibido(ctx, err.Error())
			return
		}
		response.ErroNaoEncontrado(ctx, "template não encontrado")
		return
	}
	response.Sucesso(ctx, "template deletado com sucesso")
}
