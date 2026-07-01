// Pacote: internal/usuario
// Arquivo: controller.go
// Descrição: Controlador responsável por receber as requisições HTTP
//            relacionadas aos usuários, validar os dados de entrada
//            e repassá-los ao UseCase. Nunca acessa o banco diretamente.
// Autor: OneByOne API
// Criado em: 2025

package usuario

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
	"onebyone-api/pkg/senha"
)

// tiposImagemPermitidos lista os Content-Types aceitos no upload de fotos
var tiposImagemPermitidos = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// Controller gerencia os endpoints HTTP do módulo de usuário
type Controller struct {
	// uc é a dependência do UseCase injetada via interface (nunca a implementação direta)
	uc UseCase
}

// NovoController cria e retorna uma nova instância do Controller de usuários
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra todas as rotas HTTP do módulo de usuários no grupo informado.
// A rota de login é pública; as demais exigem token JWT válido.
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, protecaoAuth ...gin.HandlerFunc) {
	// Rotas públicas — sem auth, mas com rate-limit + reCAPTCHA (anti brute-force/bots).
	publicas := router.Group("", protecaoAuth...)
	publicas.POST("/auth/login", c.Login)
	publicas.POST("/auth/registrar", c.Registrar)
	publicas.POST("/auth/google", c.LoginGoogle)

	// Rotas protegidas — SELF-SERVICE: cada usuário só acessa a PRÓPRIA conta.
	// A criação/gestão de gestores pelo RH vive no módulo /rh; a de liderados, no
	// fluxo de convite. Não há mais POST/GET genéricos em /usuarios (eram abertos a
	// qualquer logado — listar todos, criar, escalar papel, deletar qualquer conta).
	usuarios := router.Group("/usuarios")
	usuarios.Use(authMiddleware)
	{
		usuarios.GET("/:id", c.BuscarPorId)
		usuarios.PUT("/:id", c.Atualizar)
		usuarios.DELETE("/:id", c.Deletar)
		usuarios.POST("/:id/foto", c.UploadFoto)
	}
}

// Login autentica um usuário com e-mail e senha e retorna um token JWT
// @Summary      Login do usuário
// @Description  Autentica o usuário com e-mail e senha. Retorna um token JWT
//
//	para ser usado no cabeçalho Authorization das rotas protegidas.
//
// @Tags         Autenticação
// @Accept       json
// @Produce      json
// @Param        body  body      LoginDTO                                         true  "Credenciais de login"
// @Success      200   {object}  response.RespostaPadrao{dados=LoginRespostaDTO}  "Login realizado com sucesso"
// @Failure      400   {object}  response.ErroPadrao                              "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                              "Credenciais inválidas"
// @Failure      500   {object}  response.ErroPadrao                              "Erro interno do servidor"
// @Router       /auth/login [post]
func (c *Controller) Login(ctx *gin.Context) {
	var dto LoginDTO

	// Valida o corpo JSON conforme as regras de binding definidas no DTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	resultado, err := c.uc.Login(dto)
	if err != nil {
		// "credenciais inválidas" é a mensagem de negócio (segura). Qualquer outro erro
		// (ex.: falha técnica ao assinar o token) → 500 genérico, sem vazar o detalhe.
		if err.Error() == "credenciais inválidas" {
			response.Erro(ctx, http.StatusUnauthorized, "credenciais inválidas")
		} else {
			response.ErroInterno(ctx, err.Error())
		}
		return
	}

	// Atribui o login ao usuário no contexto: a rota é pública (sem authMiddleware), então
	// o middleware global de auditoria não saberia QUEM logou. Setando a chave aqui, o evento
	// LOGIN passa a ter usuario_id — é o que alimenta os "acessos por usuário" no painel admin.
	ctx.Set(middleware.ChaveUsuarioID, resultado.Usuario.ID)

	response.Sucesso(ctx, resultado)
}

