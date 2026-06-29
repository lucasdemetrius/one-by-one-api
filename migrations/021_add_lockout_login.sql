-- Migration 021 — Lockout de conta no login
-- Conta as falhas de login consecutivas por usuário e permite bloquear temporariamente a
-- conta após N falhas, freando brute-force/credential-stuffing mesmo que o atacante troque
-- de IP (o rate-limit por IP é a primeira linha; o lockout é a segunda, por conta).

ALTER TABLE tb_usuarios
    ADD COLUMN tentativas_login INT       NOT NULL DEFAULT 0 AFTER password,
    ADD COLUMN bloqueado_ate    DATETIME  NULL               AFTER tentativas_login;
