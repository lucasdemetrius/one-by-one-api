-- Migration 024 — Feedback rápido dos usuários (curti / não curti / irritado)
-- Descrição: cada linha é UMA reação de um usuário (log append-only, não toggle), com
--            contexto opcional (a tela/recurso onde reagiu), um comentário livre opcional
--            e a página. Serve para coletar a "pulsação" de satisfação e ler o feedback
--            qualitativo — tudo agregado no painel de ADMIN (GET /admin/feedbacks).
--
--            Collation FIXADA em utf8mb4_unicode_ci de propósito: assim o JOIN com
--            tb_usuarios (que é unicode_ci) é sempre seguro, independentemente da collation
--            padrão do servidor — evita a "divisão de collation" que afeta tabelas criadas
--            só com DEFAULT CHARSET=utf8mb4 (ex.: tb_acompanhamentos).
-- Autor: OneByOne API
-- Criado em: 2026

CREATE TABLE IF NOT EXISTS tb_feedbacks (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,        -- UUID
    usuario_id VARCHAR(36)  NOT NULL,                    -- quem deu o feedback (do JWT)
    reacao     VARCHAR(20)  NOT NULL,                    -- CURTI | NAO_CURTI | IRRITADO
    contexto   VARCHAR(60)  NULL,                        -- tela/recurso (ex.: '1a1','pdi','ajuda')
    comentario VARCHAR(500) NULL,                        -- texto livre opcional
    pagina     VARCHAR(255) NULL,                        -- rota/URL opcional
    criado_em  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_feedbacks_usuario (usuario_id),
    INDEX idx_feedbacks_reacao  (reacao),
    INDEX idx_feedbacks_criado  (criado_em)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