// Registrar permite que um novo usuário crie sua própria conta sem autenticação prévia
// @Summary      Registrar usuário
// @Description  Rota pública de auto-cadastro. Cria a conta de um Gestor (LIDER, padrão)
//
//	ou de um RH (role=RH). Liderado (COLABORADOR) NÃO se auto-cadastra — entra por convite.
//	O vínculo gestor→RH nunca vem do corpo; é derivado no servidor (RH nasce raiz vazia).
//
// @Tags         Autenticação
// @Accept       json
// @Produce      json
// @Param        body  body      CriarUsuarioDTO                                        true  "Dados do novo usuário"
// @Success      201   {object}  response.RespostaPadrao{dados=UsuarioRespostaDTO}      "Usuário criado com sucesso"
// @Failure      400   {object}  response.ErroPadrao                                    "Dados inválidos"
// @Failure      409   {object}  response.ErroPadrao                                    "E-mail já cadastrado"
// @Failure      500   {object}  response.ErroPadrao                                    "Erro interno do servidor"
// @Router       /auth/registrar [post]
func (c *Controller) Registrar(ctx *gin.Context) {
	var dto CriarUsuarioDTO

	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	// Complexidade da senha (≥ 8, com maiúscula, minúscula e número).
	if err := senha.Validar(dto.Password); err != nil {
		response.ErroRequisicao(ctx, err.Error())
		return
	}

	resultado, err := c.uc.Registrar(dto)
	if err != nil {
		switch {
		case errors.Is(err, ErrPapelInvalidoNoCadastro):
			// Papel não permitido no auto-cadastro (ex.: COLABORADOR) → 400
			response.ErroRequisicao(ctx, err.Error())
		case err.Error() == "já existe um usuário com este e-mail":
			response.ErroConflito(ctx, err.Error())
		default:
			response.ErroInterno(ctx, err.Error())
		}
		return
	}

	response.Criado(ctx, resultado)
}

// LoginGoogle autentica um usuário via Google (OAuth) e retorna um token JWT
// @Summary      Login com Google
// @Description  Recebe o "credential" (ID token) do Google Identity Services, valida
//
//	no servidor e retorna o mesmo JWT do login por senha. Se o e-mail ainda não tiver
//	conta, cria uma conta de Gestor (LIDER). Requer GOOGLE_CLIENT_ID configurado.
//
// @Tags         Autenticação
// @Accept       json
// @Produce      json
// @Param        body  body      LoginGoogleDTO                                   true  "ID token do Google"
// @Success      200   {object}  response.RespostaPadrao{dados=LoginRespostaDTO}  "Login realizado com sucesso"
// @Failure      400   {object}  response.ErroPadrao                              "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                              "Credenciais inválidas"
// @Failure      503   {object}  response.ErroPadrao                              "Login com Google indisponível"
// @Router       /auth/google [post]
func (c *Controller) LoginGoogle(ctx *gin.Context) {
	var dto LoginGoogleDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	resultado, err := c.uc.LoginGoogle(dto)
	if err != nil {
		switch err.Error() {
		case "credenciais inválidas":
			response.Erro(ctx, http.StatusUnauthorized, "credenciais inválidas")
		case "login com Google não está configurado":
			response.Erro(ctx, http.StatusServiceUnavailable, "login com Google não está disponível")
		default:
			response.ErroInterno(ctx, err.Error())
		}
		return
	}

	// Mesmo do Login por senha: atribui o usuário no contexto para o middleware de
	// auditoria registrar QUEM logou (a rota é pública, sem authMiddleware).
	ctx.Set(middleware.ChaveUsuarioID, resultado.Usuario.ID)

	response.Sucesso(ctx, resultado)
}

// BuscarPorId retorna os dados de um usuário pelo seu UUID
// @Summary      Buscar usuário por ID
// @Description  Retorna os dados de um usuário ativo a partir do UUID informado na URL
// @Tags         Usuários
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                                true  "UUID do usuário"
// @Success      200  {object}  response.RespostaPadrao{dados=UsuarioRespostaDTO}     "Usuário encontrado"
// @Failure      401  {object}  response.ErroPadrao                                   "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                   "Usuário não encontrado"
// @Failure      500  {object}  response.ErroPadrao                                   "Erro interno do servidor"
// @Router       /usuarios/{id} [get]
func (c *Controller) BuscarPorId(ctx *gin.Context) {
	id := ctx.Param("id")
	solicitanteID := ctx.GetString(middleware.ChaveUsuarioID)

	resultado, err := c.uc.BuscarPorId(id, solicitanteID)
	if err != nil {
		// ErrAcessoNegado e "não encontrado" caem ambos em 404 (não revela existência).
		// Mensagem fixa: o erro de "não encontrado" embrulha o SQL e não pode vazar.
		response.ErroNaoEncontrado(ctx, "usuário não encontrado")
		return
	}

	response.Sucesso(ctx, resultado)
}

