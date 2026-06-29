// Pacote: internal/colaborador
// Arquivo: controller.go
// Descrição: Controlador HTTP do módulo de colaborador. Recebe as requisições,
//            valida os dados e delega ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package colaborador

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// responderErro traduz erros do usecase em status HTTP. Falta de posse
// (ErrAcessoNegado) vira 404 de propósito — não revelamos que o recurso existe.
func responderErro(ctx *gin.Context, err error) {
	if errors.Is(err, ErrAcessoNegado) {
		response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
		return
	}
	// E-mail repetido para o mesmo líder, ou e-mail do próprio gestor → 409.
	if errors.Is(err, ErrEmailDuplicado) || errors.Is(err, ErrEmailDoGestor) {
		response.ErroConflito(ctx, err.Error())
		return
	}
	response.ErroInterno(ctx, err.Error())
}

var tiposImagemPermitidos = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// Controller gerencia os endpoints HTTP do módulo de colaborador
type Controller struct {
	// uc é a dependência do UseCase injetada via interface
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de colaborador
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de colaborador
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	cols := router.Group("/colaboradores")
	cols.Use(authMiddleware)
	{
		// Gestão: só LÍDER (defesa em profundidade — a posse ainda é checada no usecase).
		cols.POST("", middleware.ApenasLider(), c.Criar)
		cols.PUT("/:id", middleware.ApenasLider(), c.Atualizar)
		cols.DELETE("/:id", middleware.ApenasLider(), c.Deletar)
		// Leitura e foto: líder dono OU o próprio liderado (self) — sem ApenasLider.
		cols.GET("/:id", c.BuscarPorId)
		cols.POST("/:id/foto", c.UploadFoto)
		// Desligamento/reativação: gestão — só LÍDER (posse checada no usecase).
		cols.POST("/:id/desligar", middleware.ApenasLider(), c.Desligar)
		cols.POST("/:id/reativar", middleware.ApenasLider(), c.Reativar)
	}

	// "Meu colaborador": o liderado logado descobre o próprio registro de
	// colaborador (e, com ele, o id da sala do 1:1 ao vivo).
	router.GET("/meu-colaborador", authMiddleware, c.MeuColaborador)

	// Importação em lote (CSV). Rota top-level de propósito: dentro de /colaboradores
	// um caminho estático colidiria com o param /:id no Gin. Só LÍDER.
	router.POST("/importar-liderados", authMiddleware, middleware.ApenasLider(), c.ImportarLote)

	// Rotas aninhadas de listagem (PII de liderados) — só o LÍDER dono.
	// Usa /:id para consistência com os grupos /equipes e /organizacoes já registrados
	aninhadoEquipe := router.Group("/equipes/:id/colaboradores")
	aninhadoEquipe.Use(authMiddleware, middleware.ApenasLider())
	aninhadoEquipe.GET("", c.ListarPorEquipe)

	aninhadoOrg := router.Group("/organizacoes/:id/colaboradores")
	aninhadoOrg.Use(authMiddleware, middleware.ApenasLider())
	aninhadoOrg.GET("", c.ListarPorOrganizacao)
}

// MeuColaborador devolve o colaborador vinculado ao usuário autenticado (liderado).
// @Summary      Meu colaborador
// @Description  Retorna o registro de colaborador do liderado logado (e o id da sala do 1:1).
// @Tags         Colaboradores
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.RespostaPadrao{dados=ColaboradorRespostaDTO}  "Encontrado"
// @Failure      404  {object}  response.ErroPadrao                                    "Sem vínculo de colaborador"
// @Router       /meu-colaborador [get]
func (c *Controller) MeuColaborador(ctx *gin.Context) {
	id, _ := ctx.Get(middleware.ChaveUsuarioID)
	usuarioID, _ := id.(string)
	res, err := c.uc.BuscarPorUsuarioID(usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "nenhum colaborador vinculado à sua conta")
		return
	}
	response.Sucesso(ctx, res)
}

