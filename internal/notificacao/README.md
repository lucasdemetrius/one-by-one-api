# Módulo `notificacao`

Notificações **in-app** (o sino 🔔 do topo) geradas a partir da **agenda de 1:1**,
com **preferências por usuário** (cada um liga/desliga o que quer receber — para a
aplicação "não ser chata"). Vale para **gestor e liderado**.

## Como funciona (o cron)

`scheduler.go` roda em segundo plano **a cada 30 minutos** (e uma vez ao subir).
A cada passada ele varre os agendamentos ativos e, para cada um, calcula a **próxima
ocorrência** (respeitando a recorrência, sem mutar o agendamento) e decide os avisos
por **faixa** — robusto a atraso/reinício:

| Tipo          | Quando dispara                                  |
|---------------|-------------------------------------------------|
| `AGENDA_1DIA` | a ocorrência é **amanhã**                        |
| `AGENDA_HOJE` | a ocorrência é **hoje** e já passou das 6h       |
| `AGENDA_1H`   | a ocorrência está a **≤ 90 min** de distância    |

Cada aviso é **deduplicado** pela coluna `chave` (`usuario|tipo|agendamento|ocorrência`)
via `INSERT IGNORE` — o cron pode rodar várias vezes sem duplicar. Antes de criar,
checa a **preferência** do destinatário (`Pref.Ligado(tipo)`); se desligada, não cria.

> ⚠️ O envio fica isolado no repositório/scheduler. Para escalar (RabbitMQ, e-mail,
> push) basta plugar atrás dessa fronteira — a regra de faixa/dedupe/preferência não muda.

## Endpoints (todos do usuário do token)

| Método | Rota                              | O que faz                          |
|--------|-----------------------------------|------------------------------------|
| GET    | `/notificacoes`                   | Lista as 30 mais recentes          |
| GET    | `/notificacoes/contagem`          | `{ "nao_lidas": n }` (badge)       |
| PUT    | `/notificacoes/itens/:id/lida`    | Marca **uma** como lida            |
| PUT    | `/notificacoes/ler-todas`         | Marca **todas** como lidas         |
| GET    | `/notificacoes/preferencias`      | Lê as preferências                 |
| PUT    | `/notificacoes/preferencias`      | Salva (upsert) as preferências     |

> A rota de marcar-uma fica sob `/itens/:id` de propósito: evita o conflito do Gin
> entre rota **estática** (`/ler-todas`) e **param** (`/:id`) no mesmo nível.

## Tabelas

- **`tb_notificacoes`** — `id, usuario_id, tipo, titulo, mensagem, link, chave (UNIQUE),
  lida, criado_em`. Migration `014`.
- **`tb_pref_notificacoes`** — `usuario_id (PK), agenda_1dia, agenda_hoje, agenda_1h,
  alterado_em`. Sem registro = **tudo ligado** (`PrefPadrao`).

## Dependências

Lê `tb_agendamentos` + `tb_usuarios` (gestor) + `tb_colaboradores` (liderado, conta em
`colaborador.usuario_id`) — mesmos JOINs do scheduler de e-mail da agenda. Não escreve
em outros módulos.

## Posse / segurança

Toda leitura/escrita filtra por `usuario_id = <token>` — cada um só vê e mexe nas
próprias notificações. Não há `:id` de recurso de terceiro exposto.
