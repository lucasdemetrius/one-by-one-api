// Pacote: internal/aovivo
// Arquivo: handler.go
// Descrição: Handler HTTP→WebSocket do 1:1 ao vivo. Valida o token JWT (via query
//            ?token=, pois o navegador não envia cabeçalho Authorization no WS),
//            faz o upgrade da conexão e liga o cliente à sala. Também os "pumps"
//            de leitura e escrita da conexão.
// Autor: OneByOne API
// Criado em: 2025

package aovivo

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"

	"onebyone-api/internal/colaborador"
	"onebyone-api/pkg/config"
	"onebyone-api/pkg/middleware"
)

// checadorOrigem decide quais origens podem abrir o WebSocket. Em produção, só a origem
// do app (cfg.AppURL) — evita que outro site conecte o WS no navegador do usuário. Em
// desenvolvimento, aceita qualquer origem (facilita o dev local).
func checadorOrigem(cfg *config.Config) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		if cfg.Ambiente != "producao" {
			return true
		}
		origem := r.Header.Get("Origin")
		return origem == "" || origem == cfg.AppURL
	}
}

// lerPump lê as mensagens do cliente e as repassa à sala. Ao encerrar, remove o
// cliente da sala.
func (c *Cliente) lerPump() {
	defer func() {
		c.sala.sair(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(1 << 16)
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.sala.receber(c, msg)
	}
}

// escreverPump envia ao cliente as mensagens enfileiradas no canal de envio.
func (c *Cliente) escreverPump() {
	defer func() { _ = c.conn.Close() }()
	for msg := range c.envio {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// usuarioDoToken confere a assinatura e a validade do JWT e devolve o usuario_id do dono.
// ok=false para token vazio, inválido, expirado, REVOGADO (versão antiga) ou de conta excluída.
func usuarioDoToken(tokenStr, secret string, db *sqlx.DB) (string, bool) {
	if tokenStr == "" {
		return "", false
	}
	claims := &middleware.ClaimsJWT{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// Rejeita algoritmos não-HMAC (anti troca de algoritmo), igual ao middleware HTTP.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	// Revogação de sessão: o WebSocket TAMBÉM precisa checar a versão do token e se a conta
	// ainda existe — senão um token revogado (senha trocada) ou de conta excluída entraria na
	// sala mesmo já barrado no HTTP. Qualquer erro/divergência → recusa (fail-closed).
	var versaoAtual int
	if err := db.Get(&versaoAtual, "SELECT token_version FROM tb_usuarios WHERE id = ? AND deletado_em IS NULL", claims.UsuarioID); err != nil || versaoAtual != claims.Versao {
		return "", false
	}
	return claims.UsuarioID, true
}

// corPorPapel devolve a cor da marca conforme o papel (gestor/liderado).
func corPorPapel(papel string) string {
	if papel == "gestor" || papel == "LIDER" {
		return "#6366f1" // índigo (gestor)
	}
	return "#fb7185" // coral (liderado)
}

// Handler devolve o handler Gin da rota WebSocket de uma sala de 1:1. Recebe o
// colaboradorUC para autorizar a entrada na sala (posse do board colaborativo).
func Handler(hub *Hub, cfg *config.Config, colaboradorUC colaborador.UseCase, db *sqlx.DB) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checadorOrigem(cfg),
	}
	return func(ctx *gin.Context) {
		usuarioID, ok := usuarioDoToken(ctx.Query("token"), cfg.JWTSecret, db)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"erro": "token inválido"})
			return
		}

		// A sala é identificada pelo colaborador_id. Board colaborativo do 1:1: só o líder
		// dono, o PRÓPRIO liderado OU o RH do tenant podem entrar (PodeAcessar já é
		// RH-aware via PertenceAoLider). Sem isso, qualquer logado entraria em qualquer sala.
		salaID := ctx.Param("sala")
		if permitido, err := colaboradorUC.PodeAcessar(salaID, usuarioID); err != nil || !permitido {
			ctx.JSON(http.StatusForbidden, gin.H{"erro": "acesso negado a esta sala"})
			return
		}

		nome := ctx.Query("nome")
		if nome == "" {
			nome = "Alguém"
		}
		papel := ctx.Query("papel")

		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}

		cliente := &Cliente{
			ID:    uuid.New().String(),
			Nome:  nome,
			Papel: papel,
			Cor:   corPorPapel(papel),
			conn:  conn,
			envio: make(chan []byte, 32),
		}
		sala := hub.obterSala(salaID)
		cliente.sala = sala

		// Avisa o cliente do próprio id (para ele não renderizar o seu cursor).
		voce, _ := json.Marshal(map[string]string{"tipo": "voce", "id": cliente.ID})
		cliente.envio <- voce

		sala.entrar(cliente)

		go cliente.escreverPump()
		cliente.lerPump() // bloqueia até a conexão cair
		hub.removerSeVazia(sala)
	}
}
