// Pacote: internal/feedback
// Arquivo: controller.go
// Descrição: Endpoints HTTP do feedback. POST /feedback é para QUALQUER usuário logado
//            (um clique: curti / não curti / irritado, com comentário opcional). O painel
//            GET /admin/feedbacks fica sob o dashboard de gestão, protegido por ApenasAdmin.
// Autor: OneByOne API
// Criado em: 2026

package feedback

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller expõe os endpoints de feedback.
type Controller struct {
	uc UseCase
}

// NovoController cria o Controller de feedback.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra a escrita (qualquer usuário logado) e o painel (só ADMIN).
// `limiteEscrita` é um rate-limit POR USUÁRIO aplicado ao POST (anti-abuso/inflar a tabela).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, limiteEscrita gin.HandlerFunc) {
	// Escrita — qualquer usuário autenticado reage em um clique (com rate-limit por usuário).
	escrita := router.Group("/feedback")
	escrita.Use(authMiddleware)
	escrita.POST("", limiteEscrita, c.Registrar)

	// Leitura agregada — vive no caminho do dashboard de gestão e é exclusiva do ADMIN.
	painel := router.Group("/admin/feedbacks")
	painel.Use(authMiddleware, middleware.ApenasAdmin())
	painel.GET("", c.PainelAdmin)
}

// Registrar grava a reação do usuário autenticado.
// @Summary  Registrar feedback (curti / não curti / irritado)
// @Tags     Feedback
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body  CriarFeedbackDTO  true  "Reação + contexto/comentário opcionais"
// @Success  201  {object}  response.RespostaPadrao{dados=FeedbackRespostaDTO}  "Feedback registrado"
// @Failure  400  {object}  response.ErroPadrao                                 "Dados inválidos"
// @Router   /feedback [post]
func (c *Controller) Registrar(ctx *gin.Context) {
	var dto CriarFeedbackDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.Registrar(usuarioID, dto)
	if err != nil {
		if errors.Is(err, ErrReacaoInvalida) {
			response.ErroRequisicao(ctx, "reação inválida (use CURTI, NAO_CURTI ou IRRITADO)")
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, res)
}

// PainelAdmin devolve o resumo de feedback para o dashboard de gestão.
// @Summary  Painel de feedback (admin)
// @Description Totais, índice de satisfação, série temporal por reação, por contexto e comentários recentes.
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Param    dias  query  int  false  "Janela em dias (padrão 30, máx 365)"
// @Success  200  {object}  response.RespostaPadrao{dados=PainelFeedbackDTO}  "Painel de feedback"
// @Failure  403  {object}  response.ErroPadrao                               "Acesso restrito ao administrador"
// @Router   /admin/feedbacks [get]
func (c *Controller) PainelAdmin(ctx *gin.Context) {
	dias := 0
	if v := ctx.Query("dias"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			dias = n
		}
	}
	res, err := c.uc.PainelAdmin(dias)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}
