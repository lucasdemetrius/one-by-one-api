// Pacote: internal/ia
// Arquivo: provedores.go
// Descrição: Abstração dos provedores de IA (BYOK). Uma única função `completar`
//            recebe o provedor + a chave do gestor e fala com a API certa.
//            DeepSeek e Grok são compatíveis com a API da OpenAI (mesmo formato),
//            então compartilham o mesmo caminho; Claude usa a API de mensagens
//            da Anthropic. Modelos padrão sensatos por provedor (ajustáveis).
// Autor: OneByOne API
// Criado em: 2026

package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Provedores suportados (valores guardados em tb_usuarios.ia_provedor).
const (
	ProvedorClaude   = "CLAUDE"
	ProvedorOpenAI   = "OPENAI"
	ProvedorDeepSeek = "DEEPSEEK"
	ProvedorGrok     = "GROK"
)

// ProvedorValido confere se o identificador de provedor é conhecido.
func ProvedorValido(p string) bool {
	switch p {
	case ProvedorClaude, ProvedorOpenAI, ProvedorDeepSeek, ProvedorGrok:
		return true
	}
	return false
}

// config de cada provedor compatível com a API da OpenAI (baseURL + modelo padrão).
var openAICompat = map[string]struct{ baseURL, modelo string }{
	ProvedorOpenAI:   {"https://api.openai.com/v1", "gpt-4o-mini"},
	ProvedorDeepSeek: {"https://api.deepseek.com/v1", "deepseek-chat"},
	ProvedorGrok:     {"https://api.x.ai/v1", "grok-2-latest"},
}

// Modelo padrão da Anthropic (Claude). Usamos um modelo recente e equilibrado.
const modeloClaude = "claude-sonnet-4-6"

const tempoLimite = 45 * time.Second

// completar envia um prompt ao provedor escolhido e devolve o texto da resposta.
// `sistema` é a instrução de sistema (papel/contexto); `prompt` é o pedido do usuário.
func completar(provedor, chave, sistema, prompt string) (string, error) {
	if provedor == ProvedorClaude {
		return completarClaude(chave, sistema, prompt)
	}
	cfg, ok := openAICompat[provedor]
	if !ok {
		return "", fmt.Errorf("provedor de IA não suportado: %s", provedor)
	}
	return completarOpenAICompat(cfg.baseURL, cfg.modelo, chave, sistema, prompt)
}

// ── OpenAI / DeepSeek / Grok (formato chat/completions) ──────────────────────
func completarOpenAICompat(baseURL, modelo, chave, sistema, prompt string) (string, error) {
	corpo := map[string]any{
		"model": modelo,
		"messages": []map[string]string{
			{"role": "system", "content": sistema},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 1024,
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := postJSON(baseURL+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + chave,
	}, corpo, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("IA: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("a IA não retornou resposta")
	}
	return resp.Choices[0].Message.Content, nil
}

// ── Claude (Anthropic messages) ──────────────────────────────────────────────
func completarClaude(chave, sistema, prompt string) (string, error) {
	corpo := map[string]any{
		"model":      modeloClaude,
		"max_tokens": 1024,
		"system":     sistema,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	var resp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := postJSON("https://api.anthropic.com/v1/messages", map[string]string{
		"x-api-key":         chave,
		"anthropic-version": "2023-06-01",
	}, corpo, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("IA: %s", resp.Error.Message)
	}
	if len(resp.Content) == 0 {
		return "", fmt.Errorf("a IA não retornou resposta")
	}
	return resp.Content[0].Text, nil
}

// postJSON faz um POST com corpo JSON e decodifica a resposta em `destino`.
func postJSON(url string, cabecalhos map[string]string, corpo any, destino any) error {
	dados, err := json.Marshal(corpo)
	if err != nil {
		return err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), tempoLimite)
	defer cancelar()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(dados))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cabecalhos {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao falar com a IA: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(destino); err != nil {
		return fmt.Errorf("resposta inesperada da IA (status %d)", resp.StatusCode)
	}
	return nil
}
