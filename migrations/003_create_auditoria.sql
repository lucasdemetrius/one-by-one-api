CREATE TABLE IF NOT EXISTS tb_auditoria (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY,
    usuario_id  VARCHAR(36)  NULL,
    acao        VARCHAR(50)  NOT NULL,
    entidade    VARCHAR(100) NOT NULL,
    entidade_id VARCHAR(36)  NULL,
    ip          VARCHAR(45)  NULL,
    user_agent  VARCHAR(255) NULL,
    criado_em   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_auditoria_usuario  (usuario_id),
    INDEX idx_auditoria_entidade (entidade, entidade_id),
    INDEX idx_auditoria_criado   (criado_em)
);
