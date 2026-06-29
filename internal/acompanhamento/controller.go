// Pacote: internal/acompanhamento
// Arquivo: controller.go
// Descrição: Endpoints HTTP do acompanhamento. Gestão do gestor (ApenasLider) com
//            posse no usecase. Recurso alheio → 404. Listar aceita ?tipo= opcional.
// Autor: OneByOne API
// Criado em: 2026

package acompanhamento

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints de acompanhamento.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de acompanhamento.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

func responderErro(ctx *gin.Context, err error) {
	// Recurso alheio/inexistente → 404 (não revela que o id existe).
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "não encontrado")
		return
	}
	// Erros de NEGÓCIO (validação): mensagem amigável fixa → 400.
	if errors.Is(err, ErrDataInvalida) ||
		errors.Is(err, ErrHumorObrigatorio) ||
		errors.Is(err, ErrTituloObrigatorio) {
		response.ErroRequisicao(ctx, err.Error())
		return
	}
	// Qualquer outro erro é técnico/inesperado (banco/IO): não vaza ao cliente.
	// ErroInterno loga no servidor e devolve resposta genérica (500).
	response.ErroInterno(ctx, err.Error())
}

// RegistrarRotas registra as rotas de acompanhamento.
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	router.GET("/colaboradores/:id/acompanhamento", authMiddleware, c.Listar)
	router.POST("/colaboradores/:id/acompanhamento", authMiddleware, middleware.ApenasLider(), c.Criar)

	g := router.Group("/acompanhamento")
	g.Use(authMiddleware, middleware.ApenasLider())
	{
		g.PUT("/:id", c.Atualizar)
		g.DELETE("/:id", c.Deletar)
	}
}

func (c *Controller) Listar(ctx *gin.Context) {
	res, err := c.uc.Listar(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID), ctx.Query("tipo"))
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarAcompanhamentoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	res, err := c.uc.Criar(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID), dto)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, res)
}

func (c *Controller) Atualizar(ctx *gin.Context) {
	var dto AtualizarAcompanhamentoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	res, err := c.uc.Atualizar(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID), dto)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

func (c *Controller) Deletar(ctx *gin.Context) {
	if err := c.uc.Deletar(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID)); err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, "acompanhamento removido")
}
