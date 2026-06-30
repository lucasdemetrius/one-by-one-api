# Módulo `admin` — Painel da plataforma

Painel de **monitoração global** da plataforma, exclusivo da conta **ADMIN**. Enquanto o
RH é o topo de **um** tenant, o ADMIN enxerga a **plataforma inteira** (todas as contas) —
é a visão de observabilidade: quem usa, quanto usa e como a base evolui.

> **Só leitura.** O módulo não escreve nada e **não cria tabela nova**: todas as métricas
> saem de tabelas que já existem (`tb_usuarios`, `tb_auditoria`, `tb_onebyone`,
> `tb_agendamentos`, `tb_organizacoes`, `tb_equipes`, `tb_colaboradores`).

---

## Papel ADMIN e a conta admin

- **Migration `023`** acrescenta `ADMIN` ao ENUM `role` de `tb_usuarios` (aditivo).
- **`middleware.ApenasAdmin()`** protege todo o grupo `/admin` (só `role == ADMIN` entra).
- **Seed no boot** (`admin.GarantirContaAdmin`, chamado no `rotas.go`): garante a conta cujo
  e-mail é `ADMIN_EMAIL` (padrão `admin@admin.com.br`):
  - já existe → **promove** a ADMIN (e zera `rh_id`, pois ADMIN é global);
  - não existe e `ADMIN_SENHA` preenchida → **cria** como ADMIN (bcrypt custo 12);
  - não existe e sem senha → não cria (sem senha padrão, por segurança) e loga o aviso.
  - É **idempotente e defensivo** — se a migration 023 não foi aplicada, só loga e o app sobe.

> Como o ADMIN só faz **leitura agregada** (sem `:id` de recurso de outro usuário), o papel
> no JWT já é prova suficiente — não há posse a checar como nos demais módulos.

---

## Endpoints (todos `GET`, sob `/api/v1/admin`, exigem JWT + papel ADMIN)

| Rota | O que devolve |
|---|---|
| `GET /admin/visao-geral` | Cartões de KPI: contas por papel, estrutura e atividade (DAU/WAU/MAU, logins, 1:1). |
| `GET /admin/contas?papel=&busca=&limite=&offset=` | Lista paginada de contas com **resumo de uso** de cada uma (último acesso, eventos, equipes, liderados, 1:1, gestores). |
| `GET /admin/acessos?dias=30` | **Série temporal** (estilo Google Analytics): logins, usuários ativos e eventos por dia. |
| `GET /admin/uso?dias=30` | Distribuições: top funcionalidades, por hora do dia, por dia da semana, por papel. |
| `GET /admin/crescimento?dias=90` | Novos cadastros por dia/papel + curva acumulada + 1:1 realizados por dia. |
| `GET /admin/saude` | Indicadores de engajamento/adoção + ranking dos gestores mais engajados. |
| `GET /admin/feedbacks` | **Painel de feedback dos usuários** (curti/não curti/irritado) — servido pelo módulo `feedback`. |

Parâmetros: `dias` é limitado a 1–365 (padrão 30, ou 90 no crescimento); `limite` a 1–200
(padrão 50). Valores inválidos caem no padrão.

> `GET /admin/feedbacks` vive sob o caminho do dashboard (e usa o mesmo `ApenasAdmin`), mas
> é implementado no módulo [`feedback`](../feedback/README.md), que é dono da tabela
> `tb_feedbacks` — escrita aberta a qualquer usuário (`POST /feedback`) e leitura só ADMIN.

### De onde sai cada número (fonte de verdade)

- **Acessos / DAU / atividade:** `tb_auditoria` — que registra **logins** (`acao='LOGIN'`) e
  toda **escrita** (POST/PUT/DELETE) com `usuario_id`, `ip` e `criado_em`. Por isso não
  precisamos de uma tabela de acessos nem de um write por requisição.
  - *Observação:* desde esta entrega, o **login passou a ser atribuído ao usuário** (o
    controller seta `ChaveUsuarioID` após autenticar), então DAU e "acessos por usuário"
    são fiéis. Logins **antigos** (antes da mudança) aparecem como `ANONIMO`.
- **Contas / crescimento:** `tb_usuarios` (`role`, `criado_em`, `deletado_em`).
- **1:1 / agenda:** `tb_onebyone` (`status='REALIZADO'`, `realizado_em`) e `tb_agendamentos`.
- **Estrutura / engajamento:** `tb_organizacoes`, `tb_equipes`, `tb_colaboradores`.

> **Fuso horário:** as agregações usam `NOW()`/`CURDATE()` do MySQL — mesma convenção do
> resto do projeto, que assume app e banco no mesmo relógio.

---

## Contrato para o frontend (dashboard estilo Google Analytics)

As séries já vêm **alinhadas por índice** e com os **buracos preenchidos com zero**, prontas
para qualquer biblioteca de gráficos:

```jsonc
// GET /admin/acessos?dias=30  →  dados:
{
  "dias":    ["2026-06-01", "...", "2026-06-30"], // eixo X
  "logins":  [12, 0, 8, ...],                     // linha 1
  "ativos":  [9, 0, 6, ...],                       // linha 2 (usuários distintos/dia)
  "eventos": [40, 3, 22, ...],                     // linha 3
  "periodo": 30, "total_logs": 320
}
```

Sugestão de telas: **cartões** (`/visao-geral`) no topo; **gráfico de linha** (`/acessos`);
**barras** de top funcionalidades + **heatmap** hora×dia (`/uso`); **área/linha** de
crescimento (`/crescimento`); **medidores/percentuais** e **ranking** (`/saude`).

---

## Arquivos

| Arquivo | Papel |
|---|---|
| `controller.go` | 6 rotas GET sob `/admin` (JWT + ApenasAdmin) |
| `usecase.go` | orquestra as consultas, preenche séries, calcula percentuais/médias |
| `repository.go` | SÓ SQL de leitura agregada (parametrizado) |
| `dto.go` | contratos HTTP prontos para gráficos |
| `seed.go` | `GarantirContaAdmin` (boot) |

Sem `entity.go`/`mapper.go` (segue o padrão dos módulos de leitura agregada, como `rh` e
`saude1a1`).
