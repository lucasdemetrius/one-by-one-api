-- Migration 015: marca QUANDO um 1:1 foi realizado
-- Descrição: tb_onebyone passa a ser o "livro-razão" dos 1:1 efetivamente realizados.
--            O ritual de Encerrar 1:1 cria uma linha já com status=REALIZADO e
--            realizado_em=NOW(). Sem essa coluna não dá para datar a realização nem
--            calcular cadência/streak. Índices ajudam o agregado de "Saúde do 1:1".
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_onebyone
  ADD COLUMN realizado_em DATETIME NULL AFTER status;

-- Backfill: reuniões já marcadas como REALIZADO ganham uma data aproximada
UPDATE tb_onebyone
   SET realizado_em = COALESCE(alterado_em, criado_em)
 WHERE status = 'REALIZADO' AND realizado_em IS NULL;

-- Índices para o endpoint agregado de cadência/saúde
CREATE INDEX idx_onebyone_usuario_status_realizado ON tb_onebyone (usuario_id, status, realizado_em);
CREATE INDEX idx_onebyone_colabor_status ON tb_onebyone (colabor_id, status);
