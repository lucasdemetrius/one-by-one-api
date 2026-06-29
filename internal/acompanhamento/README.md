# Módulo `acompanhamento`

Módulo de acompanhamento unificado: registra sentimento do liderado (humor 1-5), entregas, feedbacks recebidos e estudos, cada um com data de referência, título e detalhe opcional.

> Documentação gerada a partir do código. O **catálogo geral** (com todos os
> módulos) está na seção 12 do [CLAUDE.md](../../CLAUDE.md). Regras de posse/IDOR
> na seção 7.1. **Ao mudar rota/regra/validação, atualize este README.**

## Endpoints

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/acompanhamento` | Lista acompanhamentos do liderado, filtrável por tipo (SENTIMENTO\|ENTREGA\|FEEDBACK\|ESTUDO) via ?tipo=. Retorna DTO com id, colaborador_id, tipo, titulo, detalhe, valor, data_ref formatada AAAA-MM-DD, criado_em. | LIDER dono do colaborador (posse verificada via PertenceAoLider) |
| `POST /colaboradores/:id/acompanhamento` | Cria acompanhamento. SENTIMENTO exige valor 1-5 (titulo opcional); demais tipos exigem titulo. Data padrão: hoje (meio-dia em timezone local). Gera UUID. | LIDER dono do colaborador (ApenasLider middleware) |
| `PUT /acompanhamento/:id` | Atualiza campos (todos opcionais): titulo, detalhe, valor (1-5), data_ref. Timestamp alterado_em preenchido automaticamente. | LIDER dono do colaborador (ApenasLider middleware, posse validada no UseCase) |
| `DELETE /acompanhamento/:id` | Remove via soft delete (preenchimento de deletado_em). Checagem de RowsAffected garante recurso existed. | LIDER dono do colaborador (ApenasLider middleware, posse validada no UseCase) |

## Tabela(s)

tb_acompanhamentos: id (PK, UUID), colaborador_id (índice + índice composto com tipo), tipo (SENTIMENTO|ENTREGA|FEEDBACK|ESTUDO), titulo (VARCHAR 255), detalhe (TEXT nullable), valor (INT 1-5 nullable, só SENTIMENTO), data_ref (DATE), criado_em (DATETIME), alterado_em (DATETIME nullable), deletado_em (DATETIME nullable soft delete).

## Regras de negócio

- Posse via Cadeia B (colaborador → equipe.usuario_id OU organizacao.usuario_id): PertenceAoLider validado em todo Criar/Listar/Atualizar/Deletar. Recurso alheio → ErrAcessoNegado → 404.
- Soft delete (deletado_em): todo SELECT filtra 'deletado_em IS NULL'; DeletarSoft preenchimento de timestamp e valida RowsAffected == 1.
- SENTIMENTO (humor): Tipo especial que exige valor 1-5, titulo opcional. Demais tipos (ENTREGA, FEEDBACK, ESTUDO) exigem titulo não-vazio.
- Data de referência (data_ref) parseada em timezone local com time.Parse e offset +12h (meio-dia) para evitar pulo de fuso em exibição.
- Listar ordenado por data_ref DESC, criado_em DESC (acompanhamentos mais recentes primeiro).
- Atualização parcial (PATCH semantics com PUT): campos omitidos no DTO são ignorados; detalhe permite null explícito (pointer *string).

## Validações

- CriarAcompanhamentoDTO: tipo obrigatório + oneof (SENTIMENTO|ENTREGA|FEEDBACK|ESTUDO); titulo max 255 chars (omitempty); detalhe omitempty; valor omitempty + min=1 max=5 se presente; data_ref omitempty (formato AAAA-MM-DD validado no UseCase, erro retorna 'data inválida — use AAAA-MM-DD').
- Regra negócio no Criar: se tipo==SENTIMENTO && valor==nil → 'informe o humor (de 1 a 5)'; else if titulo vazio (após trim) → 'informe um título'.
- AtualizarAcompanhamentoDTO: todos opcionais (omitempty); titulo max 255; detalhe *string nullable (permite null); valor min=1 max=5; data_ref omitempty.
- Data_ref: string AAAA-MM-DD, parseada via time.ParseInLocation. Se vazio, usa time.Now() + 12h. Erro customizado no message.
- Posse: BuscarPorId falha → ErrAcessoNegado (retorna como 404). garantirPosse checa PertenceAoLider(colaboradorID, usuarioID) → booleano ou erro; falso → ErrAcessoNegado.
