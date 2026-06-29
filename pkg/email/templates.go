// Pacote: pkg/email
// Arquivo: templates.go
// Descrição: Templates HTML dos e-mails (boas-vindas e lembrete de 1:1), com a
//            identidade da marca (gradiente violeta→coral). Estilos inline, como
//            exigem os clientes de e-mail.
// Autor: OneByOne API
// Criado em: 2025

package email

import (
	"fmt"
	"strings"
)

// layout envolve o conteúdo no HTML base da marca.
func layout(conteudo string) string {
	return `<!doctype html><html><body style="margin:0;background:#f7f5ff;font-family:Arial,Helvetica,sans-serif;color:#211c33;">
  <table width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:32px 16px;">
    <table width="520" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:24px;overflow:hidden;box-shadow:0 18px 40px -14px rgba(124,92,255,0.25);">
      <tr><td style="background:linear-gradient(135deg,#7c5cff,#fb7185);padding:26px 32px;">
        <span style="color:#ffffff;font-size:22px;font-weight:800;">One<span style="opacity:.85;">by</span>One</span>
      </td></tr>
      <tr><td style="padding:32px;">` + conteudo + `</td></tr>
      <tr><td style="padding:0 32px 28px;color:#6f6886;font-size:12px;">OneByOne — o 1:1 que vocês jogam juntos.</td></tr>
    </table>
  </td></tr></table></body></html>`
}

// primeiroNome devolve o primeiro nome.
func primeiroNome(nome string) string {
	if i := strings.IndexByte(nome, ' '); i > 0 {
		return nome[:i]
	}
	return nome
}

// TemplateBoasVindas devolve (assunto, html) do e-mail de boas-vindas do gestor.
func TemplateBoasVindas(nome, appURL string) (string, string) {
	conteudo := fmt.Sprintf(`
    <h1 style="margin:0 0 10px;font-size:26px;">Bem-vindo, %s! 🎉</h1>
    <p style="font-size:16px;line-height:1.6;color:#4a4560;">Que alegria ter você no OneByOne! A partir de agora você pode montar seu time,
    conduzir 1:1 que importam e acompanhar a evolução de cada liderado — tudo num lugar só.</p>
    <p style="margin:26px 0;"><a href="%s" style="background:linear-gradient(135deg,#7c5cff,#fb7185);color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:12px;font-weight:700;display:inline-block;">Abrir o OneByOne</a></p>
    <p style="font-size:14px;color:#6f6886;">Estamos muito felizes com a sua escolha. 💜</p>
  `, primeiroNome(nome), appURL)
	return "Bem-vindo ao OneByOne 🎉", layout(conteudo)
}

// TemplateConvite devolve (assunto, html) do e-mail de convite enviado ao
// liderado, com o link de aceite e o código (contra-senha).
func TemplateConvite(nomeLiderado, link, codigo string) (string, string) {
	conteudo := fmt.Sprintf(`
    <h1 style="margin:0 0 10px;font-size:26px;">Você foi convidado(a), %s! 👋</h1>
    <p style="font-size:16px;line-height:1.6;color:#4a4560;">Seu gestor te convidou para o <strong>OneByOne</strong> — o espaço dos seus 1:1.
    Crie seu acesso em poucos segundos.</p>
    <p style="margin:26px 0;"><a href="%s" style="background:linear-gradient(135deg,#7c5cff,#fb7185);color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:12px;font-weight:700;display:inline-block;">Aceitar convite</a></p>
    <p style="font-size:15px;color:#4a4560;margin:0 0 6px;">Se o app pedir, seu código de convite é:</p>
    <p style="font-size:30px;letter-spacing:8px;font-weight:800;color:#211c33;background:#f3f0ff;border-radius:12px;padding:14px 0;text-align:center;margin:0 0 18px;">%s</p>
    <p style="font-size:13px;color:#6f6886;">Se você não esperava este convite, pode ignorar este e-mail. O link expira em alguns dias.</p>
  `, primeiroNome(nomeLiderado), link, codigo)
	return "Seu convite para o OneByOne 👋", layout(conteudo)
}

