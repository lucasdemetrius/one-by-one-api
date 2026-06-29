// Pacote: internal/equipe
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de equipe. Recebe as requisições,
//            valida os dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package equipe

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

var tiposImagemPermitidos = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// Controller gerencia os endpoints HTTP do módulo de equipe
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de equipe
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de equipe
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	equipes := router.Group("/equipes")
	equipes.Use(authMiddleware)
	{
		// Gestão restrita a LIDER/RH (defesa em profundidade); a posse fina (dono OU RH do
		// tenant) é no usecase. GET tem posse no usecase e devolve 404 a quem não é dono.
		equipes.POST("", middleware.PermitirGestaoOuRH(), c.Criar)
		equipes.GET("", c.Listar)
		equipes.GET("/:id", c.BuscarPorId)
		equipes.PUT("/:id", middleware.PermitirGestaoOuRH(), c.Atualizar)
		equipes.DELETE("/:id", middleware.PermitirGestaoOuRH(), c.Deletar)
		equipes.POST("/:id/foto", middleware.PermitirGestaoOuRH(), c.UploadFoto)
	}

	// Rota aninhada: listar equipes de uma organização específica
	// Usa /:id para consistência com o grupo /organizacoes já registrado
	aninhado := router.Group("/organizacoes/:id/equipes")
	aninhado.Use(authMiddleware)
	aninhado.GET("", c.ListarPorOrganizacao)
}

// Criar cadastra uma nova equipe vinculada ao líder autenticado
// @Summary      Criar equipe
// @Description  Cria uma nova equipe dentro de uma organização
// @Tags         Equipes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarEquipeDTO                                          true  "Dados da equipe"
// @Success      201   {object}  response.RespostaPadrao{dados=EquipeRespostaDTO}        "Equipe criada"
// @Failure      400   {object}  response.ErroPadrao                                     "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                     "Não autenticado"
// @Failure      409   {object}  response.ErroPadrao                                     "Nome de equipe já usado"
// @Failure      500   {object}  response.ErroPadrao                                     "Erro interno"
// @Router       /equipes [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarEquipeDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	usuarioIDInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := usuarioIDInterface.(string)

	resultado, err := c.uc.Criar(usuarioID, dto)
	if err != nil {
		// Nome de equipe repetido para o mesmo líder → 409 (conflito).
		if err.Error() == "já existe uma equipe com este nome" {
			response.ErroConflito(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Criado(ctx, resultado)
}

// BuscarPorId retorna os dados de uma equipe pelo UUID
// @Summary      Buscar equipe por ID
// @Description  Retorna os dados de uma equipe ativa pelo UUID
// @Tags         Equipes
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                               true  "UUID da equipe"
// @Success      200  {object}  response.RespostaPadrao{dados=EquipeRespostaDTO}     "Equipe encontrada"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                  "Não encontrada"
// @Router       /equipes/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		// Pode embrulhar SQL ("sql: no rows") — devolve mensagem fixa, sem o err técnico.
		response.ErroNaoEncontrado(ctx, "equipe não encontrada")
		return
	}
	response.Sucesso(ctx, resultado)
}

// Listar retorna todas as equipes do líder autenticado
// @Summary      Listar equipes
// @Description  Retorna todas as equipes ativas do líder autenticado
// @Tags         Equipes
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=[]EquipeRespostaDTO}   "Lista de equipes"
// @Failure      401  {object}  response.ErroPadrao                                  "Não autenticado"
// @Router       /equipes [get]
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

