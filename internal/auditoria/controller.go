package auditoria

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// PosseColaborador é a parte do colaborador.UseCase de que precisamos para checar
// a posse na linha do tempo (interface enxuta para não acoplar os pacotes).
type PosseColaborador interface {
	PertenceAoLider(colaboradorID, usuarioLiderID string) (bool, error)
}

// Controller gerencia os endpoints HTTP do módulo de auditoria
type Controller struct {
	uc      UseCase
	colabUC PosseColaborador // pode ser nil (a timeline só registra a rota se presente)
}

func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// ComPosseColaborador injeta a checagem de posse usada pela linha do tempo do liderado.
func (c *Controller) ComPosseColaborador(p PosseColaborador) *Controller {
	c.colabUC = p
	return c
}

func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	grupo := router.Group("/auditoria")
	grupo.Use(authMiddleware)
	{
		// POST /auditoria/eventos — frontend envia eventos de UI
		grupo.POST("/eventos", c.RegistrarEvento)
		// GET  /auditoria/minha — timeline do usuário autenticado
		grupo.GET("/minha", c.MinhaTrilha)
	}

	// Linha do tempo de um liderado (só o líder dono). Requer a posse injetada.
	if c.colabUC != nil {
		router.GET("/colaboradores/:id/timeline", authMiddleware, c.TimelineColaborador)
	}
}

// TimelineColaborador devolve a linha do tempo (eventos) de um liderado do gestor.
func (c *Controller) TimelineColaborador(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	dono, err := c.colabUC.PertenceAoLider(colaboradorID, usuarioID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	if !dono {
		response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
		return
	}

	registros, err := c.uc.ListarPorEntidade(colaboradorID, 100)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, registros)
}

// RegistrarEvento recebe um evento de UI enviado pelo frontend
// @Summary      Registrar evento de UI
// @Description  Permite que o frontend envie eventos de navegação, cliques e visualizações
//
//	para compor a trilha de atividade do usuário.
//
// @Tags         Auditoria
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      EventoDTO                                         true  "Dados do evento"
// @Success      200   {object}  response.RespostaPadrao{dados=string}             "Evento registrado"
// @Failure      400   {object}  response.ErroPadrao                               "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                               "Não autenticado"
// @Router       /auditoria/eventos [post]
func (c *Controller) RegistrarEvento(ctx *gin.Context) {
	var dto EventoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	c.uc.RegistrarEvento(usuarioID, dto, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	response.Sucesso(ctx, "evento registrado")
}

// MinhaTrilha retorna os últimos eventos do usuário autenticado
// @Summary      Minha trilha de atividade
// @Description  Retorna os últimos N eventos registrados para o usuário autenticado (padrão 50, máx 200).
// @Tags         Auditoria
// @Produce      json
// @Security     BearerAuth
// @Param        limite  query     int                                                        false  "Quantidade de registros (padrão 50)"
// @Success      200     {object}  response.RespostaPadrao{dados=[]AuditoriaRespostaDTO}      "Lista de eventos"
// @Failure      401     {object}  response.ErroPadrao                                        "Não autenticado"
// @Failure      500     {object}  response.ErroPadrao                                        "Erro interno"
// @Router       /auditoria/minha [get]
func (c *Controller) MinhaTrilha(ctx *gin.Context) {
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	limite := 50
	if q := ctx.Query("limite"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			limite = v
		}
	}

	registros, err := c.uc.ListarPorUsuario(usuarioID, limite)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, registros)
}
