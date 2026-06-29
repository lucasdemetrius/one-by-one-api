// Pacote: internal/ia
// Arquivo: controller.go
// Descrição: Endpoints HTTP da IA do gestor. Config (ler/salvar) e chat. Todas as
//            rotas exigem JWT; a config é sempre do próprio usuário do token.
// Autor: OneByOne API
// Criado em: 2026

package ia

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints de IA.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de IA.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas de IA (todas protegidas por JWT).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	g := router.Group("/ia")
	g.Use(authMiddleware)
	{
		g.GET("/config", c.ObterConfig)
		g.PUT("/config", c.SalvarConfig)
		g.POST("/chat", c.Chat)
	}
}

// ObterConfig devolve o provedor e se há chave (nunca a chave).
func (c *Controller) ObterConfig(ctx *gin.Context) {
	res, err := c.uc.ObterConfig(ctx.GetString(middleware.ChaveUsuarioID))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// SalvarConfig grava o provedor e (se enviada) a chave.
func (c *Controller) SalvarConfig(ctx *gin.Context) {
	var dto SalvarConfigDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	if err := c.uc.SalvarConfig(ctx.GetString(middleware.ChaveUsuarioID), dto); err != nil {
		// Erro de negócio (mensagem amigável): mostra ao usuário.
		if errors.Is(err, ErrProvedorInvalido) {
			response.ErroRequisicao(ctx, err.Error())
			return
		}
		// Erro técnico (cripto/banco): NÃO vaza — loga no servidor e devolve genérico.
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, "configuração de IA salva")
}

// Chat responde uma pergunta livre do gestor.
func (c *Controller) Chat(ctx *gin.Context) {
	var dto ChatDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroRequisicao(ctx, "informe a mensagem")
		return
	}
	resposta, err := c.uc.Chat(ctx.GetString(middleware.ChaveUsuarioID), dto.Mensagem)
	if err != nil {
		if errors.Is(err, ErrSemConfig) {
			response.ErroRequisicao(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, gin.H{"resposta": resposta})
}
