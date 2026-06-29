// Pacote: internal/pdi
// Arquivo: controller.go
// Descrição: Endpoints HTTP do PDI. Gestão do gestor (ApenasLider) com posse no
//            usecase. Recurso alheio → 404.
// Autor: OneByOne API
// Criado em: 2026

package pdi

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints do PDI.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de PDI.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "não encontrado")
		return
	}
	// Mensagens de NEGÓCIO (amigáveis, fixas) vindas do usecase continuam chegando
	// ao usuário. O "item de PDI não encontrado" é um 404 de negócio; o prazo
	// inválido é uma validação de entrada (400).
	switch err.Error() {
	case "item de PDI não encontrado":
		response.ErroNaoEncontrado(ctx, err.Error())
		return
	case "prazo inválido — use AAAA-MM-DD":
		response.ErroRequisicao(ctx, err.Error())
		return
	}
	// Qualquer outro erro é TÉCNICO (posse/banco/IO): loga no servidor e devolve
	// uma mensagem genérica, sem vazar a estrutura interna.
	response.ErroInterno(ctx, err.Error())
}

// RegistrarRotas registra as rotas do PDI.
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Listar é por colaborador; criar/editar/excluir são gestão (só LÍDER).
	router.GET("/colaboradores/:id/pdi", authMiddleware, c.Listar)
	router.POST("/colaboradores/:id/pdi", authMiddleware, middleware.ApenasLider(), c.Criar)

	itens := router.Group("/pdi")
	itens.Use(authMiddleware, middleware.ApenasLider())
	{
		itens.PUT("/:id", c.Atualizar)
		itens.DELETE("/:id", c.Deletar)
	}
}

func (c *Controller) Listar(ctx *gin.Context) {
	res, err := c.uc.Listar(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID))
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarItemPDIDTO
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
	var dto AtualizarItemPDIDTO
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
	response.Sucesso(ctx, "item de PDI removido")
}
