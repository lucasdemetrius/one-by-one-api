// Pacote: internal/admin
// Arquivo: controller.go
// Descrição: Endpoints HTTP do painel de ADMIN da plataforma (/api/v1/admin/...). Todo o
//            grupo é protegido por JWT + middleware ApenasAdmin (só a conta ADMIN entra).
//            São rotas de LEITURA (GET) que alimentam o dashboard de monitoração: contas,
//            acessos (estilo Google Analytics), uso, crescimento e saúde da plataforma.
// Autor: OneByOne API
// Criado em: 2026

package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/response"
)

// Controller expõe os endpoints do painel de admin.
type Controller struct {
	uc UseCase
}

// NovoController cria o Controller de admin.
func NovoController(uc UseCase) *Controller {
	return &Controller{uc: uc}
}

// RegistrarRotas registra as rotas do admin sob /admin, exigindo JWT + papel ADMIN.
func (c *Controller) RegistrarRotas(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	grupo := router.Group("/admin")
	grupo.Use(authMiddleware)
	grupo.Use(middleware.ApenasAdmin())
	{
		grupo.GET("/visao-geral", c.VisaoGeral)
		grupo.GET("/contas", c.ListarContas)
		grupo.GET("/acessos", c.Acessos)
		grupo.GET("/uso", c.Uso)
		grupo.GET("/crescimento", c.Crescimento)
		grupo.GET("/saude", c.SaudePlataforma)
	}
}

// VisaoGeral devolve os cartões de KPI do topo do dashboard.
// @Summary  Visão geral da plataforma (admin)
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=VisaoGeralDTO}  "Resumo executivo"
// @Failure  403  {object}  response.ErroPadrao                           "Acesso restrito ao administrador"
// @Router   /admin/visao-geral [get]
func (c *Controller) VisaoGeral(ctx *gin.Context) {
	res, err := c.uc.VisaoGeral()
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// ListarContas devolve a página de contas com o resumo de uso de cada uma.
// @Summary  Listar contas (admin)
// @Description Lista paginada das contas com resumo de uso. Filtros: papel (RH/LIDER/COLABORADOR/ADMIN) e busca (nome/e-mail).
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Param    papel   query  string  false  "Filtrar por papel"
// @Param    busca   query  string  false  "Buscar em nome/e-mail"
// @Param    limite  query  int     false  "Itens por página (padrão 50, máx 200)"
// @Param    offset  query  int     false  "Deslocamento da paginação"
// @Success  200  {object}  response.RespostaPadrao{dados=ContasPaginaDTO}  "Página de contas"
// @Failure  403  {object}  response.ErroPadrao                             "Acesso restrito ao administrador"
// @Router   /admin/contas [get]
func (c *Controller) ListarContas(ctx *gin.Context) {
	papel := ctx.Query("papel")
	busca := ctx.Query("busca")
	limite := queryInt(ctx, "limite", 0)
	offset := queryInt(ctx, "offset", 0)
	res, err := c.uc.ListarContas(papel, busca, limite, offset)
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// Acessos devolve a série temporal de acessos (gráfico estilo Google Analytics).
// @Summary  Acessos por dia (admin)
// @Description Série diária de logins, usuários ativos e eventos, para o gráfico de linha.
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Param    dias  query  int  false  "Janela em dias (padrão 30, máx 365)"
// @Success  200  {object}  response.RespostaPadrao{dados=SerieAcessosDTO}  "Série de acessos"
// @Failure  403  {object}  response.ErroPadrao                             "Acesso restrito ao administrador"
// @Router   /admin/acessos [get]
func (c *Controller) Acessos(ctx *gin.Context) {
	res, err := c.uc.Acessos(queryInt(ctx, "dias", 0))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// Uso devolve as distribuições de uso (top funcionalidades, hora, dia, papel).
// @Summary  Distribuição de uso (admin)
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Param    dias  query  int  false  "Janela em dias (padrão 30, máx 365)"
// @Success  200  {object}  response.RespostaPadrao{dados=UsoDTO}  "Distribuições de uso"
// @Failure  403  {object}  response.ErroPadrao                    "Acesso restrito ao administrador"
// @Router   /admin/uso [get]
func (c *Controller) Uso(ctx *gin.Context) {
	res, err := c.uc.Uso(queryInt(ctx, "dias", 0))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// Crescimento devolve o crescimento de cadastros e de 1:1 ao longo do período.
// @Summary  Crescimento (admin)
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Param    dias  query  int  false  "Janela em dias (padrão 90, máx 365)"
// @Success  200  {object}  response.RespostaPadrao{dados=CrescimentoDTO}  "Curvas de crescimento"
// @Failure  403  {object}  response.ErroPadrao                            "Acesso restrito ao administrador"
// @Router   /admin/crescimento [get]
func (c *Controller) Crescimento(ctx *gin.Context) {
	res, err := c.uc.Crescimento(queryInt(ctx, "dias", 0))
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// SaudePlataforma devolve os indicadores de engajamento/adoção + top gestores.
// @Summary  Saúde da plataforma (admin)
// @Tags     Admin
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  response.RespostaPadrao{dados=SaudePlataformaDTO}  "Indicadores de saúde"
// @Failure  403  {object}  response.ErroPadrao                                "Acesso restrito ao administrador"
// @Router   /admin/saude [get]
func (c *Controller) SaudePlataforma(ctx *gin.Context) {
	res, err := c.uc.SaudePlataforma()
	if err != nil {
		response.ErroInterno(ctx, err.Error())
		return
	}
	response.Sucesso(ctx, res)
}

// queryInt lê um parâmetro de query como inteiro, com valor padrão se ausente/inválido.
func queryInt(ctx *gin.Context, chave string, padrao int) int {
	if v := ctx.Query(chave); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return padrao
}
