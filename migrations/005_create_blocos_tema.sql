-- Migration 005 — Blocos de conteúdo dos temas
-- Cada tema de um liderado (ex.: "Plano de Carreira") pode ter vários blocos de
-- conteúdo: texto, link/curso, imagem (no S3) ou marco com datas. É a
-- "mini-apresentação" de cada tema, por liderado.

CREATE TABLE IF NOT EXISTS tb_blocos_tema (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    colaborador_id VARCHAR(36)  NOT NULL,               -- de quem é o conteúdo
    tema           VARCHAR(150) NOT NULL,               -- título do tema (ex.: "Plano de Carreira")
    tipo           VARCHAR(20)  NOT NULL,               -- TEXTO | LINK | IMAGEM | MARCO
    texto          TEXT         NULL,                   -- texto / legenda / título do link
    url            VARCHAR(500) NULL,                   -- para LINK (cursos, materiais)
    imagem_key     VARCHAR(255) NULL,                   -- chave S3 para IMAGEM
    data_inicio    DATETIME     NULL,                   -- para MARCO
    data_fim       DATETIME     NULL,
    ordem          INT          NOT NULL DEFAULT 0,
    criado_em      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_blocos_colab_tema (colaborador_id, tema)
);
