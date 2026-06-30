// Pacote: internal/ajuda
// Arquivo: controller.go
// Descrição: Endpoints HTTP da Central de Ajuda (/api/v1/ajuda/...). Todas exigem JWT, mas
//            NÃO restringem por papel — ajuda é para todo mundo (gestor, RH, liderado, admin).
//            A rota de IA recebe um limitador de taxa próprio (protege o custo da IA).
// Autor: OneByOne API
// Criado em: 2026

package ajuda

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller expõe os endpoints da Central de Ajuda.
type Controller struct {
	uc UseCase
}

// NovoController cria o Controller da Ajuda.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas de ajuda sob /ajuda (todas exigem JWT). `limiteIA` é um
// rate-limit aplicado só na rota do assistente de IA (chamada externa, com custo).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, limiteIA gin.HandlerFunc) {
	grupo := router.Group("/ajuda")
	grupo.Use(authMiddleware)
	{
		grupo.GET("/topicos", c.ListarTopicos)
		grupo.GET("/topicos/:id", c.ObterTopico)
		grupo.GET("/tour", c.Tour)
		grupo.GET("/ia/status", c.StatusIA)
		grupo.POST("/perguntar", limiteIA, c.Perguntar)
	}
}

// ListarTopicos devolve os tópicos da ajuda visíveis para o papel do usuário.
// @Summary  Tópicos da Central de Ajuda
// @Tags     Ajuda
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=TopicosDTO}  "Tópicos"
// @Router   /ajuda/topicos [get]
func (c *Controller) ListarTopicos(ctx *gin.Context) {
	role := ctx.GetString(middleware.ChaveUsuarioRole)
	response.Sucesso(ctx, c.uc.ListarTopicos(role))
}

// ObterTopico devolve um tópico específico pelo id.
// @Summary  Tópico da ajuda por id
// @Tags     Ajuda
// @Produce  json
// @Security BearerAuth
// @Param    id   path  string  true  "ID do tópico (slug)"
// @Success  200  {object}  response.RespostaPadrao{dados=Topico}  "Tópico"
// @Failure  404  {object}  response.ErroPadrao                    "Tópico não encontrado"
// @Router   /ajuda/topicos/{id} [get]
func (c *Controller) ObterTopico(ctx *gin.Context) {
	t, err := c.uc.ObterTopico(ctx.Param("id"))
	if err != nil {
		if errors.Is(err, ErrTopicoNaoEncontrado) {
			response.ErroNaoEncontrado(ctx, "tópico não encontrado")
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, t)
}

// Tour devolve as etapas do tour de boas-vindas adequadas ao papel do usuário.
// @Summary  Tour de boas-vindas
// @Tags     Ajuda
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=TourDTO}  "Etapas do tour"
// @Router   /ajuda/tour [get]
func (c *Controller) Tour(ctx *gin.Context) {
	role := ctx.GetString(middleware.ChaveUsuarioRole)
	response.Sucesso(ctx, c.uc.Tour(role))
}

// StatusIA informa ao front se há IA disponível para este usuário (mostra ou não o chat).
// @Summary  Status do assistente de IA
// @Tags     Ajuda
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao  "Disponibilidade da IA"
// @Router   /ajuda/ia/status [get]
func (c *Controller) StatusIA(ctx *gin.Context) {
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	response.Sucesso(ctx, gin.H{"ia_disponivel": c.uc.IADisponivelPara(usuarioID)})
}

// Perguntar responde uma pergunta livre do usuário usando IA (plataforma → BYOK → curado).
// @Summary  Perguntar ao assistente de IA da ajuda
// @Description Responde com IA usando a chave da plataforma (se houver) ou a BYOK do usuário.
// @Tags     Ajuda
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    body  body  PerguntarDTO  true  "Pergunta"
// @Success  200  {object}  response.RespostaPadrao{dados=RespostaIADTO}  "Resposta do assistente"
// @Failure  400  {object}  response.ErroPadrao                           "Pergunta inválida"
// @Router   /ajuda/perguntar [post]
func (c *Controller) Perguntar(ctx *gin.Context) {
	var dto PerguntarDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroRequisicao(ctx, "informe uma pergunta (até 1000 caracteres)")
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	role := ctx.GetString(middleware.ChaveUsuarioRole)
	response.Sucesso(ctx, c.uc.Perguntar(usuarioID, role, dto.Pergunta))
}
