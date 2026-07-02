// Pacote: pkg/cripto
// Arquivo: cripto.go
// Descrição: Cifragem simétrica (AES-256-GCM) para guardar segredos sensíveis no
//            banco — hoje, a chave de API de IA do gestor (BYOK). A chave de
//            cifragem é derivada (SHA-256) de um segredo do servidor, então o
//            valor guardado no banco é inútil sem esse segredo.
// Autor: OneByOne API
// Criado em: 2026

package cripto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// derivarChave transforma um segredo de texto em uma chave de 32 bytes (AES-256).
func derivarChave(segredo string) []byte {
	h := sha256.Sum256([]byte(segredo))
	return h[:]
}

// Cifrar criptografa um texto e devolve base64 (nonce + ciphertext).
func Cifrar(textoPuro, segredo string) (string, error) {
	bloco, err := aes.NewCipher(derivarChave(segredo))
	if err != nil {
		return "", fmt.Errorf("erro ao preparar cifra: %w", err)
	}
	gcm, err := cipher.NewGCM(bloco)
	if err != nil {
		return "", fmt.Errorf("erro ao preparar GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("erro ao gerar nonce: %w", err)
	}
	cifrado := gcm.Seal(nonce, nonce, []byte(textoPuro), nil)
	return base64.StdEncoding.EncodeToString(cifrado), nil
}

// DecifrarComFallback tenta decifrar com o segredo primário e, se falhar, com o
// fallback (quando informado). Serve para migrar o segredo de cifragem sem quebrar
// dados já cifrados com o segredo antigo: passa-se o segredo novo como primário e o
// antigo como fallback; o que for regravado passa a usar o novo.
func DecifrarComFallback(base64Cifrado, segredo, fallback string) (string, error) {
	texto, err := Decifrar(base64Cifrado, segredo)
	if err == nil {
		return texto, nil
	}
	if fallback != "" && fallback != segredo {
		if texto2, err2 := Decifrar(base64Cifrado, fallback); err2 == nil {
			return texto2, nil
		}
	}
	return "", err
}

// Decifrar reverte o Cifrar, devolvendo o texto puro original.
func Decifrar(base64Cifrado, segredo string) (string, error) {
	dados, err := base64.StdEncoding.DecodeString(base64Cifrado)
	if err != nil {
		return "", fmt.Errorf("base64 inválido: %w", err)
	}
	bloco, err := aes.NewCipher(derivarChave(segredo))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(bloco)
	if err != nil {
		return "", err
	}
	if len(dados) < gcm.NonceSize() {
		return "", errors.New("dados cifrados inválidos")
	}
	nonce, ct := dados[:gcm.NonceSize()], dados[gcm.NonceSize():]
	textoPuro, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao decifrar: %w", err)
	}
	return string(textoPuro), nil
}
