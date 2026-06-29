-- Migration 006 — Classificação 9-box dos liderados
-- Guarda o posicionamento de cada liderado na matriz 9-box:
-- desempenho (eixo X) × potencial (eixo Y), cada um BAIXO/MEDIO/ALTO.

CREATE TABLE IF NOT EXISTS tb_classificacoes (
    colaborador_id VARCHAR(36) NOT NULL PRIMARY KEY,
    desempenho     VARCHAR(10) NOT NULL,            -- BAIXO | MEDIO | ALTO
    potencial      VARCHAR(10) NOT NULL,            -- BAIXO | MEDIO | ALTO
    atualizado_em  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
