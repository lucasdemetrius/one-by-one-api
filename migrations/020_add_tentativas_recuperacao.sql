-- Migration 020 — Limite de tentativas no código de recuperação
-- Conta as tentativas de código erradas por token. Ao atingir o limite (ex.: 5), o token
-- é invalidado (status USADO), impedindo brute-force do código de 6 dígitos dentro da
-- janela de validade.

ALTER TABLE tb_recuperacoes_senha
    ADD COLUMN tentativas INT NOT NULL DEFAULT 0 AFTER status;
