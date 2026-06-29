// Pacote: internal/valorregistro
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de valor de registro. Recebe requisições,
//            valida os dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package valorregistro

import (
	"errors"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz falta de posse (ErrAcessoNegado) em 404; o resto em 500.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "resposta não encontrada")
		return
	}
	response.ErroInterno(ctx, err.Error())
}

// Controller gerencia os endpoints HTTP do módulo de valor de registro
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de valor de registro
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de valor de registro
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	valores := router.Group("/valores-registro")
	valores.Use(authMiddleware)
	{
		valores.POST("", c.Criar)
		valores.GET("/:id", c.BuscarPorId)
		valores.PUT("/:id", c.Atualizar)
		valores.DELETE("/:id", c.Deletar)
	}

	// Rota aninhada: listar todos os valores de um registro específico
	aninhado := router.Group("/registros-onebyone/:id/valores")
	aninhado.Use(authMiddleware)
	aninhado.GET("", c.ListarPorRegistro)
}

// Criar salva a resposta de um bloco de formulário
// @Summary      Criar valor de registro
// @Description  Salva a resposta de um bloco de formulário dentro de um registro de one-on-one.
//
//	Use valor_texto para blocos TEXT/HIGHLIGHT e valor_json para LIST/IMAGE.
//
// @Tags         Valores de Registro
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarValorRegistroDTO                                          true  "Dados da resposta"
// @Success      201   {object}  response.RespostaPadrao{dados=ValorRegistroRespostaDTO}        "Resposta salva"
// @Failure      400   {object}  response.ErroPadrao                                            "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                            "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                            "Erro interno"
// @Router       /valores-registro [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarValorRegistroDTO
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

// BuscarPorId retorna os dados de uma resposta pelo UUID
// @Summary      Buscar valor por ID
// @Description  Retorna os dados de um valor de registro ativo pelo UUID
// @Tags         Valores de Registro
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                      true  "UUID do valor"
// @Success      200  {object}  response.RespostaPadrao{dados=ValorRegistroRespostaDTO}     "Valor encontrado"
// @Failure      404  {object}  response.ErroPadrao                                         "Não encontrado"
// @Router       /valores-registro/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "resposta não encontrada")
		return
	}
	response.Sucesso(ctx, resultado)
}

// ListarPorRegistro retorna todas as respostas de um registro
// @Summary      Listar valores por registro
// @Description  Retorna todas as respostas de um registro de one-on-one, ordenadas por criação
// @Tags         Valores de Registro
// @Produce      json
// @Security     BearerAuth
// @Param        registroId  path      string                                                          true  "UUID do registro"
// @Success      200         {object}  response.RespostaPadrao{dados=[]ValorRegistroRespostaDTO}       "Lista de valores"
// @Failure      401         {object}  response.ErroPadrao                                            "Não autenticado"
// @Router       /registros-onebyone/{registroId}/valores [get]
func (c *Controller) ListarPorRegistro(ctx *gin.Context) {
	registroID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarPorRegistro(registroID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// Atualizar modifica o conteúdo de uma resposta já existente
// @Summary      Atualizar valor de registro
// @Description  Atualiza o conteúdo textual ou JSON de uma resposta existente pelo UUID
// @Tags         Valores de Registro
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                      true  "UUID do valor"
// @Param        body  body      AtualizarValorRegistroDTO                                   true  "Novo conteúdo"
// @Success      200   {object}  response.RespostaPadrao{dados=ValorRegistroRespostaDTO}     "Valor atualizado"
// @Failure      400   {object}  response.ErroPadrao                                         "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                         "Não encontrado"
// @Router       /valores-registro/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	var dto AtualizarValorRegistroDTO
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

// Deletar realiza a exclusão lógica de um valor de registro
// @Summary      Deletar valor de registro
// @Description  Realiza o soft delete de um valor de registro pelo UUID
// @Tags         Valores de Registro
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                              true  "UUID do valor"
// @Success      200  {object}  response.RespostaPadrao{dados=string}               "Deletado com sucesso"
// @Failure      401  {object}  response.ErroPadrao                                 "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                 "Não encontrado"
// @Router       /valores-registro/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	if err := c.uc.Deletar(id, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "resposta não encontrada")
		return
	}
	response.Sucesso(ctx, "valor de registro deletado com sucesso")
}
