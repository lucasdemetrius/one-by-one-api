// Pacote: internal/usuario
// Arquivo: google.go
// Descrição: Validação do ID token do "Login com Google" (OAuth). Verifica a
//            assinatura RS256 do token contra as chaves públicas do Google (JWKS),
//            além do emissor (iss), da audiência (aud = GOOGLE_CLIENT_ID) e do
//            prazo (exp). Não adiciona dependência nova: reusa o golang-jwt/v5.
// Autor: OneByOne API
// Criado em: 2026

package usuario

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// URL das chaves públicas do Google (rotacionam de tempos em tempos).
const googleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

// Emissores válidos de um ID token do Google (aceita as duas formas).
var googleEmissores = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

// GoogleClaims são os dados que extraímos do ID token do Google após validá-lo.
type GoogleClaims struct {
	Email         string
	EmailVerified bool
	Nome          string
	Foto          string
}

// jwksCacheGoogle guarda as chaves públicas do Google em memória por um tempo,
// para não bater na rede a cada login. Recarrega quando expira ou quando aparece
// um kid desconhecido (rotação de chave) — com um cooldown entre recargas.
type jwksCacheGoogle struct {
	mu          sync.Mutex
	chaves      map[string]*rsa.PublicKey
	expiraEm    time.Time
	ultimaBusca time.Time
}

var cacheGoogle = &jwksCacheGoogle{}

// obterChave devolve a chave pública do Google para o kid informado (recarregando se preciso).
func (c *jwksCacheGoogle) obterChave(kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cache válido e chave presente → usa direto (caminho normal, sem rede).
	if time.Now().Before(c.expiraEm) {
		if k, ok := c.chaves[kid]; ok {
			return k, nil
		}
	}
	// Recarrega NO MÁXIMO 1x por minuto: sem esse cooldown, tokens forjados com um
	// kid aleatório no header forçariam uma ida ao Google a cada requisição
	// (amplificação/DoS que ainda seguraria o mutex durante o fetch).
	if time.Since(c.ultimaBusca) >= time.Minute {
		c.ultimaBusca = time.Now()
		if err := c.recarregar(); err != nil {
			return nil, err
		}
	}
	if k, ok := c.chaves[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("chave pública do Google não encontrada (kid=%s)", kid)
}

// recarregar baixa o JWKS do Google e converte cada chave para *rsa.PublicKey.
func (c *jwksCacheGoogle) recarregar() error {
	cliente := &http.Client{Timeout: 10 * time.Second}
	resp, err := cliente.Get(googleCertsURL)
	if err != nil {
		return fmt.Errorf("erro ao buscar chaves do Google: %w", err)
	}
	defer resp.Body.Close()

	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("erro ao ler chaves do Google: %w", err)
	}

	novo := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		pk, err := jwkParaRSA(k.N, k.E)
		if err != nil {
			continue
		}
		novo[k.Kid] = pk
	}
	if len(novo) == 0 {
		return fmt.Errorf("nenhuma chave pública válida retornada pelo Google")
	}

	c.chaves = novo
	c.expiraEm = time.Now().Add(1 * time.Hour) // as chaves do Google duram horas
	return nil
}

// jwkParaRSA converte os campos n (módulo) e e (expoente) do JWK (base64url) em *rsa.PublicKey.
func jwkParaRSA(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

// validarIDTokenGoogle valida a assinatura + emissor + audiência + prazo do ID token
// do Google e devolve os dados do usuário. clientID é o GOOGLE_CLIENT_ID esperado.
func validarIDTokenGoogle(idToken, clientID string) (GoogleClaims, error) {
	if clientID == "" {
		return GoogleClaims{}, fmt.Errorf("login com Google não está configurado")
	}

	claims := jwt.MapClaims{}
	// A assinatura RS256 e o exp são validados aqui; aud/iss checamos manualmente
	// abaixo (o Google pode emitir com dois emissores diferentes).
	_, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("token do Google sem kid")
		}
		return cacheGoogle.obterChave(kid)
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return GoogleClaims{}, fmt.Errorf("token do Google inválido: %w", err)
	}

	// Audiência: precisa ser exatamente o nosso Client ID (evita aceitar token de outro app).
	if !audienciaContem(claims["aud"], clientID) {
		return GoogleClaims{}, fmt.Errorf("audiência do token do Google inválida")
	}
	// Emissor: precisa ser o Google.
	if iss, _ := claims["iss"].(string); !googleEmissores[iss] {
		return GoogleClaims{}, fmt.Errorf("emissor do token do Google inválido")
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return GoogleClaims{}, fmt.Errorf("token do Google sem e-mail")
	}
	emailVerificado, _ := claims["email_verified"].(bool)
	nome, _ := claims["name"].(string)
	foto, _ := claims["picture"].(string)

	return GoogleClaims{Email: email, EmailVerified: emailVerificado, Nome: nome, Foto: foto}, nil
}

// audienciaContem verifica se o clientID está na claim "aud" (que pode ser string ou lista).
func audienciaContem(aud interface{}, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == clientID {
				return true
			}
		}
	}
	return false
}
