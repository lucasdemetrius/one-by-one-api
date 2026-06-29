-- Migration 018: fim da recorrência do 1:1 ("repete até")
-- Descrição: permite limitar uma recorrência de 1:1 — repetir "até a data X" ou
--            "N vezes" (o app converte N em data). NULL = para sempre (comportamento
--            de hoje, retrocompatível). O scheduler para de avançar/lembrar quando a
--            próxima ocorrência passa de repete_ate; o app para de projetar além dela.
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_agendamentos
  ADD COLUMN repete_ate DATE NULL DEFAULT NULL;
