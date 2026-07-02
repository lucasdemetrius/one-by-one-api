// Pacote: internal/onebyone
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de one-on-one. Recebe as requisições,
//            valida os dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package onebyone

import (
	"errors"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz falta de posse (ErrAcessoNegado) em 404; o resto em 500.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "one-on-one não encontrado")
		return
	}
	response.ErroInterno(ctx, err.Error())
}

// Controller gerencia os endpoints HTTP do módulo de one-on-one
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de one-on-one
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de one-on-one
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	reunioes := router.Group("/onebyone")
	reunioes.Use(authMiddleware)
	{
		// Criar/editar/excluir 1:1 é gestão → ApenasLider (defesa em profundidade, além
		// da posse no usecase). Liderado (COLABORADOR) não cria nem mexe em reuniões.
		reunioes.POST("", middleware.ApenasLider(), c.Criar)
		// Encerrar é gestão (só o gestor encerra) → ApenasLider de defesa em profundidade.
		reunioes.POST("/encerrar", middleware.ApenasLider(), c.Encerrar)
		reunioes.GET("", c.Listar)
		reunioes.GET("/:id", c.BuscarPorId)
		reunioes.PUT("/:id", middleware.ApenasLider(), c.Atualizar)
		reunioes.DELETE("/:id", middleware.ApenasLider(), c.Deletar)
	}
}

// Encerrar registra um 1:1 como realizado (cria a linha no livro-razão)
// @Summary      Encerrar one-on-one (marcar como realizado)
// @Description  Registra que o 1:1 com o colaborador foi realizado agora. Idempotente por dia.
// @Tags         One-on-One
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      EncerrarOneByOneDTO                                  true  "Colaborador do 1:1"
// @Success      201   {object}  response.RespostaPadrao{dados=OneByOneRespostaDTO}    "Registrado como realizado"
// @Failure      400   {object}  response.ErroPadrao                                  "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                  "Colaborador não encontrado / sem acesso"
// @Router       /onebyone/encerrar [post]
func (c *Controller) Encerrar(ctx *gin.Context) {
	var dto EncerrarOneByOneDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.Encerrar(usuarioID, dto)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, resultado)
}

// Criar agenda uma nova reunião one-on-one
// @Summary      Criar one-on-one
// @Description  Agenda uma nova reunião one-on-one entre líder e colaborador
// @Tags         One-on-One
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarOneByOneDTO                                          true  "Dados da reunião"
// @Success      201   {object}  response.RespostaPadrao{dados=OneByOneRespostaDTO}        "Reunião agendada"
// @Failure      400   {object}  response.ErroPadrao                                      "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                      "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                      "Erro interno"
// @Router       /onebyone [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarOneByOneDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.Criar(usuarioID, dto)
	if err != nil {
		// Falta de posse (colaborador de outro líder) → 404, não 500.
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, resultado)
}

// BuscarPorId retorna os dados de uma reunião pelo UUID
// @Summary      Buscar one-on-one por ID
// @Description  Retorna os dados de uma reunião ativa pelo UUID
// @Tags         One-on-One
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                               true  "UUID da reunião"
// @Success      200  {object}  response.RespostaPadrao{dados=OneByOneRespostaDTO}    "Reunião encontrada"
// @Failure      404  {object}  response.ErroPadrao                                  "Não encontrada"
// @Router       /onebyone/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "one-on-one não encontrado")
		return
	}
	response.Sucesso(ctx, resultado)
}

// Listar retorna todas as reuniões do líder autenticado
// @Summary      Listar one-on-ones
// @Description  Retorna todas as reuniões ativas do líder autenticado, da mais recente à mais antiga
// @Tags         One-on-One
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=[]OneByOneRespostaDTO}  "Lista de reuniões"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Router       /onebyone [get]
func (c *Controller) Listar(ctx *gin.Context) {
	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.ListarPorUsuario(usuarioID)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// Atualizar modifica status, recorrência ou data de uma reunião
// @Summary      Atualizar one-on-one
// @Description  Atualiza status, recorrência ou data agendada de uma reunião pelo UUID
// @Tags         One-on-One
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                               true  "UUID da reunião"
// @Param        body  body      AtualizarOneByOneDTO                                  true  "Campos a atualizar"
// @Success      200   {object}  response.RespostaPadrao{dados=OneByOneRespostaDTO}    "Reunião atualizada"
// @Failure      400   {object}  response.ErroPadrao                                  "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                  "Não encontrada"
// @Router       /onebyone/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	var dto AtualizarOneByOneDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.Atualizar(id, usuarioID, dto)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// Deletar realiza a exclusão lógica de uma reunião
// @Summary      Deletar one-on-one
// @Description  Realiza o soft delete de uma reunião pelo UUID
// @Tags         One-on-One
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                           true  "UUID da reunião"
// @Success      200  {object}  response.RespostaPadrao{dados=string}            "Deletada com sucesso"
// @Failure      401  {object}  response.ErroPadrao                              "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                              "Não encontrada"
// @Router       /onebyone/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	if err := c.uc.Deletar(id, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "one-on-one não encontrado")
		return
	}
	response.Sucesso(ctx, "reunião one-on-one deletada com sucesso")
}
