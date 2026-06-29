-- Migration 004 — Convites de liderado
-- Um convite liga um colaborador (liderado) a um link (UUID) + um código
-- (contra-senha). O liderado abre o link, informa o código e cria/usa seu
-- acesso, vinculando sua conta de usuário ao colaborador.

CREATE TABLE IF NOT EXISTS tb_convites (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,   -- UUID = token do link
    colaborador_id VARCHAR(36)  NOT NULL,               -- quem está sendo convidado
    codigo_hash    VARCHAR(255) NOT NULL,               -- hash bcrypt do código (contra-senha)
    status         VARCHAR(20)  NOT NULL DEFAULT 'PENDENTE', -- PENDENTE | ACEITO | CANCELADO
    expira_em      DATETIME     NOT NULL,               -- validade do convite
    criado_em      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aceito_em      DATETIME     NULL,

    INDEX idx_convites_colaborador (colaborador_id),
    INDEX idx_convites_status      (status)
);
