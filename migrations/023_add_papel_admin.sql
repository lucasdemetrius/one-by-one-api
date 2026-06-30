-- Migration 023: papel ADMIN (administrador da plataforma) — o super-usuário global
-- Descrição: introduz o papel ADMIN, acima dos tenants. Enquanto RH é o topo de UM
--            tenant, o ADMIN enxerga a PLATAFORMA inteira (todas as contas) — é a conta
--            de monitoração/observabilidade (dashboards de uso, acessos, indicadores).
--
--            Mudança ADITIVA e retrocompatível: só acrescenta 'ADMIN' ao FIM do ENUM
--            `role` (alteração de metadados — não reescreve a tabela). Mantém 'LIDER',
--            'COLABORADOR', 'RH' e o DEFAULT 'COLABORADOR'.
--
--            A CONTA admin (e-mail definido em ADMIN_EMAIL, padrão admin@admin.com.br)
--            é garantida no boot pela aplicação (internal/admin.GarantirContaAdmin):
--            se já existir, é promovida a ADMIN; se não existir e ADMIN_SENHA estiver
--            no .env, é criada. Por isso aqui NÃO criamos a conta — só liberamos o papel.
-- Autor: OneByOne API
-- Criado em: 2026

ALTER TABLE tb_usuarios
  MODIFY COLUMN role ENUM('LIDER','COLABORADOR','RH','ADMIN') NOT NULL DEFAULT 'COLABORADOR';
