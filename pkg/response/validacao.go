// Pacote: pkg/response
// Arquivo: validacao.go
// Descrição: Traduz os erros de validação do Gin/go-playground (ex.: "Key:
//            'LoginDTO.Email' Error:Field validation for 'Email' failed on the
//            'required' tag") em mensagens AMIGÁVEIS em português, para nunca
//            vazar texto técnico ao usuário.
// Autor: OneByOne API
// Criado em: 2026

package response

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErroBind responde 400 com uma mensagem amigável a partir do erro de bind/validação
// do Gin. Use no lugar de `ErroRequisicao(ctx, "dados inválidos: "+err.Error())`.
func ErroBind(ctx *gin.Context, err error) {
	ErroRequisicao(ctx, MensagemValidacao(err))
}

// MensagemValidacao converte o erro de validação em uma frase clara em português.
// Se não for um erro de validação reconhecido (ex.: JSON malformado), devolve uma
// mensagem genérica e segura.
func MensagemValidacao(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		partes := make([]string, 0, len(ve))
		for _, fe := range ve {
			partes = append(partes, mensagemCampo(fe))
		}
		return strings.Join(partes, " ")
	}
	return "Verifique os dados enviados e tente novamente."
}

// mensagemCampo monta a frase de um campo conforme a regra que falhou. Usa verbos
// neutros de gênero ("Informe...", "... deve ter...") para evitar concordância errada
// (ex.: nunca "a senha é obrigatório").
func mensagemCampo(fe validator.FieldError) string {
	campo := rotuloCampo(fe.Field()) // minúsculo com artigo: "o e-mail", "a senha"
	switch fe.Tag() {
	case "required":
		return "Informe " + campo + "."
	case "email":
		return "Informe um e-mail válido."
	case "min":
		return capitalizar(campo) + " deve ter no mínimo " + fe.Param() + " caracteres."
	case "max":
		return capitalizar(campo) + " deve ter no máximo " + fe.Param() + " caracteres."
	case "len":
		return capitalizar(campo) + " deve ter exatamente " + fe.Param() + " caracteres."
	case "oneof":
		return capitalizar(campo) + " tem um valor inválido."
	default:
		return capitalizar(campo) + " está em formato inválido."
	}
}

// capitalizar deixa a primeira letra maiúscula (para começo de frase).
func capitalizar(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// rotuloCampo traduz o nome do campo da struct (ex.: "Email") em um rótulo amigável,
// minúsculo e com artigo (ex.: "o e-mail"). Sem mapeamento, cai no genérico.
func rotuloCampo(campo string) string {
	rotulos := map[string]string{
		"Email":         "o e-mail",
		"Password":      "a senha",
		"Senha":         "a senha",
		"NovaSenha":     "a nova senha",
		"Nome":          "o nome",
		"Titulo":        "o título",
		"Descricao":     "a descrição",
		"DataHora":      "a data e hora",
		"DataRef":       "a data",
		"ColaboradorID": "o liderado",
		"OrganizacaoID": "a organização",
		"EquipeID":      "a equipe",
		"TemplateID":    "o template",
		"Recorrencia":   "a recorrência",
		"Codigo":        "o código",
		"Token":         "o token",
		"Tipo":          "o tipo",
		"Valor":         "o valor",
		"Empresa":       "a empresa",
		"Provedor":      "o provedor",
		"Chave":         "a chave",
		"Mensagem":      "a mensagem",
		"Desempenho":    "o desempenho",
		"Potencial":     "o potencial",
		"Whatsapp":      "o WhatsApp",
	}
	if r, ok := rotulos[campo]; ok {
		return r
	}
	return "o campo " + strings.ToLower(campo)
}
