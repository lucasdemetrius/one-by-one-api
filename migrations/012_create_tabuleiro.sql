-- Migration 012: Tabuleiro do 1:1 (persistência da pauta)
-- Descrição: guarda o estado do tabuleiro (banco/pauta/conversado + temas) de cada
--            liderado como JSON, para a pauta sobreviver ao recarregar. Um por
--            liderado (colaborador_id é a PK → upsert).
-- Autor: OneByOne API
-- Criado em: 2026

CREATE TABLE IF NOT EXISTS tb_tabuleiros (
  colaborador_id  VARCHAR(36) NOT NULL PRIMARY KEY,
  estado          JSON        NOT NULL,
  criado_em       DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME    NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
