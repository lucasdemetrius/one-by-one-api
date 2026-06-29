// Pacote: internal/agendamento
// Arquivo: controller.go
// Descrição: Controlador HTTP dos agendamentos. O gestor (usuário do JWT) cria,
//            lista e remove os seus 1:1 agendados. Rotas protegidas por JWT.
// Autor: OneByOne API
// Criado em: 2025

package agendamento

import (
	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints dos agendamentos.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de agendamentos.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas (todas protegidas por JWT).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Agenda é gestão do gestor — só LÍDER. A posse do liderado/agendamento
	// ainda é checada no usecase (Criar valida o colaborador; Listar/Deletar
	// filtram por usuario_id no SQL).
	g := router.Group("/agendamentos")
	g.Use(authMiddleware, middleware.ApenasLider())
	{
		g.POST("", c.Criar)
		g.GET("", c.Listar)
		// Cancela TODOS os 1:1 de um liderado de uma vez (?colaborador_id=). Rota sem :id
		// para não conflitar com DELETE /:id no roteador.
		g.DELETE("", c.CancelarDoColaborador)
		g.PUT("/:id", c.Reagendar)
		g.DELETE("/:id", c.Deletar)
	}
}

// usuarioDoToken recupera o id do gestor autenticado.
func usuarioDoToken(ctx *gin.Context) string {
	id, _ := ctx.Get(middleware.ChaveUsuarioID)
	s, _ := id.(string)
	return s
}

// Criar agenda um novo 1:1.
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarAgendamentoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	res, err := c.uc.Criar(usuarioDoToken(ctx), dto)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, res)
}

// Listar devolve os agendamentos ativos do gestor.
func (c *Controller) Listar(ctx *gin.Context) {
	res, err := c.uc.ListarPorUsuario(usuarioDoToken(ctx))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// Reagendar muda a data/hora de um 1:1 (arrastar no calendário). Corpo: {data_hora}.
func (c *Controller) Reagendar(ctx *gin.Context) {
	var dto struct {
		DataHora string `json:"data_hora" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroRequisicao(ctx, "informe data_hora")
		return
	}
	if err := c.uc.Reagendar(ctx.Param("id"), usuarioDoToken(ctx), dto.DataHora); err != nil {
		// Data em formato inválido é erro de NEGÓCIO (amigável) → 400; o resto (não
		// encontrado / posse) → 404 com mensagem fixa, sem vazar SQL do repositório.
		if err.Error() == "data/hora inválida (use AAAA-MM-DDTHH:MM)" {
			response.ErroRequisicao(ctx, err.Error())
		} else {
			response.ErroNaoEncontrado(ctx, "agendamento não encontrado")
		}
		return
	}
	response.Sucesso(ctx, "agendamento reagendado")
}

// Deletar remove um agendamento do gestor.
func (c *Controller) Deletar(ctx *gin.Context) {
	if err := c.uc.Deletar(ctx.Param("id"), usuarioDoToken(ctx)); err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, "agendamento removido")
}

// CancelarDoColaborador remove TODOS os 1:1 de um liderado de uma vez. O liderado vem por
// query (?colaborador_id=). Útil quando o liderado sai da empresa.
func (c *Controller) CancelarDoColaborador(ctx *gin.Context) {
	colaboradorID := ctx.Query("colaborador_id")
	if colaboradorID == "" {
		response.ErroRequisicao(ctx, "informe colaborador_id")
		return
	}
	n, err := c.uc.CancelarDoColaborador(colaboradorID, usuarioDoToken(ctx))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, gin.H{"cancelados": n})
}
