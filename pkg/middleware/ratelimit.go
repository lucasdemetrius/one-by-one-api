// Pacote: pkg/middleware
// Arquivo: ratelimit.go
// Descrição: Rate-limit por IP (token bucket em memória) para frear brute-force, bots e
//            abuso (ex.: e-mail bombing na recuperação de senha). Sem dependência externa
//            nem Redis — adequado a um deploy de instância única. Para o IP ser confiável
//            atrás do Caddy, configure router.SetTrustedProxies(...) no boot.
// Autor: OneByOne API
// Criado em: 2026

package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"onebyone-api/pkg/response"
)

// visitante guarda o "balde" de tokens de um IP.
type visitante struct {
	tokens   float64
	ultimaAt time.Time
}

// LimitadorTaxa devolve um middleware que limita cada IP a `porMinuto` requisições por
// minuto, com pico de `burst`. Excedeu → 429 com mensagem amigável. Os IPs inativos são
// limpos periodicamente para não vazar memória.
func LimitadorTaxa(porMinuto float64, burst float64) gin.HandlerFunc {
	var mu sync.Mutex
	visitantes := map[string]*visitante{}
	taxaPorSeg := porMinuto / 60.0

	// Faxina: remove IPs sem atividade recente.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, v := range visitantes {
				if time.Since(v.ultimaAt) > 10*time.Minute {
					delete(visitantes, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		agora := time.Now()

		mu.Lock()
		v, ok := visitantes[ip]
		if !ok {
			v = &visitante{tokens: burst, ultimaAt: agora}
			visitantes[ip] = v
		} else {
			// Repõe tokens conforme o tempo decorrido (até o teto `burst`).
			v.tokens += agora.Sub(v.ultimaAt).Seconds() * taxaPorSeg
			if v.tokens > burst {
				v.tokens = burst
			}
			v.ultimaAt = agora
		}
		permitido := v.tokens >= 1
		if permitido {
			v.tokens--
		}
		mu.Unlock()

		if !permitido {
			response.Erro(ctx, http.StatusTooManyRequests, "Muitas tentativas em pouco tempo. Aguarde um minuto e tente de novo.")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
