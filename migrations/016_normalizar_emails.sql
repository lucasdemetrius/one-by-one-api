-- Migration 016: normaliza e-mails existentes (minúsculo + sem espaços)
-- Descrição: a aplicação passou a gravar e comparar e-mails em forma canônica
--            (LOWER+TRIM). Esta migration alinha os dados antigos para que o login
--            e as checagens de unicidade continuem casando independentemente da
--            collation da coluna. Seguro sob collation _ci (não há e-mails que
--            difiram apenas por caixa, pois a unicidade já era case-insensitive).
-- Autor: OneByOne API
-- Criado em: 2026

UPDATE tb_usuarios      SET email = LOWER(TRIM(email));
UPDATE tb_colaboradores SET email = LOWER(TRIM(email));
