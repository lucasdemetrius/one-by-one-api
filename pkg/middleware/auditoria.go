package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// AuditoriaUseCase é a interface mínima que o middleware precisa do módulo de auditoria.
// Evita importação circular sem criar pacote extra.
type AuditoriaUseCase interface {
	Registrar(usuarioID *string, acao, entidade string, entidadeID *string, ip, userAgent string)
}

// RegistrarAuditoria é um middleware Gin que grava automaticamente toda requisição
// que modifica estado (POST, PUT, DELETE) e logins bem-sucedidos.
func RegistrarAuditoria(uc AuditoriaUseCase) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next() // executa o handler primeiro

		metodo := ctx.Request.Method
		// Só audita operações que alteram estado + GET em rotas de auth (login)
		if metodo != "POST" && metodo != "PUT" && metodo != "DELETE" {
			return
		}
		// Ignora erros — não auditamos tentativas inválidas aqui (sem usuário identificado)
		if ctx.Writer.Status() >= 400 {
			return
		}

		acao, entidade := extrairAcaoEntidade(metodo, ctx.FullPath())
		if acao == "" {
			return
		}

		var usuarioID *string
		if uid := ctx.GetString(ChaveUsuarioID); uid != "" {
			usuarioID = &uid
		}

		entidadeID := extrairEntidadeID(ctx)

		uc.Registrar(usuarioID, acao, entidade, entidadeID, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	}
}

// extrairAcaoEntidade mapeia método HTTP + path para acao e entidade legíveis
func extrairAcaoEntidade(metodo, path string) (acao, entidade string) {
	switch metodo {
	case "POST":
		acao = "CRIAR"
	case "PUT":
		acao = "ATUALIZAR"
	case "DELETE":
		acao = "DELETAR"
	}

	// Extrai o nome da entidade a partir do path (ex: /api/v1/organizacoes/:id → organizacao)
	segmentos := strings.Split(strings.Trim(path, "/"), "/")
	for _, seg := range segmentos {
		if seg == "" || seg == "api" || seg == "v1" || strings.HasPrefix(seg, ":") {
			continue
		}
		// Casos especiais
		switch seg {
		case "auth":
			continue
		case "login":
			return "LOGIN", "usuario"
		case "registrar":
			return "CRIAR", "usuario"
		case "foto":
			return "UPLOAD_FOTO", entidade
		case "eventos":
			return "", "" // eventos de UI não precisam de auditoria dupla
		// Sub-recursos do colaborador: entidade mais específica para a linha do tempo.
		case "blocos", "blocos-imagem":
			entidade = "tema_bloco"
		case "convite":
			entidade = "convite"
		case "classificacao":
			entidade = "classificacao"
		case "desligar":
			return "DESLIGAR", "colaborador"
		case "reativar":
			return "REATIVAR", "colaborador"
		case "agendamentos":
			entidade = "agendamento"
		case "template-blocos":
			entidade = "template_bloco"
		case "organizacoes":
			entidade = "organizacao"
		case "equipes":
			entidade = "equipe"
		case "colaboradores":
			entidade = "colaborador"
		case "templates":
			entidade = "template"
		case "usuarios":
			entidade = "usuario"
		case "onebyone":
			entidade = "onebyone"
		case "registros-onebyone":
			entidade = "registro_onebyone"
		case "valores-registro":
			entidade = "valor_registro"
		default:
			if entidade == "" {
				entidade = seg
			}
		}
	}
	return acao, entidade
}

// extrairEntidadeID tenta pegar o :id da rota quando existir
func extrairEntidadeID(ctx *gin.Context) *string {
	if id := ctx.Param("id"); id != "" {
		return &id
	}
	return nil
}
