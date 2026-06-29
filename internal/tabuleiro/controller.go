// Pacote: internal/tabuleiro
// Arquivo: controller.go
// Descrição: Endpoints HTTP do tabuleiro do 1:1. NÃO usa ApenasLider — o liderado
//            também participa do board ao vivo, então ambos (líder e liderado)
//            podem ler e salvar. A posse (PodeAcessar) é checada no usecase.
// Autor: OneByOne API
// Criado em: 2026

package tabuleiro

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller gerencia os endpoints do tabuleiro.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de tabuleiro.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "não encontrado")
		return
	}
	// Qualquer outro erro aqui é técnico (vem do banco via PodeAcessar/Buscar/Salvar):
	// loga no servidor e devolve resposta genérica, sem vazar detalhe de SQL ao cliente.
	response.ErroInterno(ctx, err.Error())
}

// RegistrarRotas registra as rotas do tabuleiro (ler/salvar por liderado).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	router.GET("/colaboradores/:id/tabuleiro", authMiddleware, c.Obter)
	router.PUT("/colaboradores/:id/tabuleiro", authMiddleware, c.Salvar)
}

func (c *Controller) Obter(ctx *gin.Context) {
	res, err := c.uc.Obter(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID))
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, res)
}

func (c *Controller) Salvar(ctx *gin.Context) {
	var dto SalvarTabuleiroDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	if err := c.uc.Salvar(ctx.Param("id"), ctx.GetString(middleware.ChaveUsuarioID), dto); err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, "tabuleiro salvo")
}
