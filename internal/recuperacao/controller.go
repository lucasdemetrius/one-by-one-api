// Pacote: internal/recuperacao
// Arquivo: controller.go
// Descrição: Endpoints PÚBLICOS do "esqueci minha senha" (a pessoa não está logada):
//            pedir o link, validar o link e redefinir a senha com o código.
// Autor: OneByOne API
// Criado em: 2026

package recuperacao

import (
	"errors"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/response"
)

// Controller expõe os endpoints de recuperação de senha.
type Controller struct {
	uc UseCase
}

// NovoController cria o Controller de recuperação.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas públicas (sem auth — a pessoa está deslogada).
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, protecaoAuth ...gin.HandlerFunc) {
	publicas := router.Group("", protecaoAuth...)
	publicas.POST("/auth/recuperar-senha", c.Solicitar)
	publicas.POST("/recuperacoes/:token/redefinir", c.Redefinir)
	publicas.GET("/recuperacoes/:token", c.Validar)
}

// Solicitar recebe o e-mail e dispara (se a conta existir) o link de redefinição.
// Responde sempre a mesma mensagem — não revela se o e-mail tem conta (anti-enumeração).
func (c *Controller) Solicitar(ctx *gin.Context) {
	var dto SolicitarDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	_ = c.uc.Solicitar(dto)
	response.Sucesso(ctx, gin.H{
		"mensagem": "Se este e-mail tiver conta, enviamos um link para redefinir a senha.",
	})
}

// Validar diz ao front se o link ainda é válido (para mostrar o formulário ou um aviso).
func (c *Controller) Validar(ctx *gin.Context) {
	valido, _ := c.uc.ValidarToken(ctx.Param("token"))
	response.Sucesso(ctx, StatusTokenDTO{Valido: valido})
}

// Redefinir valida o token + código e troca a senha (com checagem de complexidade).
func (c *Controller) Redefinir(ctx *gin.Context) {
	var dto RedefinirDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	if err := c.uc.Redefinir(ctx.Param("token"), dto); err != nil {
		switch {
		case errors.Is(err, ErrTokenInvalido):
			response.ErroRequisicao(ctx, "Este link é inválido ou expirou. Peça um novo.")
		case errors.Is(err, ErrCodigoInvalido):
			response.ErroRequisicao(ctx, "Código inválido. Confira o e-mail e tente novamente.")
		case errors.Unwrap(err) != nil:
			// Erro TÉCNICO: vem embrulhado com %w (ex.: fmt.Errorf("...: %w", sqlErr)
			// do banco, ou falha do bcrypt). Não pode vazar ao cliente — ErroInterno
			// loga no servidor e devolve uma mensagem genérica.
			response.ErroInterno(ctx, err.Error())
		default:
			// Erro de NEGÓCIO: mensagem amigável de complexidade da nova senha vinda do
			// módulo usuario (senha.Validar) — não embrulha outro erro, então deve aparecer.
			response.ErroRequisicao(ctx, err.Error())
		}
		return
	}
	response.Sucesso(ctx, gin.H{"mensagem": "Senha redefinida! Faça login com a nova senha."})
}
