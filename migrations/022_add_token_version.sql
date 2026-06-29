-- Migration 022 — Revogação de JWT (token_version)
-- O JWT é "stateless": uma vez emitido, vale até expirar. Para conseguir INVALIDAR tokens
-- (ex.: a pessoa trocou a senha após um vazamento, ou a conta foi excluída), guardamos uma
-- "versão" por usuário. A versão entra no token na emissão; o middleware compara a versão do
-- token com a do banco a cada requisição. Ao trocar a senha, incrementamos a versão → todos
-- os tokens antigos deixam de valer na hora.

ALTER TABLE tb_usuarios
    ADD COLUMN token_version INT NOT NULL DEFAULT 0 AFTER password;
