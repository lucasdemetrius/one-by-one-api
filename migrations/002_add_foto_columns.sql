-- Arquivo: migrations/002_add_foto_columns.sql
-- Descrição: Adiciona a coluna foto_key nas tabelas de usuário, colaborador,
--            equipe e organização para armazenar a chave (path) do objeto
--            no bucket S3. A URL pública nunca é armazenada — apenas a chave
--            interna. A URL presignada é gerada dinamicamente pela API.
-- Autor: OneAOne API
-- Criado em: 2025

ALTER TABLE tb_usuarios
  ADD COLUMN foto_key VARCHAR(500) NULL
  COMMENT 'Chave do objeto no S3 (ex: usuarios/{uuid}/foto.jpg)';

ALTER TABLE tb_colaboradores
  ADD COLUMN foto_key VARCHAR(500) NULL
  COMMENT 'Chave do objeto no S3 (ex: colaboradores/{uuid}/foto.jpg)';

ALTER TABLE tb_equipes
  ADD COLUMN foto_key VARCHAR(500) NULL
  COMMENT 'Chave do objeto no S3 (ex: equipes/{uuid}/foto.jpg)';

ALTER TABLE tb_organizacoes
  ADD COLUMN foto_key VARCHAR(500) NULL
  COMMENT 'Chave do objeto no S3 (ex: organizacoes/{uuid}/foto.jpg)';
