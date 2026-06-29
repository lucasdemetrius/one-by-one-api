# Módulo `ia`

Registra trilha de atividades dos usuários: automaticamente por middleware em escrita (POST/PUT/DELETE) e sob demanda para eventos de UI do frontend.

> Documentação gerada a partir do código. O **catálogo geral** (com todos os
> módulos) está na seção 12 do [CLAUDE.md](../../CLAUDE.md). Regras de posse/IDOR
> na seção 7.1. **Ao mudar rota/regra/validação, atualize este README.**

## Endpoints

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /auditoria/eventos` | Frontend registra evento de UI (navegação, clique, visualização). Requer JWT. Payload: acao (obrigatório, max 50), entidade (obrigatório, max 100), entidade_id (opcional). Servidor preenche usuario_id, IP e User-Agent; responde 200 com 'evento registrado'. | Qualquer usuário autenticado |
| `GET /auditoria/minha` | Retorna últimos N eventos do usuário autenticado (mais recentes primeiro). Query param: limite (default 50, máximo 200). Responde com lista de AuditoriaRespostaDTO. | Qualquer usuário autenticado |
| `GET /colaboradores/:id/timeline` | Linha do tempo (eventos) de um liderado específico. Checa posse: colaborador deve pertencer ao líder autenticado via PertenceAoLider. Se não pertencer ou erro, responde 404. Responde com lista de eventos (max 100). | LÍDER dono do colaborador |

## Tabela(s)

tb_auditoria: id (VARCHAR 36 PK), usuario_id (VARCHAR 36 NULL), acao (VARCHAR 50), entidade (VARCHAR 100), entidade_id (VARCHAR 36 NULL), ip (VARCHAR 45 NULL), user_agent (VARCHAR 255 NULL), criado_em (DATETIME). Índices: idx_auditoria_usuario, idx_auditoria_entidade (entidade+entidade_id), idx_auditoria_criado.

## Regras de negócio

- Gravação assíncrona em goroutine: Registrar() dispara insert em background, nunca bloqueia resposta HTTP, erros de banco são silenciosos (auditoria nunca quebra operação principal)
- ID gerado na aplicação: uuid.New().String() no momento do registro
- Carimbo de tempo: CriadoEm preenchido com time.Now()
- Normalização: IP e User-Agent vazios convertidos para nil (NULL no banco) via strPtr()
- Limite de paginação seguro: ListarPorUsuario e ListarPorEntidade forçam limite entre 1-200 para padrão 50, evita consultas sem teto
- Middleware global audita automaticamente POST/PUT/DELETE bem-sucedidos (status < 400), mapeia método → CRIAR/ATUALIZAR/DELETAR, derivar entidade e entidade_id do path
- Eventos POST /eventos explicitamente ignorados pelo middleware para evitar auditoria dupla
- Posse: GET /colaboradores/:id/timeline requer colaboradorUseCase.PertenceAoLider(colaboradorID, usuarioID); recurso alheio → 404
- Evento de UI sempre vinculado ao usuário autenticado: RegistrarEvento transforma usuarioID string em ponteiro para reaproveitar Registrar

## Validações

- EventoDTO.acao: required, max=50
- EventoDTO.entidade: required, max=100
- EventoDTO.entidade_id: omitempty (opcional)
- Query param limite em /auditoria/minha: convertido com strconv.Atoi, se inválido assume default 50
- Limite reforçado no UseCase: se <= 0 ou > 200, força para 50
- POST /eventos: usuario_id, IP, User-Agent lidos do contexto HTTP, nunca do JSON
- GET /colaboradores/:id/timeline: colaboradorID obrigatório no path; posse checada antes de listar
