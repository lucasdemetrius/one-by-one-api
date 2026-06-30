# Módulo `ajuda` — Central de Ajuda com IA

Ajuda para **todos os usuários** (gestor, RH, liderado, admin) conhecerem e tirarem o
máximo do OneByOne. Tem **duas camadas**:

1. **Conteúdo curado (sempre funciona, sem IA):** tópicos (artigos) e um **tour de
   boas-vindas**, filtrados pelo papel de quem está logado.
2. **Assistente de IA (opcional):** responde perguntas livres com conhecimento específico
   do produto. A IA é um **extra** — se não houver chave, o conteúdo curado continua inteiro.

---

## Como a IA resolve a chave (cascata)

```
PLATAFORMA  →  BYOK do usuário  →  indisponível (cai no conteúdo curado)
```

1. **Plataforma** (`IA_PLATAFORMA_PROVEDOR` + `IA_PLATAFORMA_CHAVE` no `.env`): se presente,
   atende **qualquer** usuário — é o jeito de a Ajuda com IA funcionar para liderados também.
2. **BYOK**: senão, usa a IA do próprio gestor/RH (módulo `ia`, `Completar`, que ainda herda
   a do RH). Liderado normalmente não tem BYOK.
3. **Indisponível**: sem nenhuma das duas, devolve uma **mensagem amigável** apontando para
   os tópicos (nunca um erro técnico).

> A chave da plataforma reusa a abstração de provedores do módulo `ia`
> (`ia.CompletarComChave`) — Claude, OpenAI, DeepSeek ou Grok.

---

## Endpoints (sob `/api/v1/ajuda`, exigem JWT — sem restrição de papel)

| Rota | O que faz |
|---|---|
| `GET /ajuda/topicos` | Lista os tópicos visíveis para o papel do usuário. |
| `GET /ajuda/topicos/:id` | Um tópico específico (markdown no campo `conteudo`). |
| `GET /ajuda/tour` | Etapas do tour de boas-vindas adequadas ao papel. |
| `GET /ajuda/ia/status` | `{ ia_disponivel: bool }` — o front mostra ou não o chat. |
| `POST /ajuda/perguntar` | Pergunta livre → resposta da IA (com **rate-limit próprio**). |

`POST /ajuda/perguntar` recebe `{ "pergunta": "..." }` (máx. 1000 chars) e devolve:

```jsonc
{ "resposta": "...", "fonte": "plataforma" | "byok" | "indisponivel", "ia_disponivel": true }
```

- A rota tem um **rate-limit dedicado** (`limiteIA`, 30/min) para proteger o custo da IA.
- Perguntas à IA **não** são auditadas (não alteram estado; `auditoria.go` ignora `perguntar`).

---

## Onde mora o conteúdo (fonte da verdade)

Tudo em **`conteudo.go`**, em português:

- `topicos` — catálogo de artigos (id, título, ícone, categoria, papéis, resumo, conteúdo).
- `tourGestor` / `tourRH` / `tourLiderado` — etapas do tour por papel.
- `baseConhecimento` — a **instrução de sistema** do assistente, com o conhecimento do
  produto (hierarquia, estrutura, pauta, 1:1, PDI, 9-box, agenda, saúde, IA, senha) + o tom.

> Ao mudar uma regra/rota do produto, **atualize o tópico** e o `baseConhecimento`
> correspondentes — é o que mantém a ajuda (curada e por IA) correta.

---

## Contrato para o frontend

- **Central de Ajuda:** liste `GET /ajuda/topicos` em cards (use `icone`, `categoria`,
  `resumo`); ao abrir um card, renderize `conteudo` como **markdown**.
- **Tour de boas-vindas:** consuma `GET /ajuda/tour` (passos com `ordem`, `icone`,
  `titulo`, `texto`) num onboarding em etapas.
- **Chat de IA:** chame `GET /ajuda/ia/status` para decidir se mostra o chat; ao perguntar,
  `POST /ajuda/perguntar`. Se `fonte == "indisponivel"`, mostre a `resposta` (mensagem
  amigável) + um CTA para os tópicos.

## Arquivos

| Arquivo | Papel |
|---|---|
| `controller.go` | rotas sob `/ajuda` (JWT; `perguntar` com rate-limit) |
| `usecase.go` | cascata de IA + serve o conteúdo curado |
| `conteudo.go` | tópicos, tour e a base de conhecimento da IA |
| `dto.go` | contratos HTTP |
