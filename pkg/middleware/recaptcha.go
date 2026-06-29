// Pacote: pkg/middleware
// Arquivo: recaptcha.go
// Descrição: Verificação do Google reCAPTCHA nas rotas públicas (anti-bot). Liga/desliga
//            pelo .env: se RECAPTCHA_SECRET estiver vazio, o middleware é DORMENTE (deixa
//            passar) — o app funciona normalmente em dev. Quando a chave é preenchida, o
//            token enviado pelo front (cabeçalho X-Recaptcha-Token) é validado com o Google.
// Autor: OneByOne API
// Criado em: 2026

package middleware

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/config"
	"onebyone-api/pkg/response"
)

var clienteHTTP = &http.Client{Timeout: 8 * time.Second}

// VerificarRecaptcha devolve um middleware que valida o reCAPTCHA. Dormente se o segredo
// não estiver configurado (RECAPTCHA_SECRET vazio).
func VerificarRecaptcha(cfg *config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Desligado: deixa passar (dev / sem chave).
		if cfg.RecaptchaSecret == "" {
			ctx.Next()
			return
		}
		token := ctx.GetHeader("X-Recaptcha-Token")
		if token == "" {
			response.Erro(ctx, http.StatusBadRequest, "Confirme que você não é um robô.")
			ctx.Abort()
			return
		}
		if !validarComGoogle(cfg.RecaptchaSecret, token, ctx.ClientIP()) {
			response.Erro(ctx, http.StatusBadRequest, "Verificação anti-robô falhou. Tente novamente.")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// validarComGoogle consulta a API de verificação do reCAPTCHA. Em qualquer erro de rede/
// formato, retorna false (falha segura).
func validarComGoogle(secret, token, ip string) bool {
	resp, err := clienteHTTP.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {secret},
		"response": {token},
		"remoteip": {ip},
	})
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var r struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false
	}
	return r.Success
}
