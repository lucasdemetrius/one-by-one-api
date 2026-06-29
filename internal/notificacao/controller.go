// Pacote: internal/notificacao
// Arquivo: controller.go
// Descrição: Endpoints HTTP das notificações (sino) e das preferências. Tudo é do
//            próprio usuário do token. A rota de marcar-uma fica sob /itens/:id
//            para não conflitar (estático x param) com /ler-todas.
// Autor: OneByOne API
// Criado em: 2026

package notificacao

import (
	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints de notificação.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de notificações.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas de notificação (todas do usuário logado).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	g := router.Group("/notificacoes")
	g.Use(authMiddleware)
	{
		g.GET("", c.Listar)
		g.GET("/contagem", c.Contar)
		g.GET("/preferencias", c.ObterPref)
		g.PUT("/preferencias", c.SalvarPref)
		g.PUT("/ler-todas", c.LerTodas)
		g.PUT("/itens/:id/lida", c.MarcarLida)
	}
}

func (c *Controller) uid(ctx *gin.Context) string { return ctx.GetString(middleware.ChaveUsuarioID) }

func (c *Controller) Listar(ctx *gin.Context) {
	res, err := c.uc.Listar(c.uid(ctx))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

func (c *Controller) Contar(ctx *gin.Context) {
	n, err := c.uc.ContarNaoLidas(c.uid(ctx))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, gin.H{"nao_lidas": n})
}

func (c *Controller) MarcarLida(ctx *gin.Context) {
	if err := c.uc.MarcarLida(ctx.Param("id"), c.uid(ctx)); err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, "ok")
}

func (c *Controller) LerTodas(ctx *gin.Context) {
	if err := c.uc.MarcarTodasLidas(c.uid(ctx)); err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, "ok")
}

func (c *Controller) ObterPref(ctx *gin.Context) {
	p, err := c.uc.ObterPref(c.uid(ctx))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, p)
}

func (c *Controller) SalvarPref(ctx *gin.Context) {
	var dto PrefDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	if err := c.uc.SalvarPref(c.uid(ctx), dto); err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, "preferências salvas")
}
