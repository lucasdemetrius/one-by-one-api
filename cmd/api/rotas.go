// Pacote: cmd/api
// Arquivo: rotas.go
// Descrição: Concentra o "fiação" (injeção de dependências) de todos os módulos
//            e o registro das rotas HTTP. Mantém o main.go enxuto: aqui é onde
//            cada módulo é montado no trio Repository → UseCase → Controller e
//            tem suas rotas registradas no grupo /api/v1.
//
//            Para adicionar um módulo novo, basta acrescentar um bloco aqui
//            (e nada no main.go).
// Autor: OneByOne API
// Criado em: 2025

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"onebyone-api/internal/acompanhamento"
	"onebyone-api/internal/admin"
	"onebyone-api/internal/agendamento"
	"onebyone-api/internal/ajuda"
	"onebyone-api/internal/aovivo"
	"onebyone-api/internal/auditoria"
	"onebyone-api/internal/blocotema"
	"onebyone-api/internal/classificacao"
	"onebyone-api/internal/colaborador"
	"onebyone-api/internal/convite"
	"onebyone-api/internal/equipe"
	"onebyone-api/internal/feedback"
	"onebyone-api/internal/ia"
	"onebyone-api/internal/notificacao"
	"onebyone-api/internal/onebyone"
	"onebyone-api/internal/organizacao"
	"onebyone-api/internal/pdi"
	"onebyone-api/internal/recuperacao"
	"onebyone-api/internal/registroonebyone"
	"onebyone-api/internal/rh"
	"onebyone-api/internal/saude1a1"
	"onebyone-api/internal/tabuleiro"
	"onebyone-api/internal/template"
	"onebyone-api/internal/templatebloco"
	"onebyone-api/internal/usuario"
	"onebyone-api/internal/valorregistro"
	"onebyone-api/pkg/config"
	"onebyone-api/pkg/email"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/storage"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "onebyone-api/docs"
)

