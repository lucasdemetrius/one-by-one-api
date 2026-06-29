# Módulo `pdi`

Gestão de itens de PDI (objetivos/ações) de um liderado com prazos e status de conclusão, registrando quando cada meta foi concluída para desenhar evoluções.

> Documentação gerada a partir do código. O **catálogo geral** (com todos os
> módulos) está na seção 12 do [CLAUDE.md](../../CLAUDE.md). Regras de posse/IDOR
> na seção 7.1. **Ao mudar rota/regra/validação, atualize este README.**

## Endpoints

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/pdi` | Lista todos os itens de PDI de um liderado, ordenados por status (incompletos primeiro), depois por prazo (mais próximos primeiro), depois por criação (mais recentes primeiro). Filtra deletados (soft delete). | LIDER dono (via colaborador.PertenceAoLider) ou próprio liderado (verifica posse no usecase) |
| `POST /colaboradores/:id/pdi` | Cria um novo item de PDI para o liderado (título obrigatório, descrição opcional, prazo opcional em YYYY-MM-DD). Gera UUID, seta concluido=false e criado_em=agora. | LIDER dono (middleware ApenasLider + posse no usecase) |
| `PUT /pdi/:id` | Atualiza campos do item (todos opcionais): título, prazo, status de conclusão. Se marca como concluído, carimba concluido_em=agora; se reabre, limpa concluido_em. Atualiza alterado_em. | LIDER dono (middleware ApenasLider + posse no usecase via item.colaborador_id) |
| `DELETE /pdi/:id` | Remove logicamente o item de PDI (soft delete): seta deletado_em=agora. Retorna 404 se já deletado ou não encontrado. | LIDER dono (middleware ApenasLider + posse no usecase) |

## Tabela(s)

tb_pdi_itens (id UUID, colaborador_id UUID, titulo VARCHAR(255), descricao TEXT NULL, prazo DATE NULL, concluido TINYINT(1) DEFAULT 0, concluido_em DATETIME NULL, criado_em DATETIME, alterado_em DATETIME NULL, deletado_em DATETIME NULL); INDEX idx_pdi_colaborador (colaborador_id); FK: colaborador_id garantida pela aplicação

## Regras de negócio

- Posse: verificada via colaborador.PertenceAoLider(colaboradorID, usuarioID) — garante que o líder JWT é o gestor do liderado (Cadeia B: colaborador → equipe.usuario_id OU organizacao.usuario_id)
- Soft delete: registros não são apagados; preenchem deletado_em. Todas as queries filtram deletado_em IS NULL
- Prazo: parseado como YYYY-MM-DD (time.Local); adicionado 12h para evitar pulo de fuso (meio-dia UTC local)
- Conclusão com carimbo: quando marcado como concluído (concluido=true), concluido_em recebe agora; ao reabrir, limpa. Permite desenhar evolução de burn-up do PDI no tempo
- Recurso alheio: retorna 404 (ErrAcessoNegado mapeado para ErroNaoEncontrado no controller), não 403 (não revela que id existe)
- Descrição opcional: salva como NULL se não informada; nos DTOs, ignorada se string vazia
- Título não atualizável para vazio: PUT com título vazio é ignorado (só atualiza se `dto.Titulo != ""`)

## Validações

- CriarItemPDIDTO.Titulo: required, min=2, max=255 (binding tags)
- CriarItemPDIDTO.Descricao: omitempty (optional)
- CriarItemPDIDTO.Prazo: omitempty, formato YYYY-MM-DD (validado no usecase com parsearPrazo, erro: 'prazo inválido — use AAAA-MM-DD')
- AtualizarItemPDIDTO.Titulo: omitempty, min=2, max=255 quando presente
- AtualizarItemPDIDTO.Prazo: omitempty, formato YYYY-MM-DD quando presente
- AtualizarItemPDIDTO.Concluido: omitempty, *bool (update só se != nil e diferente do valor atual)
- Listar: valida posse do colaborador no usecase; se sem acesso, retorna ErrAcessoNegado → 404
- Criar/Atualizar/Deletar: valida posse do colaborador (ou do item via BuscarPorId → colaborador_id) antes de operação
