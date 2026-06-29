-- Migration 008: desligamento (inativação) de colaborador
-- Descrição: adiciona a data de desligamento do liderado. Quando preenchida, o
--            colaborador é considerado INATIVO (saiu da empresa / não faz mais
--            parte da equipe), mas o registro é PRESERVADO para a linha do tempo
--            e o histórico. Diferente do soft delete (deletado_em), que é remoção.
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_colaboradores
  ADD COLUMN desligado_em DATETIME NULL DEFAULT NULL;
