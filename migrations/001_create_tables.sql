-- Arquivo: migrations/001_create_tables.sql
-- Descrição: Script de criação de todas as tabelas do sistema OneAOne.
--            Utiliza soft delete em todas as entidades (deletado_em, deletado_por).
--            IDs no formato UUID (VARCHAR 36). Datas no formato DATETIME.
-- Autor: OneAOne API
-- Criado em: 2025

-- ─────────────────────────────────────────────
-- Tabela de usuários: armazena líderes e colaboradores com acesso ao sistema
-- ─────────────────────────────────────────────
CREATE TABLE tb_usuarios (
  id            VARCHAR(36)  NOT NULL PRIMARY KEY,
  nome          VARCHAR(100) NOT NULL,
  email         VARCHAR(150) NOT NULL UNIQUE,
  password      VARCHAR(255) NOT NULL,
  role          ENUM('LIDER','COLABORADOR') NOT NULL DEFAULT 'COLABORADOR',
  criado_em     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em   DATETIME     NULL,
  deletado_em   DATETIME     NULL,          -- preenchido no soft delete
  deletado_por  VARCHAR(36)  NULL           -- ID do usuário que realizou o delete
);

-- ─────────────────────────────────────────────
-- Tabela de templates: modelos de formulário para os one-on-ones
-- ─────────────────────────────────────────────
CREATE TABLE tb_template (
  id            VARCHAR(36)  NOT NULL PRIMARY KEY,
  usuario_id    VARCHAR(36)  NOT NULL,      -- líder dono do template
  nome          VARCHAR(100) NOT NULL,
  criado_em     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em   DATETIME     NULL,
  deletado_em   DATETIME     NULL,
  deletado_por  VARCHAR(36)  NULL,
  FOREIGN KEY (usuario_id) REFERENCES tb_usuarios(id)
);

-- ─────────────────────────────────────────────
-- Tabela de blocos de template: cada bloco é um campo do formulário
-- Tipos suportados: TEXT, IMAGE, LIST, HIGHLIGHT
-- ─────────────────────────────────────────────
CREATE TABLE tb_template_blocos (
  id            VARCHAR(36)  NOT NULL PRIMARY KEY,
  template_id   VARCHAR(36)  NOT NULL,
  tipo          ENUM('TEXT','IMAGE','LIST','HIGHLIGHT') NOT NULL,
  posicao       INT          NOT NULL DEFAULT 0,  -- ordem de exibição no formulário
  rotulo        VARCHAR(150) NOT NULL,             -- label exibido ao usuário
  criado_em     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em   DATETIME     NULL,
  deletado_em   DATETIME     NULL,
  deletado_por  VARCHAR(36)  NULL,
  FOREIGN KEY (template_id) REFERENCES tb_template(id)
);

-- ─────────────────────────────────────────────
-- Tabela de organizações: agrupamento de equipes e colaboradores
-- template_id é opcional: se preenchido, é o template padrão da organização
-- ─────────────────────────────────────────────
CREATE TABLE tb_organizacoes (
  id            VARCHAR(36)  NOT NULL PRIMARY KEY,
  usuario_id    VARCHAR(36)  NOT NULL,      -- líder dono da organização
  template_id   VARCHAR(36)  NULL,          -- template padrão (herança de template)
  nome          VARCHAR(100) NOT NULL,
  criado_em     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em   DATETIME     NULL,
  deletado_em   DATETIME     NULL,
  deletado_por  VARCHAR(36)  NULL,
  FOREIGN KEY (usuario_id)   REFERENCES tb_usuarios(id),
  FOREIGN KEY (template_id)  REFERENCES tb_template(id)
);

-- ─────────────────────────────────────────────
-- Tabela de equipes: subgrupos dentro de uma organização
-- template_id é opcional: sobrescreve o template da organização se preenchido
-- ─────────────────────────────────────────────
CREATE TABLE tb_equipes (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  usuario_id      VARCHAR(36)  NOT NULL,    -- líder responsável pela equipe
  organizacao_id  VARCHAR(36)  NOT NULL,
  template_id     VARCHAR(36)  NULL,        -- template padrão (herança de template nível 2)
  nome            VARCHAR(100) NOT NULL,
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  deletado_por    VARCHAR(36)  NULL,
  FOREIGN KEY (usuario_id)     REFERENCES tb_usuarios(id),
  FOREIGN KEY (organizacao_id) REFERENCES tb_organizacoes(id),
  FOREIGN KEY (template_id)    REFERENCES tb_template(id)
);

