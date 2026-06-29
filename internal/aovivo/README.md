# Módulo `aovivo`

> O **1:1 ao vivo** (tempo real via WebSocket). Gestor e liderado entram na mesma
> sala e veem tudo junto: o tabuleiro sincroniza na hora e cada um vê o **cursor**
> do outro (estilo Gartic/Metro Retro). É a "aproximidade" levada ao literal.

## O que faz

Um **Hub** gerencia várias **Salas** (uma por 1:1, identificada pelo `colaborador_id`).
Cada sala mantém os participantes conectados, retransmite cursores e mudanças do
tabuleiro, e guarda o último estado para quem entra no meio.

## Arquivos

| Arquivo | Responsabilidade |
|---|---|
| `sala.go` | `Hub`, `Sala`, `Cliente` — presença, broadcast e estado da sala |
| `handler.go` | Upgrade HTTP→WebSocket, validação do JWT e os pumps de leitura/escrita |

## Rota

| Método | Rota | Descrição |
|---|---|---|
| `GET` (upgrade) | `/api/v1/ws/1a1/{sala}?token=&nome=&papel=` | Conecta à sala de 1:1 (WebSocket) |

> Registrada direto no router **fora** do grupo `/api/v1` para não passar pelo
> middleware de auditoria (que envolve o `ResponseWriter` e quebraria o upgrade).
> O JWT vai por `?token=` porque o navegador não manda `Authorization` no WS.

## Protocolo de mensagens (JSON)

**Cliente → servidor:**
- `{tipo:"cursor", x, y}` — posição do mouse (frações 0..1 do viewport)
- `{tipo:"tabuleiro", tabuleiro:{...}}` — o board mudou
- `{tipo:"tema-atualizado", tema}` — o conteúdo de um tema mudou (bloco add/removido)

**Servidor → cliente:**
- `{tipo:"voce", id}` — id da própria conexão (para não desenhar o próprio cursor)
- `{tipo:"presenca", participantes:[{id,nome,papel,cor}]}` — quem está na sala
- `{tipo:"cursor", de, x, y}` — cursor de outro participante
- `{tipo:"tabuleiro", tabuleiro:{...}}` — board atualizado por alguém (e enviado a quem entra)
- `{tipo:"tema-atualizado", tema}` — avise que o conteúdo de um tema mudou; o outro
  recarrega os blocos daquele tema via REST (o conteúdo não trafega pelo WS, só o aviso)

## Detalhes importantes

- Cada conexão tem um `id` (UUID) e uma cor pela persona (gestor índigo, liderado coral).
- O broadcast exclui o remetente (sem eco) — por isso o cliente aplica a própria
  mudança localmente e recebe só as dos outros.
- Envio não-bloqueante: se o buffer de um cliente lento enche, a mensagem é descartada.
- v1: estado por última-escrita-vence; sem ping/pong (sessões curtas).

## Frontend

`useSalaAoVivo` (hook) conecta, expõe `participantes`/`cursores` e as funções
`enviarCursor`/`enviarTabuleiro`/`enviarTemaAtualizado`. `CursoresAoVivo` desenha os
cursores. Integrado em `PaginaOneByOne` (o `TemaEditor` chama `aoMudarConteudo` ao
salvar/remover um bloco e recarrega ao receber o sinal). O proxy de WebSocket está
no `nginx.conf` e no `vite.config.ts`.
