-- Migration 014: Notificações in-app + preferências
-- Descrição: avisos in-app (sino) gerados pelo cron a partir da agenda (1 dia antes,
--            no dia de manhã, 1h antes). `chave` é única para deduplicar (um aviso
--            por usuário+tipo+agendamento+ocorrência). Preferências por usuário
--            permitem desligar cada tipo (para não ser uma app chata).
-- Autor: OneByOne API
-- Criado em: 2026

CREATE TABLE IF NOT EXISTS tb_notificacoes (
  id          VARCHAR(36)  NOT NULL PRIMARY KEY,
  usuario_id  VARCHAR(36)  NOT NULL, -- destinatário
  tipo        VARCHAR(30)  NOT NULL,
  titulo      VARCHAR(160) NOT NULL,
  mensagem    VARCHAR(400) NOT NULL,
  link        VARCHAR(200) NULL,
  chave       VARCHAR(200) NOT NULL, -- dedupe: usuario|tipo|agendamento|ocorrencia
  lida        TINYINT(1)   NOT NULL DEFAULT 0,
  criado_em   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_notif_chave (chave),
  INDEX idx_notif_usuario (usuario_id, lida, criado_em)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tb_pref_notificacoes (
  usuario_id  VARCHAR(36) NOT NULL PRIMARY KEY,
  agenda_1dia TINYINT(1)  NOT NULL DEFAULT 1, -- aviso 1 dia antes
  agenda_hoje TINYINT(1)  NOT NULL DEFAULT 1, -- aviso no dia (de manhã)
  agenda_1h   TINYINT(1)  NOT NULL DEFAULT 1, -- aviso ~1h antes
  alterado_em DATETIME    NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