// ListarPorOrganizacao retorna todas as equipes de uma organização específica
// @Summary      Listar equipes por organização
// @Description  Retorna todas as equipes ativas de uma organização
// @Tags         Equipes
// @Produce      json
// @Security     BearerAuth
// @Param        organizacaoId  path      string                                             true  "UUID da organização"
// @Success      200            {object}  response.RespostaPadrao{dados=[]EquipeRespostaDTO} "Lista de equipes"
// @Failure      401            {object}  response.ErroPadrao                                "Não autenticado"
// @Failure      500            {object}  response.ErroPadrao                                "Erro interno"
// @Router       /organizacoes/{organizacaoId}/equipes [get]
func (c *Controller) ListarPorOrganizacao(ctx *gin.Context) {
	organizacaoID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarPorOrganizacao(organizacaoID, usuarioID)
	if err != nil {
		if errors.Is(err, ErrAcessoNegado) {
			response.ErroNaoEncontrado(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// Atualizar modifica os dados de uma equipe existente
// @Summary      Atualizar equipe
// @Description  Atualiza parcialmente os dados de uma equipe pelo UUID
// @Tags         Equipes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                               true  "UUID da equipe"
// @Param        body  body      AtualizarEquipeDTO                                   true  "Campos a atualizar"
// @Success      200   {object}  response.RespostaPadrao{dados=EquipeRespostaDTO}     "Equipe atualizada"
// @Failure      400   {object}  response.ErroPadrao                                  "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                  "Não encontrada"
// @Failure      409   {object}  response.ErroPadrao                                  "Nome de equipe já usado"
// @Router       /equipes/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	var dto AtualizarEquipeDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	resultado, err := c.uc.Atualizar(id, usuarioID, dto)
	if err != nil {
		if errors.Is(err, ErrAcessoNegado) {
			response.ErroNaoEncontrado(ctx, err.Error())
			return
		}
		// Nome de equipe repetido para o mesmo líder → 409 (conflito).
		if err.Error() == "já existe uma equipe com este nome" {
			response.ErroConflito(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, resultado)
}

// UploadFoto faz o upload da foto da equipe para o S3 e retorna a URL presignada
// @Summary      Upload de foto da equipe
// @Description  Recebe um arquivo de imagem (JPEG, PNG ou WebP, máx. 5MB) via multipart/form-data
// @Tags         Equipes
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                               true  "UUID da equipe"
// @Param        foto  formData  file                                                 true  "Arquivo de imagem"
// @Success      200   {object}  response.RespostaPadrao{dados=EquipeRespostaDTO}     "Foto enviada"
// @Failure      400   {object}  response.ErroPadrao                                  "Arquivo inválido"
// @Failure      404   {object}  response.ErroPadrao                                  "Não encontrada"
// @Router       /equipes/{id}/foto [post]
func (c *Controller) UploadFoto(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	const maxTamanho = 5 << 20
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxTamanho)

	arquivo, header, err := ctx.Request.FormFile("foto")
	if err != nil {
		// Erro de multipart/FormFile (ex.: body grande demais) — mensagem fixa, sem o err técnico.
		response.ErroRequisicao(ctx, "arquivo 'foto' inválido ou ausente")
		return
	}
	defer arquivo.Close()

	tipoConteudo := header.Header.Get("Content-Type")
	if !tiposImagemPermitidos[tipoConteudo] {
		response.ErroRequisicao(ctx, "tipo de imagem não suportado; use JPEG, PNG ou WebP")
		return
	}
	// Defesa extra: o Content-Type acima é forjável. Confere o tipo REAL pelos magic bytes
	// e reposiciona o ponteiro para o envio ao S3 ler o arquivo do começo.
	cabecalho := make([]byte, 512)
	lidos, _ := arquivo.Read(cabecalho)
	if _, err := arquivo.Seek(0, 0); err != nil {
		response.ErroRequisicao(ctx, "erro ao processar a imagem")
		return
	}
	if !tiposImagemPermitidos[http.DetectContentType(cabecalho[:lidos])] {
		response.ErroRequisicao(ctx, "o arquivo enviado não é uma imagem válida (JPEG, PNG ou WebP)")
		return
	}

	resultado, err := c.uc.UploadFoto(id, usuarioID, arquivo, header.Size, tipoConteudo)
	if err != nil {
		if errors.Is(err, ErrAcessoNegado) {
			response.ErroNaoEncontrado(ctx, err.Error())
			return
		}
		response.ErroInterno(ctx, err.Error())
		return
	}

	response.Sucesso(ctx, resultado)
}

// Deletar realiza a exclusão lógica de uma equipe
// @Summary      Deletar equipe
// @Description  Realiza o soft delete de uma equipe pelo UUID
// @Tags         Equipes
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                           true  "UUID da equipe"
// @Success      200  {object}  response.RespostaPadrao{dados=string}            "Deletada com sucesso"
// @Failure      401  {object}  response.ErroPadrao                              "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                              "Não encontrada"
// @Router       /equipes/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	deletadoPorInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	deletadoPor, _ := deletadoPorInterface.(string)

	if err := c.uc.Deletar(id, deletadoPor); err != nil {
		// Pode embrulhar SQL ("sql: no rows") ou ser ErrAcessoNegado — devolve mensagem
		// fixa de 404, sem revelar o err técnico nem a existência do recurso.
		response.ErroNaoEncontrado(ctx, "equipe não encontrada")
		return
	}
	response.Sucesso(ctx, "equipe deletada com sucesso")
}
