# Módulo `feedback` — Reações dos usuários (curti / não curti / irritado)

Coleta a **pulsação de satisfação** dos usuários em um clique e leva tudo para o
**dashboard de gestão (ADMIN)**. Cada reação é um registro (log append‑only, **não** é
toggle de "like"), com **contexto** e **comentário** opcionais — permite ver a tendência de
humor e ler o feedback qualitativo.

> Módulo **autocontido**: é dono da tabela `tb_feedbacks` (escrita) **e** das suas leituras
> agregadas (o painel). Por isso ele mesmo expõe a rota `/admin/feedbacks` (com `ApenasAdmin`).

---

## Endpoints

| Método | Rota | Quem pode | O que faz |
|---|---|---|---|
| `POST` | `/api/v1/feedback` | **qualquer usuário logado** | Registra uma reação |
| `GET`  | `/api/v1/admin/feedbacks?dias=30` | **só ADMIN** | Painel de feedback do dashboard |

### `POST /feedback`

```jsonc
{
  "reacao": "CURTI",        // obrigatório: CURTI | NAO_CURTI | IRRITADO
  "contexto": "1a1",        // opcional: tela/recurso (ex.: "1a1","pdi","ajuda","dashboard")
  "comentario": "Adorei!",  // opcional: texto livre (até 500)
  "pagina": "/onebyone/123" // opcional: rota/URL
}
```

O `usuario_id` vem **sempre do JWT**, nunca do corpo. Responde `201` com `{ id, reacao,
contexto, criado_em }`. A reação **não** é auditada (tem tabela própria; não infla as
métricas de atividade).

### `GET /admin/feedbacks` (dados prontos para o dashboard)

```jsonc
{
  "periodo": 30,
  "total": 42, "curti": 30, "nao_curti": 8, "irritado": 4,
  "indice_satisfacao": 71.4,          // curti / total (%)
  "dias": ["2026-06-01", "..."],      // eixo X
  "serie_curti": [..], "serie_nao_curti": [..], "serie_irritado": [..],
  "por_contexto": [ { "contexto": "1a1", "curti": 9, "nao_curti": 1, "irritado": 0, "total": 10 } ],
  "recentes": [ { "reacao": "IRRITADO", "contexto": "ajuda", "comentario": "...",
                  "autor_nome": "Maria", "autor_papel": "LIDER", "criado_em": "..." } ]
}
```

As séries vêm **alinhadas por índice** com `dias` (buracos preenchidos com zero). `recentes`
traz só os feedbacks **com comentário** (o qualitativo), já com o autor.

---

## Contrato para o frontend

- **Widget de reação** (em qualquer tela): 3 botões — 👍 `CURTI`, 👎 `NAO_CURTI`,
  😠 `IRRITADO` — e um campo opcional de comentário. Ao clicar, `POST /feedback` com a
  `reacao` e o `contexto` daquela tela (ex.: `"1a1"`, `"pdi"`, `"ajuda"`). Mostre um
  "obrigado!" rápido.
- **Painel no dashboard de admin**: cartões (total + índice de satisfação + irritados),
  **gráfico de linha/área** com as 3 séries, **barras** por contexto, e uma **lista de
  comentários recentes** com autor e reação.

---

## Arquivos

| Arquivo | Papel |
|---|---|
| `controller.go` | `POST /feedback` (qualquer JWT) + `GET /admin/feedbacks` (ApenasAdmin) |
| `usecase.go` | registrar + montar o painel (pivot da série, índice de satisfação) |
| `repository.go` | I/O em `tb_feedbacks` + agregações |
| `entity.go` / `dto.go` / `mapper.go` | entidade, contratos e tradução |

Tabela: `tb_feedbacks` (migration **024**, collation fixada em `utf8mb4_unicode_ci` para o
JOIN com `tb_usuarios` ser sempre seguro).