-- ─────────────────────────────────────────────
-- Tabela de colaboradores: membros de uma equipe
-- usuario_id é nullable pois o colaborador pode não ter conta no sistema
-- template_id é opcional: sobrescreve todos os outros se preenchido (maior prioridade)
-- ─────────────────────────────────────────────
CREATE TABLE tb_colaboradores (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  usuario_id      VARCHAR(36)  NULL,        -- conta no sistema (opcional)
  organizacao_id  VARCHAR(36)  NOT NULL,
  equipe_id       VARCHAR(36)  NOT NULL,
  template_id     VARCHAR(36)  NULL,        -- template exclusivo do colaborador (prioridade máxima)
  nome            VARCHAR(100) NOT NULL,
  email           VARCHAR(150) NOT NULL,
  whatsapp        VARCHAR(20)  NULL,
  data_nascimento DATE         NULL,
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  deletado_por    VARCHAR(36)  NULL,
  FOREIGN KEY (usuario_id)     REFERENCES tb_usuarios(id),
  FOREIGN KEY (organizacao_id) REFERENCES tb_organizacoes(id),
  FOREIGN KEY (equipe_id)      REFERENCES tb_equipes(id),
  FOREIGN KEY (template_id)    REFERENCES tb_template(id)
);

-- ─────────────────────────────────────────────
-- Tabela de one-on-ones: reuniões agendadas entre líder e colaborador
-- Recorrência: NENHUMA (avulso), MENSAL ou QUINZENAL
-- Status: AGENDADO, REALIZADO, PENDENTE
-- ─────────────────────────────────────────────
CREATE TABLE tb_onebyone (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  usuario_id      VARCHAR(36)  NOT NULL,    -- líder que agendou
  organizacao_id  VARCHAR(36)  NOT NULL,
  equipe_id       VARCHAR(36)  NOT NULL,
  colabor_id      VARCHAR(36)  NOT NULL,    -- colaborador da reunião
  recorrencia     ENUM('NENHUMA','MENSAL','QUINZENAL') NOT NULL DEFAULT 'NENHUMA',
  status          ENUM('AGENDADO','REALIZADO','PENDENTE') NOT NULL DEFAULT 'AGENDADO',
  data_agendada   DATE         NOT NULL,
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  deletado_por    VARCHAR(36)  NULL,
  FOREIGN KEY (usuario_id)     REFERENCES tb_usuarios(id),
  FOREIGN KEY (organizacao_id) REFERENCES tb_organizacoes(id),
  FOREIGN KEY (equipe_id)      REFERENCES tb_equipes(id),
  FOREIGN KEY (colabor_id)     REFERENCES tb_colaboradores(id)
);

-- ─────────────────────────────────────────────
-- Tabela de registros de one-on-one: formulário preenchido durante a reunião
-- Vincula um one-on-one ao template usado no momento do preenchimento
-- ─────────────────────────────────────────────
CREATE TABLE tb_registros_onebyone (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  oneaone_id      VARCHAR(36)  NOT NULL,
  template_id     VARCHAR(36)  NOT NULL,    -- snapshot do template usado
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  deletado_por    VARCHAR(36)  NULL,
  FOREIGN KEY (oneaone_id)   REFERENCES tb_onebyone(id),
  FOREIGN KEY (template_id)  REFERENCES tb_template(id)
);

-- ─────────────────────────────────────────────
-- Tabela de valores de registro: respostas de cada bloco do formulário
-- valor_texto: usado para TEXT e HIGHLIGHT
-- valor_json: usado para LIST e IMAGE (estrutura flexível)
-- ─────────────────────────────────────────────
CREATE TABLE tb_valores_registro (
  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
  registro_id     VARCHAR(36)  NOT NULL,
  bloco_id        VARCHAR(36)  NOT NULL,    -- qual bloco do template foi respondido
  valor_texto     TEXT         NULL,        -- conteúdo textual simples
  valor_json      JSON         NULL,        -- conteúdo estruturado (listas, imagens)
  criado_em       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alterado_em     DATETIME     NULL,
  deletado_em     DATETIME     NULL,
  deletado_por    VARCHAR(36)  NULL,
  FOREIGN KEY (registro_id)  REFERENCES tb_registros_onebyone(id),
  FOREIGN KEY (bloco_id)     REFERENCES tb_template_blocos(id)
);