// ConfigurarRotas monta o grafo de injeção de dependências de todos os módulos,
// registra os middlewares globais e todas as rotas, e retorna o router Gin
// já pronto para ser iniciado pelo main.go.
//
// Recebe tudo o que os módulos precisam compartilhar:
//   - cfg:   configurações da aplicação (JWT, etc.)
//   - db:    conexão com o banco usada por todos os repositórios
//   - s3Svc: serviço de armazenamento S3 usado pelos módulos que têm foto
func ConfigurarRotas(cfg *config.Config, db *sqlx.DB, s3Svc storage.Armazenamento) *gin.Engine {
	// Serviço de e-mail (SMTP). Dormente se as variáveis SMTP_* não estiverem no .env.
	emailSvc := email.NovoServico(cfg)

	// Garante a conta de ADMIN da plataforma (promove se já existir; cria se ADMIN_SENHA
	// estiver no .env). Idempotente e defensivo — só loga em caso de erro, nunca derruba o boot.
	admin.GarantirContaAdmin(db, cfg.AdminEmail, cfg.AdminSenha)

	// ─── Módulo: auditoria ───────────────────────────────────────────────────────
	// Montado primeiro porque o seu UseCase é usado pelo middleware global de
	// auditoria, que precisa estar registrado antes das demais rotas.
	auditoriaRepo := auditoria.NovoRepositorio(db)
	auditoriaUseCase := auditoria.NovoUseCase(auditoriaRepo)
	auditoriaController := auditoria.NovoController(auditoriaUseCase)

	// ─── Router e middlewares globais ────────────────────────────────────────────
	// Em produção, modo release (sem o logging verboso/avisos de debug do Gin).
	if cfg.Ambiente == "producao" {
		gin.SetMode(gin.ReleaseMode)
	}
	// gin.New (em vez de Default) para usar nosso logger que MASCARA o ?token= do WebSocket.
	router := gin.New()
	router.Use(gin.Recovery(), middleware.LoggerMascarado(), middleware.CabecalhosSeguranca())
	// Confia só nos proxies privados (Docker/Caddy) — assim o ClientIP (rate-limit e
	// auditoria) é o IP REAL do cliente e não pode ser forjado via X-Forwarded-For.
	_ = router.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	api := router.Group("/api/v1")
	authMiddleware := middleware.AutenticarJWT(cfg, db)

	// Rate-limit por IP: teto geral (anti-DoS) + um mais apertado nas rotas públicas de
	// auth/recuperação/convite (os cabeçalhos de segurança já saem no router, cobrindo tudo).
	api.Use(middleware.LimitadorTaxa(300, 120))
	limiteAuth := middleware.LimitadorTaxa(12, 8)
	// Limite do assistente de IA da Ajuda (chamada externa com CUSTO): POR USUÁRIO (não só por
	// IP), para um único usuário não queimar a chave de IA da plataforma trocando de IP.
	// (Defesa de instância única; para um teto de orçamento global, configure também a cota no
	// provedor de IA e/ou restrinja a IA de plataforma a papéis pagantes.)
	limiteIA := middleware.LimitadorTaxaPorUsuario(30, 10)
	// Limite da gravação de feedback (append-only): por usuário, evita inflar a tabela.
	limiteFeedback := middleware.LimitadorTaxaPorUsuario(20, 10)
	recaptchaMW := middleware.VerificarRecaptcha(cfg)

	// Middleware de auditoria aplicado globalmente — grava toda operação de escrita
	api.Use(middleware.RegistrarAuditoria(auditoriaUseCase))

	// Healthcheck — verificação simples de que a API está no ar
	api.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok", "versao": "1.0"})
	})

	// Config pública para o front (ex.: site key do reCAPTCHA e Client ID do Google, se
	// ligados no .env). O front lê em runtime — não precisa rebuildar quando muda no .env.
	api.GET("/config", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"sucesso": true, "dados": gin.H{
			"recaptcha_habilitado": cfg.RecaptchaSecret != "",
			"recaptcha_site_key":   cfg.RecaptchaSiteKey,
			"google_habilitado":    cfg.GoogleClientID != "",
			"google_client_id":     cfg.GoogleClientID,
		}})
	})

	// Swagger UI — documentação interativa em /swagger/index.html
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ─── Módulo: usuario ─────────────────────────────────────────────────────────
	usuarioRepo := usuario.NovoRepositorio(db)
	usuarioUseCase := usuario.NovoUseCase(usuarioRepo, cfg, s3Svc, emailSvc)
	usuarioController := usuario.NovoController(usuarioUseCase)
	usuarioController.RegistrarRotas(api, authMiddleware, limiteAuth, recaptchaMW)

	// ─── Módulo: template ────────────────────────────────────────────────────────
	templateRepo := template.NovoRepositorio(db)
	templateUseCase := template.NovoUseCase(templateRepo)
	templateController := template.NovoController(templateUseCase)
	templateController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: templatebloco ───────────────────────────────────────────────────
	// Depende do templateUseCase para checar a posse (bloco → template → usuario_id).
	// Montado DEPOIS do módulo template.
	templateBlocoRepo := templatebloco.NovoRepositorio(db)
	templateBlocoUseCase := templatebloco.NovoUseCase(templateBlocoRepo, templateUseCase)
	templateBlocoController := templatebloco.NovoController(templateBlocoUseCase)
	templateBlocoController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: organizacao ─────────────────────────────────────────────────────
	organizacaoRepo := organizacao.NovoRepositorio(db)
	organizacaoUseCase := organizacao.NovoUseCase(organizacaoRepo, s3Svc)
	organizacaoController := organizacao.NovoController(organizacaoUseCase)
	organizacaoController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: equipe ──────────────────────────────────────────────────────────
	equipeRepo := equipe.NovoRepositorio(db)
	equipeUseCase := equipe.NovoUseCase(equipeRepo, s3Svc)
	equipeController := equipe.NovoController(equipeUseCase)
	equipeController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: colaborador ─────────────────────────────────────────────────────
	colaboradorRepo := colaborador.NovoRepositorio(db)
	colaboradorUseCase := colaborador.NovoUseCase(colaboradorRepo, s3Svc)
	colaboradorController := colaborador.NovoController(colaboradorUseCase)
	// A linha do tempo (auditoria) usa a posse do colaborador para autorizar.
	auditoriaController.ComPosseColaborador(colaboradorUseCase)
	colaboradorController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: convite ─────────────────────────────────────────────────────────
	// Convite de liderado. Reaproveita usuarioUseCase (criar conta + login/JWT) e
	// colaboradorUseCase (vincular a conta ao colaborador).
	conviteRepo := convite.NovoRepositorio(db)
	conviteUseCase := convite.NovoUseCase(conviteRepo, usuarioUseCase, colaboradorUseCase, emailSvc, cfg.AppURL)
	conviteController := convite.NovoController(conviteUseCase)
	conviteController.RegistrarRotas(api, authMiddleware, limiteAuth, recaptchaMW)

	// Recuperação de senha ("esqueci minha senha") — ROTAS PÚBLICAS (a pessoa está deslogada).
	// Reaproveita o usuarioUseCase (buscar por e-mail + trocar a senha) e o e-mail (link + código).
	recuperacaoRepo := recuperacao.NovoRepositorio(db)
	recuperacaoUseCase := recuperacao.NovoUseCase(recuperacaoRepo, usuarioUseCase, emailSvc, cfg.AppURL)
	recuperacaoController := recuperacao.NovoController(recuperacaoUseCase)
	recuperacaoController.RegistrarRotas(api, limiteAuth, recaptchaMW)

	// ─── Módulo: blocotema ───────────────────────────────────────────────────────
	// Conteúdo rico dos temas (texto/link/imagem/marco), por liderado. Usa o S3 e
	// valida o colaborador via colaboradorUseCase.
	blocoTemaRepo := blocotema.NovoRepositorio(db)
	blocoTemaUseCase := blocotema.NovoUseCase(blocoTemaRepo, s3Svc, colaboradorUseCase)
	blocoTemaController := blocotema.NovoController(blocoTemaUseCase)
	blocoTemaController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: classificacao ───────────────────────────────────────────────────
	// Matriz 9-box (desempenho × potencial) dos liderados. Valida via colaboradorUseCase.
	classificacaoRepo := classificacao.NovoRepositorio(db)
	classificacaoUseCase := classificacao.NovoUseCase(classificacaoRepo, colaboradorUseCase)
	classificacaoController := classificacao.NovoController(classificacaoUseCase)
	classificacaoController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: agendamento ─────────────────────────────────────────────────────
	// Agenda de 1:1 (com recorrência) + scheduler que lembra o gestor por e-mail.
	agendamentoRepo := agendamento.NovoRepositorio(db)
	agendamentoUseCase := agendamento.NovoUseCase(agendamentoRepo, colaboradorUseCase, emailSvc)
	agendamentoController := agendamento.NovoController(agendamentoUseCase)
	agendamentoController.RegistrarRotas(api, authMiddleware)
	agendamento.NovoScheduler(agendamentoRepo, emailSvc).Iniciar()

	// ─── Módulo: notificacao (sino in-app + preferências) ────────────────────────
	// Cron a cada 30 min gera avisos da agenda (1d/hoje/1h) por faixa+dedupe,
	// respeitando as preferências de cada usuário (gestor e liderado).
	notificacaoRepo := notificacao.NovoRepositorio(db)
	notificacaoUseCase := notificacao.NovoUseCase(notificacaoRepo)
	notificacaoController := notificacao.NovoController(notificacaoUseCase)
	notificacaoController.RegistrarRotas(api, authMiddleware)
	notificacao.NovoScheduler(notificacaoRepo).Iniciar()

	// ─── Módulo: onebyone ────────────────────────────────────────────────────────
	// Recebe o colaboradorUseCase para o "Encerrar 1:1" resolver posse (Cadeia B) e a
	// estrutura (organização/equipe) do liderado. Montado depois do colaborador.
	onebyoneRepo := onebyone.NovoRepositorio(db)
	onebyoneUseCase := onebyone.NovoUseCase(onebyoneRepo, colaboradorUseCase)
	onebyoneController := onebyone.NovoController(onebyoneUseCase)
	onebyoneController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: saude1a1 (Saúde do 1:1 + Streak no /painel) ─────────────────────
	// Leitura agregada de cadência: % em dia, atrasados, realizados (30d) e streak.
	// Lê tb_onebyone (realizados) + tb_agendamentos (esperados) do próprio gestor.
	saude1a1Repo := saude1a1.NovoRepositorio(db)
	saude1a1UseCase := saude1a1.NovoUseCase(saude1a1Repo)
	saude1a1Controller := saude1a1.NovoController(saude1a1UseCase)
	saude1a1Controller.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: rh (Recursos Humanos — topo do tenant) ──────────────────────────
	// Exclusivo de contas RH (middleware ApenasRH). Cadastra gestores (vínculo rh_id
	// derivado do JWT) e dá visão consolidada: lista de gestores com KPIs (reusa
	// saude1a1) e drill-down nos 1:1 e agendas de cada gestor do tenant. Montado DEPOIS
	// de usuario, saude1a1, onebyone e agendamento (suas dependências).
	rhRepo := rh.NovoRepositorio(db)
	rhUseCase := rh.NovoUseCase(rhRepo, usuarioUseCase, organizacaoUseCase, saude1a1UseCase, onebyoneUseCase, agendamentoUseCase)
	rhController := rh.NovoController(rhUseCase)
	rhController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: registroonebyone ────────────────────────────────────────────────
	// Depende do onebyoneUseCase para resolver o template pela regra de herança:
	// colaborador → equipe → organização → padrão do líder. Por isso é montado
	// DEPOIS do módulo onebyone.
	registroRepo := registroonebyone.NovoRepositorio(db)
	registroUseCase := registroonebyone.NovoUseCase(registroRepo, onebyoneUseCase)
	registroController := registroonebyone.NovoController(registroUseCase)
	registroController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: valorregistro ───────────────────────────────────────────────────
	// Depende do registroUseCase para resolver a posse (valor → registro → reunião
	// → usuario_id). Montado DEPOIS do módulo registroonebyone.
	valorRepo := valorregistro.NovoRepositorio(db)
	valorUseCase := valorregistro.NovoUseCase(valorRepo, registroUseCase)
	valorController := valorregistro.NovoController(valorUseCase)
	valorController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: pdi (Plano de Desenvolvimento Individual) ───────────────────────
	// Posse via colaboradorUseCase (PertenceAoLider). Montado depois do colaborador.
	pdiRepo := pdi.NovoRepositorio(db)
	pdiUseCase := pdi.NovoUseCase(pdiRepo, colaboradorUseCase)
	pdiController := pdi.NovoController(pdiUseCase)
	pdiController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: acompanhamento (sentimento, entregas, feedbacks, estudos) ───────
	// Tudo do liderado num lugar só. Posse via colaboradorUseCase (PertenceAoLider).
	acompRepo := acompanhamento.NovoRepositorio(db)
	acompUseCase := acompanhamento.NovoUseCase(acompRepo, colaboradorUseCase)
	acompController := acompanhamento.NovoController(acompUseCase)
	acompController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: tabuleiro (persistência da pauta do 1:1) ────────────────────────
	// Board COLABORATIVO: posse via PodeAcessar (líder OU o próprio liderado).
	tabuleiroRepo := tabuleiro.NovoRepositorio(db)
	tabuleiroUseCase := tabuleiro.NovoUseCase(tabuleiroRepo, colaboradorUseCase)
	tabuleiroController := tabuleiro.NovoController(tabuleiroUseCase)
	tabuleiroController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: ia (BYOK — IA do gestor) ────────────────────────────────────────
	// Guarda o provedor + a chave cifrada e expõe config/chat. Segredo de cifragem:
	// IA_CRIPTO_SECRET (dedicado) se definido, senão o JWT_SECRET (retrocompatível).
	// O JWT_SECRET entra como fallback de decifra para não quebrar chaves já salvas
	// caso se migre para o segredo dedicado.
	iaSegredo := cfg.IACriptoSecret
	iaFallback := ""
	if iaSegredo == "" {
		iaSegredo = cfg.JWTSecret
	} else {
		iaFallback = cfg.JWTSecret
	}
	iaRepo := ia.NovoRepositorio(db)
	iaUseCase := ia.NovoUseCase(iaRepo, iaSegredo, iaFallback)
	iaController := ia.NovoController(iaUseCase)
	iaController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: ajuda (Central de Ajuda com IA) ─────────────────────────────────
	// Conteúdo curado (tópicos + tour, funcionam sem IA) + assistente de IA que resolve a
	// chave em cascata: PLATAFORMA (cfg.IAPlataforma*) → BYOK (reusa o iaUseCase) → curado.
	// Montado DEPOIS do módulo ia (depende do iaUseCase para o fallback BYOK).
	ajudaUseCase := ajuda.NovoUseCase(iaUseCase, cfg.IAPlataformaProvedor, cfg.IAPlataformaChave)
	ajudaController := ajuda.NovoController(ajudaUseCase)
	ajudaController.RegistrarRotas(api, authMiddleware, limiteIA)

	// ─── Módulo: admin (painel da plataforma — só a conta ADMIN) ─────────────────
	// Leitura agregada da plataforma inteira: contas, acessos (estilo Google Analytics),
	// uso, crescimento e saúde. Protegido por ApenasAdmin (dentro do RegistrarRotas).
	adminRepo := admin.NovoRepositorio(db)
	adminUseCase := admin.NovoUseCase(adminRepo)
	adminController := admin.NovoController(adminUseCase)
	adminController.RegistrarRotas(api, authMiddleware)

	// ─── Módulo: feedback (reações dos usuários → dashboard de gestão) ───────────
	// Escrita aberta a qualquer usuário logado (POST /feedback); o painel agregado
	// (GET /admin/feedbacks) é protegido por ApenasAdmin dentro do RegistrarRotas.
	feedbackRepo := feedback.NovoRepositorio(db)
	feedbackUseCase := feedback.NovoUseCase(feedbackRepo)
	feedbackController := feedback.NovoController(feedbackUseCase)
	feedbackController.RegistrarRotas(api, authMiddleware, limiteFeedback)

	// ─── Módulo: auditoria (rotas) ───────────────────────────────────────────────
	// O Controller já foi criado lá no início; aqui só registramos suas rotas.
	auditoriaController.RegistrarRotas(api, authMiddleware)

	// ─── WebSocket: 1:1 ao vivo ──────────────────────────────────────────────────
	// Registrado direto no router (fora do grupo /api/v1) para NÃO passar pelo
	// middleware de auditoria, que envolve o ResponseWriter e quebraria o upgrade
	// para WebSocket. A autenticação é feita pelo ?token= dentro do handler.
	hubAoVivo := aovivo.NovoHub()
	router.GET("/api/v1/ws/1a1/:sala", aovivo.Handler(hubAoVivo, cfg, colaboradorUseCase, db))

	return router
}
