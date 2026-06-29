-- Migration 007 — Agendamentos de 1:1 (com recorrência)
-- O gestor agenda um 1:1 com um liderado, com recorrência opcional. Um job diário
-- (scheduler) lembra o gestor por e-mail dos 1:1 de hoje e amanhã.

CREATE TABLE IF NOT EXISTS tb_agendamentos (
    id             VARCHAR(36) NOT NULL PRIMARY KEY,
    usuario_id     VARCHAR(36) NOT NULL,                  -- gestor dono do agendamento
    colaborador_id VARCHAR(36) NOT NULL,                  -- liderado do 1:1
    data_hora      DATETIME    NOT NULL,                  -- próxima ocorrência
    recorrencia    VARCHAR(20) NOT NULL DEFAULT 'NENHUMA',-- NENHUMA|SEMANAL|QUINZENAL|MENSAL
    ativo          TINYINT(1)  NOT NULL DEFAULT 1,
    criado_em      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_agend_usuario (usuario_id),
    INDEX idx_agend_data    (data_hora)
);
