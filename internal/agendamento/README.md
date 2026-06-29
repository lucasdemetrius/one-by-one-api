# Módulo `agendamento`

> Agenda de 1:1 (com **recorrência**) + um **scheduler** que lembra o gestor por
> e-mail dos 1:1 de hoje e amanhã. A interação e a constância são o coração do
> OneByOne.

## O que faz

O gestor agenda um 1:1 com um liderado, com recorrência opcional (semanal,
quinzenal, mensal). Um job em segundo plano (`scheduler.go`) roda diariamente:
avança as ocorrências recorrentes que já passaram e envia um e-mail de lembrete
para cada gestor com os 1:1 de **hoje e amanhã**.

## Arquivos

| Arquivo | Responsabilidade |
|---|---|
| `entity.go` | `Agendamento` + `AgendamentoContexto` (com nomes/e-mail) |
| `dto.go` | `CriarAgendamentoDTO`, `AgendamentoRespostaDTO` |
| `repository.go` | CRUD + listagem para lembrete + avançar/desativar ocorrência |
| `usecase.go` | Valida liderado, parseia data/hora no fuso local, persiste/lista |
| `controller.go` | Rotas (o gestor vem do JWT) |
| `scheduler.go` | O "cron" de lembretes (roda no boot e a cada 24h) |

## Entidade e tabela

Tabela **`tb_agendamentos`** (migration `007`): `id`, `usuario_id` (gestor),
`colaborador_id` (liderado), `data_hora` (próxima ocorrência), `recorrencia`
(NENHUMA/SEMANAL/QUINZENAL/MENSAL), `ativo`, `criado_em`.

## Endpoints

| Método | Rota | Descrição | Auth |
|---|---|---|---|
| `POST` | `/api/v1/agendamentos` | Agenda um 1:1 (gestor do JWT) | JWT |
| `GET` | `/api/v1/agendamentos` | Lista os 1:1 ativos do gestor | JWT |
| `DELETE` | `/api/v1/agendamentos/{id}` | Cancela um agendamento do gestor | JWT |

## Regras de negócio

- `data_hora` é parseada **no fuso local** (DSN `loc=Local`) — ex.: 14:00 fica 14:00.
- O scheduler **avança** ocorrências passadas pela recorrência (ou desativa as `NENHUMA`),
  agrupa por gestor e envia **um e-mail** com os 1:1 de hoje/amanhã.
- Os e-mails (boas-vindas e lembretes) usam o `pkg/email`, que é **dormente** se o
  SMTP não estiver configurado (apenas loga). Configure `SMTP_*` no `.env` (ex.: AWS SES).

## Dependências

`NovoUseCase(repo, colaboradorUC)` valida o liderado. `NovoScheduler(repo, emailSvc).Iniciar()`
sobe o cron. Montado em `cmd/api/rotas.go` após `colaborador`. Ver também `pkg/email`.
