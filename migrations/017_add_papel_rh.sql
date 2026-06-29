-- Migration 017: papel RH (Recursos Humanos) — o topo da hierarquia do tenant
-- Descrição: introduz o papel RH no sistema. A hierarquia passa a ser
--            RH (raiz do tenant) → Gestores (LIDER) → Liderados (COLABORADOR).
--            Todas as mudanças aqui são ADITIVAS e retrocompatíveis — contas e
--            comportamentos existentes (gestor solo e liderado) NÃO mudam:
--
--              1) A coluna `role` ganha o valor 'RH' no ENUM. Mantém 'LIDER' e
--                 'COLABORADOR' e o DEFAULT 'COLABORADOR'. Adicionar um valor no
--                 FIM do ENUM é uma alteração de metadados (não reescreve a tabela).
--
--              2) Nova coluna `rh_id`: em um GESTOR, guarda o usuario_id do RH dono
--                 dele — é o vínculo que define "quais gestores um RH enxerga".
--                 NULL significa: gestor "solo" (sem RH acima, igual a hoje) OU o
--                 próprio RH (que é a raiz e não tem RH acima de si).
--                 Sem FOREIGN KEY de propósito: por convenção do projeto, as
--                 migrations 004+ não declaram FK (evita incompatibilidade de
--                 collation do id) — a integridade é garantida pela aplicação.
--
--              3) Índice em `rh_id` para a futura listagem "gestores deste RH"
--                 (consulta WHERE rh_id = ?).
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_usuarios
  MODIFY COLUMN role ENUM('LIDER','COLABORADOR','RH') NOT NULL DEFAULT 'COLABORADOR',
  ADD COLUMN rh_id VARCHAR(36) NULL DEFAULT NULL,
  ADD INDEX idx_usuarios_rh (rh_id);
