// Pacote: pkg/senha
// Arquivo: senha.go
// Descrição: Política de complexidade de senha do OneByOne. Centraliza a regra para
//            valer em TODA criação/troca de senha (cadastro, convite, recuperação).
//            Mensagens de erro amigáveis em português.
// Autor: OneByOne API
// Criado em: 2026

package senha

import (
	"fmt"
	"unicode"
)

// MinChars é o tamanho mínimo exigido.
const MinChars = 8

// Validar verifica a complexidade mínima da senha e devolve um erro AMIGÁVEL se for
// fraca. Política: ≥ 8 caracteres, com pelo menos uma letra maiúscula, uma minúscula
// e um número. (Símbolos são bem-vindos, mas não obrigatórios.)
func Validar(s string) error {
	if len([]rune(s)) < MinChars {
		return fmt.Errorf("a senha deve ter no mínimo %d caracteres", MinChars)
	}
	var temMaiuscula, temMinuscula, temNumero bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			temMaiuscula = true
		case unicode.IsLower(r):
			temMinuscula = true
		case unicode.IsDigit(r):
			temNumero = true
		}
	}
	faltam := make([]string, 0, 3)
	if !temMaiuscula {
		faltam = append(faltam, "uma letra maiúscula")
	}
	if !temMinuscula {
		faltam = append(faltam, "uma letra minúscula")
	}
	if !temNumero {
		faltam = append(faltam, "um número")
	}
	if len(faltam) > 0 {
		return fmt.Errorf("a senha precisa de %s", juntar(faltam))
	}
	return nil
}

// juntar lista itens em português ("a", "a e b", "a, b e c").
func juntar(xs []string) string {
	switch len(xs) {
	case 1:
		return xs[0]
	case 2:
		return xs[0] + " e " + xs[1]
	default:
		return xs[0] + ", " + xs[1] + " e " + xs[2]
	}
}
