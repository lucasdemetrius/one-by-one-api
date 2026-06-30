-- Migration 019 — Recuperação de senha
-- Uma recuperação liga um usuário a um link (UUID = token) + um código
-- (contra-senha). A pessoa pede "esqueci a senha", recebe por e-mail o link e o
-- código, abre o link, informa o código e define a nova senha. Token expira e é
-- de uso único. Mesmo padrão dos convites (tb_convites).
-- A validade real (15 minutos) é definida no código (internal/recuperacao,
-- const validadeLink) e gravada na coluna expira_em a cada pedido.

CREATE TABLE IF NOT EXISTS tb_recuperacoes_senha (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,        -- UUID = token do link
    usuario_id  VARCHAR(36)  NOT NULL,                    -- dono da conta
    codigo_hash VARCHAR(255) NOT NULL,                    -- hash bcrypt do código (contra-senha)
    status      VARCHAR(20)  NOT NULL DEFAULT 'PENDENTE', -- PENDENTE | USADO
    expira_em   DATETIME     NOT NULL,                    -- validade (15 minutos, ver const validadeLink)
    criado_em   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    usado_em    DATETIME     NULL,

    INDEX idx_recuperacoes_usuario (usuario_id),
    INDEX idx_recuperacoes_status  (status)
);