// Atualizar modifica os dados de um usuário existente
// @Summary      Atualizar usuário
// @Description  Atualiza parcialmente os dados de um usuário pelo UUID.
//
//	Apenas os campos informados no corpo serão alterados.
//
// @Tags         Usuários
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                                true  "UUID do usuário"
// @Param        body  body      AtualizarUsuarioDTO                                   true  "Campos a atualizar"
// @Success      200   {object}  response.RespostaPadrao{dados=UsuarioRespostaDTO}     "Usuário atualizado"
// @Failure      400   {object}  response.ErroPadrao                                   "Dados inválidos"
// @Failure      401   {object}  response.ErroPadrao                                   "Não autenticado"
// @Failure      404   {object}  response.ErroPadrao                                   "Usuário não encontrado"
// @Failure      409   {object}  response.ErroPadrao                                   "E-mail já em uso"
// @Failure      500   {object}  response.ErroPadrao                                   "Erro interno do servidor"
// @Router       /usuarios/{id} [put]
func (c *Controller) Atualizar(ctx *gin.Context) {
	id := ctx.Param("id")
	solicitanteID := ctx.GetString(middleware.ChaveUsuarioID)

	var dto AtualizarUsuarioDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		response.ErroBind(ctx, err)
		return
	}

	resultado, err := c.uc.Atualizar(id, solicitanteID, dto)
	if err != nil {
		switch {
		case errors.Is(err, ErrAcessoNegado):
			response.ErroNaoEncontrado(ctx, err.Error())
		case err.Error() == "este e-mail já está em uso por outro usuário":
			response.ErroConflito(ctx, err.Error())
		default:
			response.ErroInterno(ctx, err.Error())
		}
		return
	}

	response.Sucesso(ctx, resultado)
}

// UploadFoto faz o upload da foto de perfil do usuário para o S3 e retorna a URL presignada
// @Summary      Upload de foto do usuário
// @Description  Recebe um arquivo de imagem (JPEG, PNG ou WebP, máx. 5MB) via multipart/form-data,
//
//	armazena no S3 de forma privada e retorna o DTO do usuário com a URL presignada temporária.
//
// @Tags         Usuários
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                            true  "UUID do usuário"
// @Param        foto  formData  file                                              true  "Arquivo de imagem"
// @Success      200   {object}  response.RespostaPadrao{dados=UsuarioRespostaDTO} "Foto enviada com sucesso"
// @Failure      400   {object}  response.ErroPadrao                               "Arquivo inválido ou muito grande"
// @Failure      401   {object}  response.ErroPadrao                               "Não autenticado"
// @Failure      404   {object}  response.ErroPadrao                               "Usuário não encontrado"
// @Failure      500   {object}  response.ErroPadrao                               "Erro interno"
// @Router       /usuarios/{id}/foto [post]
func (c *Controller) UploadFoto(ctx *gin.Context) {
	id := ctx.Param("id")
	solicitanteID := ctx.GetString(middleware.ChaveUsuarioID)

	// Limita o tamanho do corpo a 5MB para evitar uploads abusivos
	const maxTamanho = 5 << 20
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxTamanho)

	arquivo, header, err := ctx.Request.FormFile("foto")
	if err != nil {
		// Erro técnico de multipart/tamanho do corpo — não expor ao cliente.
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

	resultado, err := c.uc.UploadFoto(id, solicitanteID, arquivo, header.Size, tipoConteudo)
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

// Deletar realiza a exclusão lógica de um usuário pelo UUID
// @Summary      Deletar usuário
// @Description  Realiza o soft delete de um usuário pelo UUID. O registro permanece no
//
//	banco com deletado_em e deletado_por preenchidos. Não é reversível pela API.
//
// @Tags         Usuários
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                              true  "UUID do usuário"
// @Success      200  {object}  response.RespostaPadrao{dados=string}              "Usuário deletado com sucesso"
// @Failure      401  {object}  response.ErroPadrao                                "Não autenticado"
// @Failure      404  {object}  response.ErroPadrao                                "Usuário não encontrado"
// @Failure      500  {object}  response.ErroPadrao                                "Erro interno do servidor"
// @Router       /usuarios/{id} [delete]
func (c *Controller) Deletar(ctx *gin.Context) {
	id := ctx.Param("id")

	// Recupera o ID do usuário autenticado para registrar quem executou a exclusão
	deletadoPorInterface, _ := ctx.Get(middleware.ChaveUsuarioID)
	deletadoPor, ok := deletadoPorInterface.(string)
	if !ok {
		response.ErroInterno(ctx, "erro ao identificar o usuário autenticado")
		return
	}

	if err := c.uc.Deletar(id, deletadoPor); err != nil {
		// ErrAcessoNegado e "não encontrado" caem ambos em 404 (não revela existência).
		// Mensagem fixa: o erro de "não encontrado" embrulha o SQL e não pode vazar.
		response.ErroNaoEncontrado(ctx, "usuário não encontrado")
		return
	}

	response.Sucesso(ctx, "usuário deletado com sucesso")
}
