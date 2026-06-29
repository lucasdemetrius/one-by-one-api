// Pacote: pkg/email
// Arquivo: email.go
// Descrição: Serviço de envio de e-mail por SMTP (stdlib net/smtp). Se o SMTP não
//            estiver configurado no .env, o envio fica "dormente" (apenas loga) —
//            assim o app funciona sem e-mail até você ter o provedor (ex.: AWS SES).
// Autor: OneByOne API
// Criado em: 2025

package email

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"onebyone-api/pkg/config"
)

// Servico envia e-mails.
type Servico interface {
	// EnviarHTML envia um e-mail HTML. Não bloqueante de erro: loga e segue.
	EnviarHTML(para []string, assunto, corpoHTML string) error
	// Configurado indica se o SMTP está pronto para enviar de verdade.
	Configurado() bool
}

type servicoSMTP struct {
	host      string
	porta     string
	usuario   string
	senha     string
	remetente string
}

// NovoServico cria o serviço de e-mail a partir das configurações.
func NovoServico(cfg *config.Config) Servico {
	return &servicoSMTP{
		host:      cfg.SMTPHost,
		porta:     cfg.SMTPPort,
		usuario:   cfg.SMTPUser,
		senha:     cfg.SMTPPassword,
		remetente: cfg.SMTPRemetente,
	}
}

func (s *servicoSMTP) Configurado() bool {
	return s.host != "" && s.remetente != ""
}

func (s *servicoSMTP) EnviarHTML(para []string, assunto, corpoHTML string) error {
	if !s.Configurado() {
		log.Printf("[email] SMTP não configurado — '%s' para %v NÃO enviado (dormente)", assunto, para)
		return nil
	}

	msg := montarMIME(s.remetente, para, assunto, corpoHTML)
	auth := smtp.PlainAuth("", s.usuario, s.senha, s.host)
	endereco := fmt.Sprintf("%s:%s", s.host, s.porta)

	if err := smtp.SendMail(endereco, auth, s.remetente, para, []byte(msg)); err != nil {
		log.Printf("[email] erro ao enviar '%s' para %v: %v", assunto, para, err)
		return err
	}
	log.Printf("[email] enviado '%s' para %v", assunto, para)
	return nil
}

// montarMIME monta a mensagem MIME de um e-mail HTML.
func montarMIME(de string, para []string, assunto, html string) string {
	var b strings.Builder
	b.WriteString("From: " + de + "\r\n")
	b.WriteString("To: " + strings.Join(para, ", ") + "\r\n")
	b.WriteString("Subject: " + assunto + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(html)
	return b.String()
}
