-- Migration 011: Acompanhamento do liderado
-- Descrição: registros de acompanhamento por liderado, num só lugar — SENTIMENTO
--            (humor da semana, valor 1-5), ENTREGA, FEEDBACK (recebido) e ESTUDO.
--            Soft delete (deletado_em) como o resto do projeto.
-- Autor: OneByOne API
-- Criado em: 2026

-- Sem FK explícita (vínculo colaborador_id garantido pela aplicação, como nos
-- demais módulos) para evitar incompatibilidade de collation do id.
CREATE TABLE IF NOT EXISTS tb_acompanhamentos (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  colaborador_id  VARCHAR(36)  NOT NULL,
  tipo            VARCHAR(20)  NOT NULL, -- SENTIMENTO | ENTREGA | FEEDBACK | ESTUDO
  titulo          VARCHAR(255) NOT NULL DEFAULT '',
  detalhe         TEXT         NULL,
  valor           INT          NULL,     -- humor 1-5 no SENTIMENTO; nulo nos demais
  data_ref        DATE         NOT NULL,
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  INDEX idx_acomp_colaborador (colaborador_id),
  INDEX idx_acomp_colaborador_tipo (colaborador_id, tipo)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
