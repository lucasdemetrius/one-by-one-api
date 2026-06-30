// Pacote: pkg/middleware
// Arquivo: auth.go
// Descrição: Middleware de autenticação JWT que intercepta requisições HTTP,
//            valida o token Bearer no cabeçalho Authorization e injeta os dados
//            do usuário autenticado no contexto do Gin para uso nos controllers.
// Autor: OneByOne API
// Criado em: 2025

package middleware

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/config"
	"onebyone-api/pkg/response"
)

// ChaveUsuarioID é a chave para recuperar o ID do usuário autenticado do contexto Gin
const ChaveUsuarioID = "usuario_id"

// ChaveUsuarioRole é a chave para recuperar a role do usuário autenticado do contexto Gin
const ChaveUsuarioRole = "usuario_role"

// ClaimsJWT representa o payload do token JWT gerado pelo sistema.
// Contém os dados do usuário autenticado além dos campos padrão RFC 7519.
type ClaimsJWT struct {
	// UsuarioID é o UUID do usuário autenticado
	UsuarioID string `json:"usuario_id"`
	// Role é o papel do usuário: LIDER, COLABORADOR, RH ou ADMIN
	Role string `json:"role"`
	// Versao é a versão do token do usuário no momento da emissão (revogação de sessão)
	Versao int `json:"versao"`
	// RegisteredClaims embute os campos padrão JWT (exp, iat, sub, etc.)
	jwt.RegisteredClaims
}

// AutenticarJWT retorna um middleware Gin que valida o token JWT no cabeçalho Authorization.
// Tokens inválidos, expirados ou ausentes resultam em HTTP 401 e interrompem a requisição.
func AutenticarJWT(cfg *config.Config, db *sqlx.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Extrai o valor completo do cabeçalho Authorization
		cabecalho := ctx.GetHeader("Authorization")
		if cabecalho == "" {
			response.ErroNaoAutorizado(ctx, "cabeçalho Authorization ausente")
			ctx.Abort()
			return
		}

		// Valida o formato esperado: "Bearer <token>"
		partes := strings.SplitN(cabecalho, " ", 2)
		if len(partes) != 2 || strings.ToLower(partes[0]) != "bearer" {
			response.ErroNaoAutorizado(ctx, "formato inválido — use: Bearer <token>")
			ctx.Abort()
			return
		}

		tokenString := partes[1]

		// Faz o parse do token validando assinatura e expiração simultaneamente
		claims := &ClaimsJWT{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			// Rejeita tokens que não usem HMAC para evitar ataques de troca de algoritmo
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			response.ErroNaoAutorizado(ctx, "token inválido ou expirado")
			ctx.Abort()
			return
		}

		// Revogação de sessão: o token carrega a "versão" do usuário na emissão. Se a versão
		// no banco mudou (senha trocada) ou a conta não existe mais (excluída), recusa o token
		// mesmo dentro do prazo. Custo: 1 consulta indexada por requisição autenticada.
		var versaoAtual int
		err = db.Get(&versaoAtual, "SELECT token_version FROM tb_usuarios WHERE id = ? AND deletado_em IS NULL", claims.UsuarioID)
		if errors.Is(err, sql.ErrNoRows) {
			response.ErroNaoAutorizado(ctx, "sessão expirada — entre novamente")
			ctx.Abort()
			return
		}
		if err != nil {
			response.ErroInterno(ctx, "falha ao validar a sessão: "+err.Error())
			ctx.Abort()
			return
		}
		if versaoAtual != claims.Versao {
			response.ErroNaoAutorizado(ctx, "sessão expirada — entre novamente")
			ctx.Abort()
			return
		}

		// Injeta os dados do usuário no contexto para acesso nos controllers
		ctx.Set(ChaveUsuarioID, claims.UsuarioID)
		ctx.Set(ChaveUsuarioRole, claims.Role)

		ctx.Next()
	}
}

// ApenasLider é um middleware de autorização que restringe o acesso a usuários com role LIDER.
// Deve ser usado APÓS AutenticarJWT, pois depende dos dados já injetados no contexto.
// Mantido LIDER-only de propósito para rituais do gestor (ex.: encerrar 1:1, que grava o
// usuario_id do dono real). Para rotas de gestão que o RH também exerce, use PermitirGestaoOuRH.
func ApenasLider() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Recupera a role injetada pelo middleware AutenticarJWT
		role, existe := ctx.Get(ChaveUsuarioRole)
		if !existe || role != "LIDER" {
			response.ErroProibido(ctx, "acesso restrito a líderes")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// PermitirGestaoOuRH restringe rotas de GESTÃO a contas LIDER (gestor) ou RH. Barra
// COLABORADOR (liderado) de imediato. É defesa em profundidade — NÃO substitui a checagem
// de posse no UseCase (um gestor/RH ainda precisa provar que o recurso é do seu escopo).
func PermitirGestaoOuRH() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, existe := ctx.Get(ChaveUsuarioRole)
		if !existe || (role != "LIDER" && role != "RH") {
			response.ErroProibido(ctx, "acesso restrito a gestores e RH")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// ApenasRH restringe a rota a contas RH (o topo do tenant). Usado nas rotas exclusivas do
// módulo de RH (cadastrar gestores, dashboards consolidados, timeline da empresa).
func ApenasRH() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, existe := ctx.Get(ChaveUsuarioRole)
		if !existe || role != "RH" {
			response.ErroProibido(ctx, "acesso restrito ao RH")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// ApenasAdmin restringe a rota ao ADMIN da plataforma (super-usuário global de
// monitoração). É o portão de todo o módulo /admin: dashboards de uso, acessos e
// indicadores da plataforma inteira. Como o ADMIN só age em LEITURA agregada (sem
// :id de recurso de outro usuário), o papel no JWT já é prova suficiente — não há
// posse a checar como nos demais módulos.
func ApenasAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, existe := ctx.Get(ChaveUsuarioRole)
		if !existe || role != "ADMIN" {
			response.ErroProibido(ctx, "acesso restrito ao administrador")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