// TemplateRecuperacaoSenha devolve (assunto, html) do e-mail de recuperação de senha,
// com o link de redefinição e o código (contra-senha).
func TemplateRecuperacaoSenha(nome, link, codigo string) (string, string) {
	conteudo := fmt.Sprintf(`
    <h1 style="margin:0 0 10px;font-size:26px;">Vamos redefinir sua senha, %s 🔑</h1>
    <p style="font-size:16px;line-height:1.6;color:#4a4560;">Recebemos um pedido para criar uma nova senha da sua conta no <strong>OneByOne</strong>.
    Clique no botão e, se pedir, informe o código abaixo.</p>
    <p style="margin:26px 0;"><a href="%s" style="background:linear-gradient(135deg,#7c5cff,#fb7185);color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:12px;font-weight:700;display:inline-block;">Redefinir minha senha</a></p>
    <p style="font-size:15px;color:#4a4560;margin:0 0 6px;">Seu código de segurança:</p>
    <p style="font-size:30px;letter-spacing:8px;font-weight:800;color:#211c33;background:#f3f0ff;border-radius:12px;padding:14px 0;text-align:center;margin:0 0 18px;">%s</p>
    <p style="font-size:13px;color:#6f6886;">O link vale por 1 hora e só pode ser usado uma vez. Se você não pediu isso, pode ignorar este e-mail — sua senha continua a mesma.</p>
  `, primeiroNome(nome), link, codigo)
	return "Recuperação de senha — OneByOne 🔑", layout(conteudo)
}

// ItemLembrete é uma linha do e-mail de lembrete (um 1:1 agendado).
type ItemLembrete struct {
	Liderado string
	Quando   string // ex.: "amanhã, 14:00" / "hoje, 09:30"
}

// TemplateLembrete devolve (assunto, html) do e-mail com os 1:1 agendados.
func TemplateLembrete(nomeGestor string, itens []ItemLembrete) (string, string) {
	var linhas strings.Builder
	for _, it := range itens {
		linhas.WriteString(fmt.Sprintf(
			`<tr><td style="padding:12px 0;border-bottom:1px solid #eee;font-weight:700;">%s</td>`+
				`<td style="padding:12px 0;border-bottom:1px solid #eee;text-align:right;color:#6f6886;">%s</td></tr>`,
			it.Liderado, it.Quando))
	}
	conteudo := fmt.Sprintf(`
    <h1 style="margin:0 0 10px;font-size:24px;">Olá, %s! Seus 1:1 chegando 🗓️</h1>
    <p style="font-size:16px;line-height:1.6;color:#4a4560;">Estes são os 1:1 que você tem agendados:</p>
    <table width="100%%" cellpadding="0" cellspacing="0" style="font-size:15px;margin-top:8px;">%s</table>
    <p style="font-size:14px;color:#6f6886;margin-top:22px;">Prepare a pauta e fortaleça a conexão com o time. 💜</p>
  `, primeiroNome(nomeGestor), linhas.String())
	return "Lembrete: seus 1:1 agendados", layout(conteudo)
}

// TemplateAgendaCancelada avisa o liderado de que um 1:1 agendado foi cancelado.
func TemplateAgendaCancelada(nomeLiderado string) (string, string) {
	conteudo := fmt.Sprintf(`
    <h1 style="margin:0 0 10px;font-size:24px;">Olá, %s 👋</h1>
    <p style="font-size:16px;line-height:1.6;color:#4a4560;">Seu gestor <strong>cancelou</strong> um 1:1 que estava agendado com você.
    Em breve um novo encontro pode ser marcado — fique de olho na sua agenda.</p>
    <p style="font-size:14px;color:#6f6886;margin-top:22px;">A conversa continua. 💜</p>
  `, primeiroNome(nomeLiderado))
	return "Seu 1:1 foi cancelado", layout(conteudo)
}

// TemplateAgendaRemarcada avisa o liderado da nova data de um 1:1 remarcado.
func TemplateAgendaRemarcada(nomeLiderado, quando string) (string, string) {
	conteudo := fmt.Sprintf(`
    <h1 style="margin:0 0 10px;font-size:24px;">Olá, %s 👋</h1>
    <p style="font-size:16px;line-height:1.6;color:#4a4560;">Seu gestor <strong>remarcou</strong> um 1:1 com você. A nova data é:</p>
    <p style="font-size:22px;font-weight:800;color:#211c33;background:#f3f0ff;border-radius:12px;padding:14px 18px;margin:14px 0;text-align:center;">📅 %s</p>
    <p style="font-size:14px;color:#6f6886;margin-top:22px;">Te espero lá. 💜</p>
  `, primeiroNome(nomeLiderado), quando)
	return "Seu 1:1 foi remarcado", layout(conteudo)
}