// ImportarLote cria vários liderados de uma vez a partir de um CSV (nome,email)
// @Summary      Importar liderados (lote/CSV)
// @Description  Cria vários liderados numa equipe do líder. Valida linha a linha; retorna criados + erros.
// @Tags         Colaboradores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      ImportarColaboradoresDTO                                     true  "Equipe-alvo + linhas"
// @Success      200   {object}  response.RespostaPadrao{dados=ResultadoImportacaoDTO}         "Resultado do import"
// @Failure      400   {object}  response.ErroPadrao                                          "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                          "Equipe/organização não é do líder"
// @Router       /importar-liderados [post]
func (c *Controller) ImportarLote(ctx *gin.Context) {
	var dto ImportarColaboradoresDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	res, err := c.uc.ImportarLote(dto.Itens, dto.OrganizacaoID, dto.EquipeID, usuarioID)
	if err != nil {
		responderErro(ctx, err) // ErrAcessoNegado → 404
		return
	}
	response.Sucesso(ctx, res)
}

// Criar cadastra um novo colaborador
// @Summary      Criar colaborador
// @Description  Cria um novo colaborador dentro de uma equipe e organização
// @Tags         Colaboradores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CriarColaboradorDTO                                          true  "Dados do colaborador"
// @Success      201   {object}  response.RespostaPadrao{dados=ColaboradorRespostaDTO}        "Colaborador criado"
// @Failure      400   {object}  response.ErroPadrao                                          "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                          "Não autenticado"
// @Failure      500   {object}  response.ErroPadrao                                          "Erro interno"
// @Router       /colaboradores [post]
func (c *Controller) Criar(ctx *gin.Context) {
	var dto CriarColaboradorDTO
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

// BuscarPorId retorna os dados de um colaborador pelo UUID
// @Summary      Buscar colaborador por ID
// @Description  Retorna os dados de um colaborador ativo pelo UUID
// @Tags         Colaboradores
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                     true  "UUID do colaborador"
// @Success      200  {object}  response.RespostaPadrao{dados=ColaboradorRespostaDTO}      "Colaborador encontrado"
// @Failure      404  {object}  response.ErroPadrao                                        "Não encontrado"
// @Router       /colaboradores/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.BuscarPorId(id, usuarioID)
	if err != nil {
		response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
		return
	}
	response.Sucesso(ctx, resultado)
}

