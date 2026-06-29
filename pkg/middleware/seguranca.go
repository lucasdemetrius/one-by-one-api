// Pacote: pkg/middleware
// Arquivo: seguranca.go
// Descrição: Cabeçalhos de segurança em toda resposta da API (defesa em profundidade,
//            independente do Caddy) e um logger que MASCARA o ?token= do WebSocket no
//            log de acesso (o JWT viaja na query string do handshake; sem mascarar, um
//            Bearer válido ficaria em texto puro no log).
// Autor: OneByOne API
// Criado em: 2026

package middleware

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
)

// CabecalhosSeguranca adiciona cabeçalhos de segurança a toda resposta.
func CabecalhosSeguranca() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h := ctx.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff") // não "adivinhar" tipo de arquivo
		h.Set("X-Frame-Options", "SAMEORIGIN")     // anti-clickjacking
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cache-Control", "no-store") // respostas com PII não são cacheadas
		ctx.Next()
	}
}

// LoggerMascarado é o logger de acesso do Gin com o parâmetro token escondido.
func LoggerMascarado() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			p.TimeStamp.Format("2006/01/02 - 15:04:05"),
			p.StatusCode, p.Latency, p.ClientIP, p.Method, mascararToken(p.Path))
	})
}

// reTokenLog captura o valor de qualquer parâmetro token (?token= / &TOKEN= ...) na query:
// case-insensitive, todas as ocorrências, e só quando é parâmetro de fato (precedido por ? ou &)
// — assim não confunde um "token=" que apareça DENTRO do valor de outro parâmetro.
var reTokenLog = regexp.MustCompile(`([?&](?i:token)=)[^&]*`)

// mascararToken troca o valor de TODO parâmetro token por *** na URL que vai pro log.
func mascararToken(caminho string) string {
	return reTokenLog.ReplaceAllString(caminho, "${1}***")
}
