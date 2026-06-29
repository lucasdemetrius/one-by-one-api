// Pacote: internal/rh
// Arquivo: controller.go
// Descrição: Endpoints HTTP exclusivos do RH (/api/v1/rh/...). Todo o grupo é protegido
//            pelo middleware ApenasRH — só contas RH chegam aqui. O rhID vem SEMPRE do
//            JWT (ChaveUsuarioID), nunca do corpo, então um RH só age sobre o seu tenant.
// Autor: OneByOne API
// Criado em: 2026

package rh

import (
	"errors"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
	"onebyone-api/pkg/senha"
)

// Controller expõe os endpoints do módulo de RH.
type Controller struct {
	uc UseCase
}

// NovoController cria o Controller de RH.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas do RH sob /rh, todas exigindo JWT + papel RH.
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	grupo := router.Group("/rh")
	grupo.Use(authMiddleware)
	grupo.Use(middleware.ApenasRH())
	{
		grupo.POST("/gestores", c.CriarGestor)
		grupo.GET("/gestores", c.ListarGestores)
		grupo.GET("/gestores/:id/onebyones", c.OneByonesDoGestor)
		grupo.GET("/gestores/:id/agendamentos", c.AgendamentosDoGestor)
		grupo.GET("/agenda", c.Agenda)
		grupo.GET("/matrix", c.Matrix)
		grupo.GET("/acompanhamento", c.Acompanhamento)
	}
}

// Acompanhamento devolve a evolução dos liderados de cada gestor do tenant (foco em qualidade,
// não em quantidade de 1:1), ordenado por necessidade de atenção.
// @Summary  Acompanhamento dos gestores (evolução dos liderados)
// @Tags     RH
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=[]GestorEvolucaoDTO}  "Evolução por gestor"
// @Router   /rh/acompanhamento [get]
func (c *Controller) Acompanhamento(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.AcompanhamentoDosGestores(rhID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// Agenda devolve a agenda consolidada do tenant (todos os gestores) para o calendário do RH.
// @Summary  Agenda consolidada do RH
// @Tags     RH
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=[]AgendaItemDTO}  "Agenda do tenant"
// @Router   /rh/agenda [get]
func (c *Controller) Agenda(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.AgendaDoTenant(rhID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// Matrix devolve a 9-box consolidada do tenant (todos os liderados) para o RH.
// @Summary  Matrix 9-box consolidada do RH
// @Tags     RH
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=[]MatrixItemDTO}  "9-box do tenant"
// @Router   /rh/matrix [get]
func (c *Controller) Matrix(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.MatrixDoTenant(rhID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// CriarGestor cadastra um gestor sob o RH autenticado
// @Summary      RH cadastra um gestor
// @Description  Cria a conta de um Gestor (LIDER) já vinculada ao tenant do RH autenticado.
//
//	O vínculo (rh_id) é derivado do JWT do RH — nunca do corpo.
//
// @Tags         RH
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarGestorDTO                                     true  "Dados do gestor"
// @Success      201   {object}  response.RespostaPadrao{dados=GestorResumoDTO}     "Gestor criado"
// @Failure      400   {object}  response.ErroPadrao                                "Dados inválidos"
// @Failure      403   {object}  response.ErroPadrao                                "Acesso restrito ao RH"
// @Failure      409   {object}  response.ErroPadrao                                "E-mail já cadastrado"
// @Router       /rh/gestores [post]
func (c *Controller) CriarGestor(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)

	var dto CriarGestorDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	// Complexidade da senha do gestor (≥ 8, com maiúscula, minúscula e número).
	if err := senha.Validar(dto.Password); err != nil {
		response.ErroRequisicao(ctx, err.Error())
		return
	}

	resultado, err := c.uc.CriarGestor(rhID, dto)
	if err != nil {
		if err.Error() == "já existe um usuário com este e-mail" {
			response.ErroConflito(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, resultado)
}

// ListarGestores retorna os gestores do RH com KPIs de produtividade
// @Summary      Listar gestores do RH (dashboard)
// @Description  Lista os gestores do tenant do RH com KPIs de saúde do 1:1 (produtividade)
// @Tags         RH
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=[]GestorResumoDTO}   "Lista de gestores"
// @Failure      403  {object}  response.ErroPadrao                                "Acesso restrito ao RH"
// @Router       /rh/gestores [get]
func (c *Controller) ListarGestores(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarGestores(rhID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// OneByonesDoGestor lista os 1:1 de um gestor do tenant
// @Summary      1:1 de um gestor (visão do RH)
// @Description  Lista os one-on-ones de um gestor do tenant do RH (drill-down)
// @Tags         RH
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "UUID do gestor"
// @Success      200  {object}  response.RespostaPadrao  "Lista de 1:1"
// @Failure      403  {object}  response.ErroPadrao      "Acesso restrito ao RH"
// @Failure      404  {object}  response.ErroPadrao      "Gestor não encontrado no tenant"
// @Router       /rh/gestores/{id}/onebyones [get]
func (c *Controller) OneByonesDoGestor(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)
	gestorID := ctx.Param("id")
	resultado, err := c.uc.OneByonesDoGestor(rhID, gestorID)
	if err != nil {
		if errors.Is(err, ErrGestorForaDoTenant) {
			response.ErroNaoEncontrado(ctx, "gestor não encontrado")
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// AgendamentosDoGestor lista a agenda de 1:1 de um gestor do tenant
// @Summary      Agenda de um gestor (visão do RH)
// @Description  Lista os agendamentos de 1:1 de um gestor do tenant do RH (drill-down)
// @Tags         RH
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "UUID do gestor"
// @Success      200  {object}  response.RespostaPadrao  "Lista de agendamentos"
// @Failure      403  {object}  response.ErroPadrao      "Acesso restrito ao RH"
// @Failure      404  {object}  response.ErroPadrao      "Gestor não encontrado no tenant"
// @Router       /rh/gestores/{id}/agendamentos [get]
func (c *Controller) AgendamentosDoGestor(ctx *gin.Context) {
	rhID := ctx.GetString(middleware.ChaveUsuarioID)
	gestorID := ctx.Param("id")
	resultado, err := c.uc.AgendamentosDoGestor(rhID, gestorID)
	if err != nil {
		if errors.Is(err, ErrGestorForaDoTenant) {
			response.ErroNaoEncontrado(ctx, "gestor não encontrado")
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}
