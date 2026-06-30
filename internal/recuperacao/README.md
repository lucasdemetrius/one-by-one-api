# Módulo `recuperacao` — Esqueci minha senha

Fluxo público de **redefinição de senha** por link + **código de contra‑senha**.
A pessoa está **deslogada**, então todas as rotas são públicas (sem JWT), mas com
**rate‑limit** e **reCAPTCHA** (quando ligado) para barrar bots e brute‑force.

> Reaproveita o módulo `usuario` (achar a conta pelo e‑mail e trocar a senha com
> validação de complexidade) e o `pkg/email` (e‑mail bonito com o link + o código).

---

## Visão do fluxo

```
1. POST /auth/recuperar-senha        { email }
   → SEMPRE responde "se existir, enviamos" (anti-enumeração)
   → se a conta existir: gera token (UUID) + código de 6 dígitos,
     guarda o hash bcrypt do código, validade de 15 min, e envia o e-mail.

2. (e-mail) Link:  {APP_URL}/redefinir-senha/{token}   + código de 6 dígitos

3. GET  /recuperacoes/{token}        → { "valido": true|false }
   → o front usa para mostrar o formulário ou um aviso "link expirado".

4. POST /recuperacoes/{token}/redefinir   { codigo, nova_senha }
   → valida token (pendente + não expirado) + código (bcrypt) + complexidade
   → troca a senha, marca o token como USADO e invalida as sessões antigas.
```

## Endpoints

| Método | Rota | Corpo | Resposta |
|---|---|---|---|
| `POST` | `/api/v1/auth/recuperar-senha` | `{ "email": "a@b.com" }` | `200 { "mensagem": "Se este e-mail tiver conta, enviamos um link..." }` |
| `GET`  | `/api/v1/recuperacoes/:token` | — | `200 { "dados": { "valido": true } }` |
| `POST` | `/api/v1/recuperacoes/:token/redefinir` | `{ "codigo": "048213", "nova_senha": "NovaSenha1" }` | `200 { "mensagem": "Senha redefinida!..." }` |

Erros de negócio (todos com mensagem amigável):
- **400** link inválido/expirado → "Este link é inválido ou expirou. Peça um novo."
- **400** código incorreto → "Código inválido. Confira o e-mail e tente novamente."
- **400** senha fraca → mensagem de complexidade vinda de `pkg/senha`.

## Regras de segurança (já implementadas)

- **Validade curta: 15 minutos** (`const validadeLink` em `usecase.go`). O texto do
  e‑mail usa esse mesmo número, então nunca fica fora de sincronia.
- **Anti‑enumeração:** `Solicitar` devolve `nil` mesmo se o e‑mail não existir; o
  controller responde sempre a mesma frase. Não dá para descobrir quem tem conta.
- **Código nunca em texto:** guardamos só o **hash bcrypt** do código de 6 dígitos
  (aleatoriedade criptográfica via `crypto/rand`).
- **Uso único:** ao redefinir, o token vira `USADO`. Um novo pedido invalida os
  pendentes anteriores (`InvalidarPendentesDoUsuario`).
- **Anti‑brute‑force do código:** após **5** tentativas erradas (`maxTentativas`), o
  token é invalidado — combinado com a janela de 15 min, fecha o chute do código.
- **Revogação de sessão:** trocar a senha incrementa o `token_version` do usuário
  (via `usuario.RedefinirSenha`), derrubando todos os tokens JWT antigos.
- **Rate‑limit + reCAPTCHA:** as rotas usam o mesmo `limiteAuth` e `recaptchaMW` do
  login/cadastro (ver `rotas.go`).

## E‑mail (visual)

`pkg/email.TemplateRecuperacaoSenha(nome, link, codigo, minutos)` monta o e‑mail com a
identidade da marca (gradiente violeta→coral) e renderiza o código **uma caixa por
dígito** (helper `caixasCodigo`) — o mesmo visual do input "um campo por dígito" da
tela de redefinição. O e‑mail é **dormente** se o SMTP não estiver configurado (só loga).

## Contrato para o frontend (`onebyone-app`)

A tela de redefinição (`/redefinir-senha/:token`) deve:

1. Ao abrir, chamar `GET /recuperacoes/:token`. Se `valido=false`, mostrar "link
   expirado" + botão para pedir outro.
2. Mostrar **6 inputs (um por dígito)** para o código — avançar o foco a cada
   dígito, aceitar colar os 6 de uma vez, e juntar em `codigo` (string de 6 dígitos).
3. Campo de **nova senha** (com as regras de complexidade visíveis) e confirmação.
4. Enviar `POST /recuperacoes/:token/redefinir` com `{ codigo, nova_senha }` e, no
   sucesso, levar ao login.

A tela de "esqueci a senha" (entrada de e‑mail) chama `POST /auth/recuperar-senha` e
mostra sempre a mensagem neutra de retorno (não revela se o e‑mail existe).

## Arquivos

| Arquivo | Papel |
|---|---|
| `controller.go` | 3 rotas públicas (solicitar / validar / redefinir) |
| `usecase.go` | regras (gerar, validar token+código, trocar senha) + `validadeLink` |
| `repository.go` | I/O em `tb_recuperacoes_senha` |
| `entity.go` | espelho da tabela |
| `dto.go` | contratos HTTP |

Tabela: `tb_recuperacoes_senha` (migrations **019** + **020** tentativas).
