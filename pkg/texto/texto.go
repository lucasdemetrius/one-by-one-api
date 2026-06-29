// Pacote: pkg/texto
// Arquivo: texto.go
// Descrição: Pequenos utilitários de normalização de texto compartilhados entre
//            módulos. NormalizarEmail deixa o e-mail em forma canônica (sem espaços
//            nas pontas e tudo minúsculo) para que armazenamento e comparação sejam
//            consistentes — fecha a brecha de "mesmo e-mail em caixas diferentes".
// Autor: OneByOne API
// Criado em: 2026

package texto

import "strings"

// NormalizarEmail devolve o e-mail em forma canônica: TrimSpace + minúsculo.
// Use SEMPRE antes de gravar ou comparar e-mails, para que "Joao@X.com" e
// "joao@x.com " sejam tratados como o mesmo endereço.
func NormalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
