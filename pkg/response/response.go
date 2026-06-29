// Pacote: pkg/response
// Arquivo: response.go
// Descrição: Define o envelope padrão de resposta JSON da API e funções
//            auxiliares para enviar respostas de sucesso e erro de forma
//            consistente em todos os endpoints.
// Autor: OneByOne API
// Criado em: 2025

package response

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RespostaPadrao é o envelope JSON utilizado em todas as respostas da API.
// Em sucesso: { "sucesso": true, "dados": {...} }
// Em erro:    { "sucesso": false, "erro": "mensagem" }
type RespostaPadrao struct {
	// Sucesso indica se a operação foi concluída sem erros
	Sucesso bool `json:"sucesso"`
	// Dados contém o payload de sucesso; omitido quando a resposta for de erro
	Dados interface{} `json:"dados,omitempty"`
	// Erro contém a mensagem legível do problema; omitido em respostas de sucesso
	Erro string `json:"erro,omitempty"`
}

// ErroPadrao é a estrutura usada nas anotações Swagger para documentar erros
type ErroPadrao struct {
	// Sucesso sempre será false em respostas de erro
	Sucesso bool `json:"sucesso" example:"false"`
	// Erro contém a mensagem descritiva do problema ocorrido
	Erro string `json:"erro" example:"mensagem de erro legível"`
}

// Sucesso envia ao cliente uma resposta HTTP 200 com o payload informado
func Sucesso(ctx *gin.Context, dados interface{}) {
	ctx.JSON(http.StatusOK, RespostaPadrao{
		Sucesso: true,
		Dados:   dados,
	})
}

// Criado envia ao cliente uma resposta HTTP 201 indicando que o recurso foi criado
func Criado(ctx *gin.Context, dados interface{}) {
	ctx.JSON(http.StatusCreated, RespostaPadrao{
		Sucesso: true,
		Dados:   dados,
	})
}

// Erro envia ao cliente uma resposta de falha com o status HTTP e mensagem fornecidos
func Erro(ctx *gin.Context, status int, mensagem string) {
	ctx.JSON(status, RespostaPadrao{
		Sucesso: false,
		Erro:    mensagem,
	})
}

// ErroInterno responde HTTP 500. O `detalhe` (que pode conter mensagem de driver/SQL) é
// LOGADO no servidor para diagnóstico, mas NUNCA enviado ao cliente — para fora vai sempre
// uma mensagem genérica, evitando vazar a estrutura interna do banco.
func ErroInterno(ctx *gin.Context, detalhe string) {
	log.Printf("[erro interno] %s %s — %s", ctx.Request.Method, ctx.Request.URL.Path, detalhe)
	Erro(ctx, http.StatusInternalServerError, "Ocorreu um erro interno. Tente novamente em instantes.")
}

// ErroNaoEncontrado envia uma resposta HTTP 404 quando o recurso solicitado não existe
func ErroNaoEncontrado(ctx *gin.Context, mensagem string) {
	Erro(ctx, http.StatusNotFound, mensagem)
}

// ErroRequisicao envia uma resposta HTTP 400 para requisições com dados inválidos
func ErroRequisicao(ctx *gin.Context, mensagem string) {
	Erro(ctx, http.StatusBadRequest, mensagem)
}

// ErroNaoAutorizado envia uma resposta HTTP 401 quando o token está ausente ou inválido
func ErroNaoAutorizado(ctx *gin.Context, mensagem string) {
	Erro(ctx, http.StatusUnauthorized, mensagem)
}

// ErroProibido envia uma resposta HTTP 403 quando o usuário não tem permissão para o recurso
func ErroProibido(ctx *gin.Context, mensagem string) {
	Erro(ctx, http.StatusForbidden, mensagem)
}

// ErroConflito envia uma resposta HTTP 409 quando há conflito com dados já existentes
func ErroConflito(ctx *gin.Context, mensagem string) {
	Erro(ctx, http.StatusConflict, mensagem)
}
