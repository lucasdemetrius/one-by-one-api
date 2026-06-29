// Pacote: internal/convite
// Arquivo: controller.go
// Descrição: Controlador HTTP dos convites de liderado. Expõe a geração (protegida,
//            para o gestor) e as rotas públicas que o liderado usa para ver e
//            aceitar o convite.
// Autor: OneByOne API
// Criado em: 2025

package convite

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
	"onebyone-api/pkg/senha"
)

// Controller gerencia os endpoints HTTP do módulo de convite.
type Controller struct {
	uc UseCase
}

// NovoController cria o controlador de convites.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas do módulo de convite.
// A geração é protegida (gestor autenticado); ver e aceitar são públicas.
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, protecaoAuth ...gin.HandlerFunc) {
	// Públicas — o liderado ainda não tem login. Sob protecaoAuth (rate-limit + reCAPTCHA)
	// para que nem a busca por token (enumeração) nem o aceite fiquem sem proteção.
	publicas := router.Group("", protecaoAuth...)
	publicas.GET("/convites/:token", c.BuscarPublico)
	publicas.POST("/convites/:token/aceitar", c.Aceitar)

	// Protegida — o gestor gera um convite para um colaborador SEU (posse checada
	// no usecase). ApenasLider barra contas COLABORADOR de imediato.
	router.POST("/colaboradores/:id/convite", authMiddleware, middleware.ApenasLider(), c.Gerar)
}

// Gerar cria um convite para o colaborador informado.
// @Summary      Gerar convite de liderado
// @Description  Gera um link (UUID) e um código para o liderado acessar. O código
//
//	só aparece nesta resposta — o gestor o compartilha com o liderado.
//
// @Tags         Convites
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                              true  "UUID do colaborador"
// @Success      201  {object}  response.RespostaPadrao{dados=ConviteGeradoDTO}      "Convite gerado"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Failure      500  {object}  response.ErroPadrao                                  "Erro interno"
// @Router       /colaboradores/{id}/convite [post]
func (c *Controller) Gerar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.Gerar(id, usuarioID)
	if err != nil {
		// Único caso de negócio (não-existência/posse): mensagem fixa amigável → 404.
		// O resto (PertenceAoLider que embrulha SQL, falhas de gerar/criar convite) é
		// técnico → ErroInterno (loga o detalhe no servidor, devolve genérico ao cliente).
		if err.Error() == "colaborador não encontrado" {
			response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, res)
}

// BuscarPublico devolve os dados públicos do convite (para a tela do liderado).
// @Summary      Ver convite (público)
// @Description  Retorna os dados públicos do convite a partir do token do link.
// @Tags         Convites
// @Produce      json
// @Param        token  path      string                                          true  "Token (UUID) do convite"
// @Success      200    {object}  response.RespostaPadrao{dados=ConvitePublicoDTO} "Convite encontrado"
// @Failure      404    {object}  response.ErroPadrao                             "Convite não encontrado"
// @Router       /convites/{token} [get]
func (c *Controller) BuscarPublico(ctx *gin.Context) {
	token := ctx.Param("token")
	res, err := c.uc.BuscarPublico(token)
	if err != nil {
		// O usecase já traduz a falha (que embrulha SQL) numa mensagem fixa; usamos
		// uma mensagem fixa amigável SEM err.Error() para nunca vazar detalhe técnico.
		response.ErroNaoEncontrado(ctx, "convite não encontrado")
		return
	}
	response.Sucesso(ctx, res)
}

// Aceitar processa o aceite do convite pelo liderado.
// @Summary      Aceitar convite
// @Description  Valida o código, cria/usa a conta do liderado, vincula ao
//
//	colaborador e devolve o token JWT para acesso imediato.
//
// @Tags         Convites
// @Accept       json
// @Produce      json
// @Param        token  path      string             true  "Token (UUID) do convite"
// @Param        body   body      AceitarConviteDTO  true  "Código e senha de acesso"
// @Success      200    {object}  response.RespostaPadrao  "Aceito — retorna token e usuário"
// @Failure      400    {object}  response.ErroPadrao      "Código inválido, convite expirado, etc."
// @Router       /convites/{token}/aceitar [post]
func (c *Controller) Aceitar(ctx *gin.Context) {
	token := ctx.Param("token")

	var dto AceitarConviteDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	// Complexidade da senha do liderado (≥ 8, com maiúscula, minúscula e número).
	if err := senha.Validar(dto.Senha); err != nil {
		response.ErroRequisicao(ctx, err.Error())
		return
	}

	res, err := c.uc.Aceitar(token, dto)
	if err != nil {
		// A falha ao vincular a conta embrulha um erro técnico (banco/IO) com %w →
		// é técnica e NÃO pode chegar ao cliente: vai para ErroInterno (loga o detalhe,
		// devolve genérico). Os demais erros do aceite são de negócio (mensagens fixas
		// amigáveis: convite inválido/usado/expirado, código inválido, etc.) → 400.
		if strings.HasPrefix(err.Error(), "erro ao vincular a conta ao colaborador") {
			response.ErroInterno(ctx, err.Error())
			return
		}
		response.Erro(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}
