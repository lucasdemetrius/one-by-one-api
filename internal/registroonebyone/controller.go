// Pacote: internal/registroonebyone
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de registro de one-on-one. Ao criar um
//            registro, o template é resolvido automaticamente pelo UseCase sem
//            intervenção do cliente.
// Autor: OneByOne API
// Criado em: 2025

package registroonebyone

import (
	"errors"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz falta de posse (ErrAcessoNegado) em 404; o resto em 500.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "registro não encontrado")
		return
	}
	response.ErroInterno(ctx, err.Error())
}

// Controller gerencia os endpoints HTTP do módulo de registro de one-on-one
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de registro de one-on-one
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de registro de one-on-one
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	registros := router.Group("/registros-onebyone")
	registros.Use(authMiddleware)
	{
		registros.POST("", c.Criar)
		registros.GET("/:id", c.BuscarPorId)
		registros.DELETE("/:id", c.Deletar)
	}

	// Rota aninhada: listar registros de uma reunião específica
	aninhado := router.Group("/onebyone/:id/registros")
	aninhado.Use(authMiddleware)
	aninhado.GET("", c.ListarPorOneByOne)
}

// Criar abre um novo formulário de registro para uma reunião
// @Summary      Criar registro de one-on-one
// @Description  Abre um novo registro para uma reunião. O template é resolvido automaticamente
//
//	seguindo a regra de herança: colaborador → equipe → organização → padrão do líder.
//
// @Tags         Registros de One-on-One
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarRegistroOneByOneDTO                                          true  "Dados do registro"
// @Success      201   {object}  response.RespostaPadrao{dados=RegistroOneByOneRespostaDTO}        "Registro criado"
// @Failure      400   {object}  response.ErroPadrao                                              "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                              "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                              "Nenhum template configurado"
// @Router       /registros-onebyone [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarRegistroOneByOneDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.Criar(dto, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Criado(ctx, resultado)
}

// BuscarPorId retorna os dados de um registro pelo UUID
// @Summary      Buscar registro por ID
// @Description  Retorna os dados de um registro de one-on-one ativo pelo UUID
// @Tags         Registros de One-on-One
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                         true  "UUID do registro"
// @Success      200  {object}  response.RespostaPadrao{dados=RegistroOneByOneRespostaDTO}      "Registro encontrado"
// @Failure      404  {object}  response.ErroPadrao                                            "Não encontrado"
// @Router       /registros-onebyone/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "registro não encontrado")
		return
	}
	response.Sucesso(ctx, resultado)
}

// ListarPorOneByOne retorna todos os registros de uma reunião
// @Summary      Listar registros por one-on-one
// @Description  Retorna todos os registros ativos de uma reunião, do mais recente ao mais antigo
// @Tags         Registros de One-on-One
// @Produce      json
// @Security     BearerAuth
// @Param        oneaoneId  path      string                                                              true  "UUID da reunião"
// @Success      200        {object}  response.RespostaPadrao{dados=[]RegistroOneByOneRespostaDTO}         "Lista de registros"
// @Failure      401        {object}  response.ErroPadrao                                                "Não autenticado"
// @Router       /onebyone/{oneaoneId}/registros [get]
func (c *Controller) ListarPorOneByOne(ctx *gin.Context) {
	oneaoneID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarPorOneByOne(oneaoneID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// Deletar realiza a exclusão lógica de um registro de one-on-one
// @Summary      Deletar registro
// @Description  Realiza o soft delete de um registro de one-on-one pelo UUID
// @Tags         Registros de One-on-One
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                               true  "UUID do registro"
// @Success      200  {object}  response.RespostaPadrao{dados=string}                "Deletado com sucesso"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                  "Não encontrado"
// @Router       /registros-onebyone/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	if err := c.uc.Deletar(id, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "registro não encontrado")
		return
	}
	response.Sucesso(ctx, "registro de one-on-one deletado com sucesso")
}
