-- Migration 013: data de conclusão dos itens de PDI
-- Descrição: guarda QUANDO cada objetivo de PDI foi concluído, para desenhar a
--            evolução do PDI ao longo do tempo (burn-up). Itens concluídos antes
--            desta coluna ficam com concluido_em NULL (não entram na série).
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_pdi_itens ADD COLUMN concluido_em DATETIME NULL AFTER concluido;
