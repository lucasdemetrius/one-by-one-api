-- Migration 010: PDI (Plano de Desenvolvimento Individual)
-- Descrição: itens de PDI por liderado — objetivos/ações com prazo e status de
--            conclusão. Soft delete (deletado_em) como o resto do projeto.
-- Autor: OneByOne API
-- Criado em: 2026

-- Sem FK explícita (o vínculo colaborador_id é garantido pela aplicação, como
-- nos demais módulos) para evitar incompatibilidade de collation do id.
CREATE TABLE IF NOT EXISTS tb_pdi_itens (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  colaborador_id  VARCHAR(36)  NOT NULL,
  titulo          VARCHAR(255) NOT NULL,
  descricao       TEXT         NULL,
  prazo           DATE         NULL,
  concluido       TINYINT(1)   NOT NULL DEFAULT 0,
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  INDEX idx_pdi_colaborador (colaborador_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
