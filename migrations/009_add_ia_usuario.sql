-- Migration 009: configuração de IA por usuário (BYOK — traga sua própria chave)
-- Descrição: cada gestor escolhe um provedor de IA (CLAUDE/OPENAI/DEEPSEEK/GROK) e
--            guarda a PRÓPRIA chave de API, cifrada (AES-GCM). A chave nunca é
--            retornada pela API nem registrada em log — só o provedor e um
--            indicador de "tem chave".
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_usuarios
  ADD COLUMN ia_provedor VARCHAR(20) NULL DEFAULT NULL,
  ADD COLUMN ia_chave_cifrada TEXT NULL DEFAULT NULL;
