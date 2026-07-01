// Pacote: internal/classificacao
// Arquivo: controller.go
// Descrição: Controlador HTTP da classificação 9-box. Rotas protegidas por JWT.
// Autor: OneByOne API
// Criado em: 2025

package classificacao

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz falta de posse em 404; demais erros em 500.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "não encontrado")
		return
	}
	response.ErroInterno(ctx, err.Error())
}

// Controller gerencia os endpoints da classificação.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de classificação.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas (todas protegidas por JWT).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Classificar (definir nota 9-box): só o LÍDER dono (defesa em profundidade + posse no usecase).
	router.PUT("/colaboradores/:id/classificacao", authMiddleware, middleware.ApenasLider(), c.Definir)
	// Remover da 9-box (tira o liderado da matriz, volta para "A classificar"): só o LÍDER dono.
	router.DELETE("/colaboradores/:id/classificacao", authMiddleware, middleware.ApenasLider(), c.Remover)
	// Listar a 9-box de uma organização: GESTOR dono OU o RH do tenant (visão total, só leitura).
	// A posse no usecase (OrganizacaoPertenceAoLider) já é RH-aware.
	router.GET("/organizacoes/:id/classificacoes", authMiddleware, middleware.PermitirGestaoOuRH(), c.ListarPorOrganizacao)
}

// Definir posiciona um liderado na 9-box.
func (c *Controller) Definir(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	var dto DefinirClassificacaoDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.Definir(colaboradorID, dto, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

// Remover tira um liderado da 9-box (volta para "A classificar").
func (c *Controller) Remover(ctx *gin.Context) {
	colaboradorID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	if err := c.uc.Remover(colaboradorID, usuarioID); err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, gin.H{"removido": true})
}

// ListarPorOrganizacao devolve as classificações dos liderados da organização.
func (c *Controller) ListarPorOrganizacao(ctx *gin.Context) {
	organizacaoID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.ListarPorOrganizacao(organizacaoID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}