// ListarPorEquipe retorna todos os colaboradores de uma equipe
// @Summary      Listar colaboradores por equipe
// @Description  Retorna todos os colaboradores ativos de uma equipe
// @Tags         Colaboradores
// @Produce      json
// @Security     BearerAuth
// @Param        equipeId  path      string                                                         true  "UUID da equipe"
// @Success      200       {object}  response.RespostaPadrao{dados=[]ColaboradorRespostaDTO}         "Lista de colaboradores"
// @Failure      401       {object}  response.ErroPadrao                                            "Não autenticado"
// @Router       /equipes/{equipeId}/colaboradores [get]
func (c *Controller) ListarPorEquipe(ctx *gin.Context) {
	equipeID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarPorEquipe(equipeID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// ListarPorOrganizacao retorna todos os colaboradores de uma organização
// @Summary      Listar colaboradores por organização
// @Description  Retorna todos os colaboradores ativos de uma organização
// @Tags         Colaboradores
// @Produce      json
// @Security     BearerAuth
// @Param        organizacaoId  path      string                                                         true  "UUID da organização"
// @Success      200            {object}  response.RespostaPadrao{dados=[]ColaboradorRespostaDTO}         "Lista de colaboradores"
// @Failure      401            {object}  response.ErroPadrao                                            "Não autenticado"
// @Router       /organizacoes/{organizacaoId}/colaboradores [get]
func (c *Controller) ListarPorOrganizacao(ctx *gin.Context) {
	organizacaoID := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.ListarPorOrganizacao(organizacaoID, usuarioID)
	if err != nil {
		responderErro(ctx, err)
		return
	}
	response.Sucesso(ctx, resultado)
}

// Atualizar modifica os dados de um colaborador existente
// @Summary      Atualizar colaborador
// @Description  Atualiza parcialmente os dados de um colaborador pelo UUID
// @Tags         Colaboradores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                     true  "UUID do colaborador"
// @Param        body  body      AtualizarColaboradorDTO                                    true  "Campos a atualizar"
// @Success      200   {object}  response.RespostaPadrao{dados=ColaboradorRespostaDTO}      "Colaborador atualizado"
// @Failure      400   {object}  response.ErroPadrao                                        "Dados inválidos"
// @Failure      404   {object}  response.ErroPadrao                                        "Não encontrado"
// @Router       /colaboradores/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	var dto AtualizarColaboradorDTO
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

// UploadFoto faz o upload da foto do colaborador para o S3 e retorna a URL presignada
// @Summary      Upload de foto do colaborador
// @Description  Recebe um arquivo de imagem (JPEG, PNG ou WebP, máx. 5MB) via multipart/form-data
// @Tags         Colaboradores
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                     true  "UUID do colaborador"
// @Param        foto  formData  file                                                       true  "Arquivo de imagem"
// @Success      200   {object}  response.RespostaPadrao{dados=ColaboradorRespostaDTO}      "Foto enviada"
// @Failure      400   {object}  response.ErroPadrao                                        "Arquivo inválido"
// @Failure      404   {object}  response.ErroPadrao                                        "Não encontrado"
// @Router       /colaboradores/{id}/foto [post]
func (c *Controller) UploadFoto(ctx *gin.Context) {
	id := ctx.Param("id")

	const maxTamanho = 5 << 20
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxTamanho)

	arquivo, header, err := ctx.Request.FormFile("foto")
	if err != nil {
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

	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	resultado, err := c.uc.UploadFoto(id, usuarioID, arquivo, header.Size, tipoConteudo)
	if err != nil {
		responderErro(ctx, err)
		return
	}

	response.Sucesso(ctx, resultado)
}

// Desligar marca um liderado como inativo (saída da empresa/equipe), preservando o histórico
// @Summary      Desligar (inativar) liderado
// @Description  Marca o liderado como inativo com uma data de desligamento (default: hoje). O registro é preservado.
// @Tags         Colaboradores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                     true  "UUID do colaborador"
// @Param        body  body      DesligarColaboradorDTO                                     false "Data de desligamento (opcional)"
// @Success      200   {object}  response.RespostaPadrao{dados=string}                      "Desligado"
// @Failure      404   {object}  response.ErroPadrao                                        "Não encontrado"
// @Router       /colaboradores/{id}/desligar [post]
func (c *Controller) Desligar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	var dto DesligarColaboradorDTO
	_ = ctx.ShouldBindJSON(&dto) // corpo é opcional

	desligadoEm := time.Now()
	if dto.DataDesligamento != "" {
		// Interpreta ao meio-dia local para conversões de fuso não virarem o dia.
		data, err := time.ParseInLocation("2006-01-02", dto.DataDesligamento, time.Local)
		if err != nil {
			response.ErroRequisicao(ctx, "data de desligamento inválida — use YYYY-MM-DD")
			return
		}
		desligadoEm = data.Add(12 * time.Hour)
	}

	if err := c.uc.Desligar(id, usuarioID, desligadoEm); err != nil {
		response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
		return
	}
	response.Sucesso(ctx, "liderado desligado")
}

// Reativar volta um liderado desligado para ativo
// @Summary      Reativar liderado
// @Description  Limpa a data de desligamento, reativando o liderado
// @Tags         Colaboradores
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                             true  "UUID do colaborador"
// @Success      200  {object}  response.RespostaPadrao{dados=string}              "Reativado"
// @Failure      404  {object}  response.ErroPadrao                                "Não encontrado"
// @Router       /colaboradores/{id}/reativar [post]
func (c *Controller) Reativar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)
	if err := c.uc.Reativar(id, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
		return
	}
	response.Sucesso(ctx, "liderado reativado")
}

// Deletar realiza a exclusão lógica de um colaborador
// @Summary      Deletar colaborador
// @Description  Realiza o soft delete de um colaborador pelo UUID
// @Tags         Colaboradores
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                             true  "UUID do colaborador"
// @Success      200  {object}  response.RespostaPadrao{dados=string}              "Deletado com sucesso"
// @Failure      401  {object}  response.ErroPadrao                                "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                "Não encontrado"
// @Router       /colaboradores/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")
	usuarioID := ctx.GetString(middleware.ChaveUsuarioID)

	if err := c.uc.Deletar(id, usuarioID); err != nil {
		response.ErroNaoEncontrado(ctx, "colaborador não encontrado")
		return
	}
	response.Sucesso(ctx, "colaborador deletado com sucesso")
}
