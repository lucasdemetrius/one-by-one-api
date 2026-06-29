// Pacote: internal/saude1a1
// Arquivo: controller.go
// Descrição: Endpoint da "Saúde do 1:1" (card do /painel). Só o gestor (ApenasLider)
//            vê os próprios números. Sempre escopado pelo usuario_id do token.
// Autor: OneByOne API
// Criado em: 2026

package saude1a1

import (
	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller expõe a rota de saúde do 1:1.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de saúde do 1:1.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra GET /saude-1a1 (gestor logado).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	g := router.Group("/saude-1a1")
	g.Use(authMiddleware, middleware.ApenasLider())
	g.GET("", c.Obter)
}

// Obter retorna a saúde do 1:1 do gestor autenticado
// @Summary      Saúde do 1:1 (cadência + streak)
// @Description  % da agenda em dia, atrasados, realizados nos últimos 30 dias e sequência de semanas
// @Tags         Saúde do 1:1
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=SaudeRespostaDTO}  "Saúde do 1:1"
// @Failure      401  {object}  response.ErroPadrao                              "Não autenticado"
// @Failure      403  {object}  response.ErroPadrao                              "Apenas líder"
// @Router       /saude-1a1 [get]
func (c *Controller) Obter(ctx *gin.Context) {
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.Obter(usuarioID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}
