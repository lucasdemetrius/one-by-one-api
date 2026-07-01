// Pacote: internal/usuario
// Arquivo: usecase.go
// Descrição: Contém todas as regras de negócio do módulo de usuário.
//            Intermediário entre o Controller (HTTP) e o Repository (banco).
//            Nunca conhece detalhes de request/response HTTP.
// Autor: OneByOne API
// Criado em: 2025

package usuario

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"onebyone-api/pkg/config"
	"onebyone-api/pkg/email"
	"onebyone-api/pkg/middleware"
	"onebyone-api/pkg/senha"
	"onebyone-api/pkg/storage"
	"onebyone-api/pkg/texto"
)

// ErrAcessoNegado indica que o solicitante não tem permissão sobre a conta alvo.
// Mantém a mensagem genérica "não encontrado" para não revelar a existência do recurso
// (o controller mapeia para HTTP 404, nunca 403) — mesmo padrão dos demais módulos.
var ErrAcessoNegado = errors.New("usuário não encontrado")

// ErrPapelInvalidoNoCadastro indica tentativa de auto-cadastro público com papel não
// permitido. Só Gestor (LIDER) ou RH podem se auto-cadastrar; liderado entra por convite.
var ErrPapelInvalidoNoCadastro = errors.New("cadastro público disponível apenas para Gestor ou RH")

// UseCase define o contrato das operações de negócio do módulo de usuário.
// O Controller depende desta interface, não da implementação concreta.
type UseCase interface {
	// Criar persiste um novo usuário com a role informada (padrão COLABORADOR) e rh_id
	// nulo. É um método INTERNO/confiável — usado pelo aceite de convite para criar a
	// conta do liderado. NÃO é exposto por rota pública genérica (use Registrar).
	Criar(dto CriarUsuarioDTO) (UsuarioRespostaDTO, error)
	// Registrar é o auto-cadastro PÚBLICO. Só permite criar Gestor (LIDER) ou RH — nunca
	// COLABORADOR — e sempre com rh_id nulo (gestor solo ou RH raiz). O vínculo gestor→RH
	// nunca vem do cliente; é derivado no servidor nos fluxos autenticados.
	Registrar(dto CriarUsuarioDTO) (UsuarioRespostaDTO, error)
	// CriarGestorParaRH cria a conta de um GESTOR (LIDER) já vinculada ao tenant do RH
	// (rh_id = rhID, derivado do JWT do RH — nunca do corpo). Usado pelo módulo /rh.
	CriarGestorParaRH(dto CriarUsuarioDTO, rhID string) (UsuarioRespostaDTO, error)
	// BuscarPorId retorna o DTO do usuário SE o solicitante for o próprio dono da conta
	// (self-service). Caso contrário devolve ErrAcessoNegado (mapeado para 404).
	BuscarPorId(id string, solicitanteID string) (UsuarioRespostaDTO, error)
	// Atualizar altera nome/e-mail da PRÓPRIA conta do solicitante. Nunca altera a role.
	Atualizar(id string, solicitanteID string, dto AtualizarUsuarioDTO) (UsuarioRespostaDTO, error)
	// Deletar realiza a exclusão lógica da PRÓPRIA conta do solicitante.
	Deletar(id string, deletadoPor string) error
	// Login autentica o usuário com e-mail e senha e retorna um token JWT
	Login(dto LoginDTO) (LoginRespostaDTO, error)
	// LoginGoogle autentica via Google (OAuth): valida o ID token do Google e faz
	// login (conta existente) ou cadastro de Gestor (conta nova). Mesmo JWT do Login.
	LoginGoogle(dto LoginGoogleDTO) (LoginRespostaDTO, error)
	// UploadFoto envia a foto da PRÓPRIA conta do solicitante para o S3 e persiste a chave.
	UploadFoto(id string, solicitanteID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (UsuarioRespostaDTO, error)
	// BuscarPorEmail localiza um usuário ativo pelo e-mail. Uso interno confiável (ex.:
	// fluxo de recuperação de senha). Devolve erro se não existir.
	BuscarPorEmail(email string) (UsuarioRespostaDTO, error)
	// RedefinirSenha valida a complexidade e troca a senha de um usuário (por id). Usado
	// pelo fluxo de recuperação, depois de validar o token + código.
	RedefinirSenha(usuarioID, novaSenha string) error
}

// useCaseImpl é a implementação concreta do UseCase de usuário
type useCaseImpl struct {
	// repo é o repositório de acesso ao banco, injetado via interface
	repo Repositorio
	// cfg fornece as configurações necessárias para geração do JWT
	cfg *config.Config
	// armazenamento é o serviço S3 para upload e geração de URLs presignadas de fotos
	armazenamento storage.Armazenamento
	// emailSvc envia e-mails (ex.: boas-vindas do gestor). Pode estar dormente.
	emailSvc email.Servico
}

// NovoUseCase cria e retorna uma nova instância do UseCase de usuário com as dependências injetadas
func NovoUseCase(repo Repositorio, cfg *config.Config, armazenamento storage.Armazenamento, emailSvc email.Servico) UseCase {
	return &useCaseImpl{repo: repo, cfg: cfg, armazenamento: armazenamento, emailSvc: emailSvc}
}

// emailReservado indica se o e-mail é o da conta de ADMIN da plataforma (cfg.AdminEmail).
// Esse e-mail é gerenciado APENAS pelo seed de boot (admin.GarantirContaAdmin): nenhuma via
// pública (auto-cadastro, convite, edição de perfil ou cadastro pelo RH) pode tomá-lo. Sem
// essa reserva, alguém poderia registrar a conta com esse e-mail ANTES de o operador subir o
// admin e, no próximo boot, o seed promoveria essa conta a ADMIN (escalonamento de privilégio).
func (uc *useCaseImpl) emailReservado(email string) bool {
	return uc.cfg != nil && uc.cfg.AdminEmail != "" && email == uc.cfg.AdminEmail
}

// Criar verifica unicidade do e-mail, gera UUID, aplica hash bcrypt na senha
// e persiste o novo usuário. Retorna o DTO sem expor dados sensíveis.
func (uc *useCaseImpl) Criar(dto CriarUsuarioDTO) (UsuarioRespostaDTO, error) {
	// Normaliza o e-mail (minúsculo + sem espaços) para gravar e comparar de forma canônica.
	dto.Email = texto.NormalizarEmail(dto.Email)

	// O e-mail do ADMIN da plataforma é reservado: nenhuma via de criação pode usá-lo
	// (anti-escalonamento). Responde como "já em uso" — não revela que é a conta de admin.
	if uc.emailReservado(dto.Email) {
		return UsuarioRespostaDTO{}, fmt.Errorf("já existe um usuário com este e-mail")
	}

	// Verifica se o e-mail já está em uso por outro usuário ativo para evitar duplicatas
	if _, err := uc.repo.BuscarPorEmail(dto.Email); err == nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("já existe um usuário com este e-mail")
	}

	// Gera o hash da senha com custo 12 — equilibra segurança e performance em produção
	hashSenha, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao processar senha: %w", err)
	}

	// Define COLABORADOR como role padrão caso o cliente não informe
	role := dto.Role
	if role == "" {
		role = "COLABORADOR"
	}

	novoUsuario := Usuario{
		ID:       uuid.New().String(),
		Nome:     dto.Nome,
		Email:    dto.Email,
		Password: string(hashSenha),
		Role:     role,
		CriadoEm: time.Now(),
	}

	criado, err := uc.repo.Criar(novoUsuario)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao criar usuário: %w", err)
	}

	// E-mail de boas-vindas — só para o gestor (LIDER). Envio assíncrono; se o
	// SMTP não estiver configurado, o serviço apenas loga (dormente).
	if criado.Role == "LIDER" && uc.emailSvc != nil {
		assunto, html := email.TemplateBoasVindas(criado.Nome, uc.cfg.AppURL)
		go func() { _ = uc.emailSvc.EnviarHTML([]string{criado.Email}, assunto, html) }()
	}

	return ParaRespostaDTO(criado, nil), nil
}

// BuscarPorEmail localiza um usuário ativo pelo e-mail (uso interno confiável, ex.:
// recuperação de senha). Devolve erro genérico se não existir.
func (uc *useCaseImpl) BuscarPorEmail(email string) (UsuarioRespostaDTO, error) {
	u, err := uc.repo.BuscarPorEmail(texto.NormalizarEmail(email))
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("usuário não encontrado")
	}
	return ParaRespostaDTO(u, nil), nil
}

// RedefinirSenha valida a complexidade e troca a senha do usuário (por id). Usado pelo
// fluxo de recuperação, depois de validar o token + código.
func (uc *useCaseImpl) RedefinirSenha(usuarioID, novaSenha string) error {
	if err := senha.Validar(novaSenha); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(novaSenha), 12)
	if err != nil {
		return fmt.Errorf("erro ao processar senha: %w", err)
	}
	if err := uc.repo.AtualizarSenha(usuarioID, string(hash)); err != nil {
		return err
	}
	// Senha trocada → invalida todos os tokens antigos (revogação de sessão).
	_ = uc.repo.IncrementarVersaoToken(usuarioID)
	return nil
}

// Registrar é o auto-cadastro público (POST /auth/registrar). Sela o papel: aceita só
// Gestor (LIDER) ou RH; ausência de role vira LIDER (gestor solo). Bloqueia COLABORADOR
// (liderado entra por convite, não por auto-cadastro) e qualquer outro valor. O rh_id
// nasce nulo — um RH que se auto-cadastra é uma RAIZ vazia, sem nenhum gestor abaixo,
// portanto não enxerga tenant algum. Isso neutraliza escalonamento de privilégio: não há
// como se atrelar a uma empresa existente pelo cadastro.
func (uc *useCaseImpl) Registrar(dto CriarUsuarioDTO) (UsuarioRespostaDTO, error) {
	switch dto.Role {
	case "":
		dto.Role = "LIDER"
	case "LIDER", "RH":
		// papéis permitidos no auto-cadastro
	default:
		return UsuarioRespostaDTO{}, ErrPapelInvalidoNoCadastro
	}
	// Reaproveita toda a lógica de Criar (unicidade de e-mail, hash, persistência).
	// Criar não preenche rh_id, então a conta nasce com rh_id nulo (raiz/solo).
	return uc.Criar(dto)
}

// CriarGestorParaRH cria a conta de um GESTOR (role LIDER) já amarrada ao tenant do RH.
// O rhID vem SEMPRE do JWT do RH autenticado (no controller do módulo /rh), nunca do
// corpo da requisição — é isso que impede um RH de "adotar" gestores de outro tenant.
func (uc *useCaseImpl) CriarGestorParaRH(dto CriarUsuarioDTO, rhID string) (UsuarioRespostaDTO, error) {
	dto.Email = texto.NormalizarEmail(dto.Email)
	// E-mail do ADMIN é reservado (anti-escalonamento) — o RH também não pode atribuí-lo.
	if uc.emailReservado(dto.Email) {
		return UsuarioRespostaDTO{}, fmt.Errorf("já existe um usuário com este e-mail")
	}
	if _, err := uc.repo.BuscarPorEmail(dto.Email); err == nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("já existe um usuário com este e-mail")
	}

	hashSenha, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao processar senha: %w", err)
	}

	novo := Usuario{
		ID:       uuid.New().String(),
		Nome:     dto.Nome,
		Email:    dto.Email,
		Password: string(hashSenha),
		Role:     "LIDER", // o RH cadastra GESTORES
		RhID:     &rhID,   // vínculo com o tenant do RH (derivado do JWT)
		CriadoEm: time.Now(),
	}

	criado, err := uc.repo.Criar(novo)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao criar gestor: %w", err)
	}

	// E-mail de boas-vindas do gestor (mesmo template do auto-cadastro).
	if uc.emailSvc != nil {
		assunto, html := email.TemplateBoasVindas(criado.Nome, uc.cfg.AppURL)
		go func() { _ = uc.emailSvc.EnviarHTML([]string{criado.Email}, assunto, html) }()
	}

	return ParaRespostaDTO(criado, nil), nil
}

// gerarFotoURL gera uma URL presignada para a foto do usuário se a chave S3 estiver preenchida.
// Retorna nil silenciosamente em caso de erro para não bloquear listagens.
func (uc *useCaseImpl) gerarFotoURL(fotoKey *string) *string {
	if fotoKey == nil || uc.armazenamento == nil {
		return nil
	}
	url, err := uc.armazenamento.GerarURLPresignada(*fotoKey, storage.ExpiracaoURLFoto)
	if err != nil {
		return nil
	}
	return &url
}

// BuscarPorId retorna a conta SE o solicitante for o próprio dono (self-service).
// Conta de outro usuário responde ErrAcessoNegado → 404 (não revela existência).
func (uc *useCaseImpl) BuscarPorId(id string, solicitanteID string) (UsuarioRespostaDTO, error) {
	if id != solicitanteID {
		return UsuarioRespostaDTO{}, ErrAcessoNegado
	}
	u, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("usuário não encontrado: %w", err)
	}
	return ParaRespostaDTO(u, uc.gerarFotoURL(u.FotoKey)), nil
}

// Atualizar aplica apenas os campos informados no DTO, preservando os demais.
// Verifica unicidade do novo e-mail antes de aplicar a alteração.
func (uc *useCaseImpl) Atualizar(id string, solicitanteID string, dto AtualizarUsuarioDTO) (UsuarioRespostaDTO, error) {
	// Self-service: só o próprio dono edita sua conta. Recurso alheio = 404.
	if id != solicitanteID {
		return UsuarioRespostaDTO{}, ErrAcessoNegado
	}

	// Carrega o estado atual do usuário para aplicar as alterações parciais
	atual, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("usuário não encontrado: %w", err)
	}

	// Aplica cada campo somente se foi informado no DTO (atualização parcial)
	if dto.Nome != "" {
		atual.Nome = dto.Nome
	}

	if dto.Email != "" {
		dto.Email = texto.NormalizarEmail(dto.Email)
	}
	if dto.Email != "" && dto.Email != atual.Email {
		// O e-mail do ADMIN é reservado: ninguém pode RENOMEAR sua conta para ele e ser
		// promovido a ADMIN no próximo boot (anti-escalonamento). Responde como "em uso".
		if uc.emailReservado(dto.Email) {
			return UsuarioRespostaDTO{}, fmt.Errorf("este e-mail já está em uso por outro usuário")
		}
		// Garante que o novo e-mail não pertence a outro usuário ativo
		existente, err := uc.repo.BuscarPorEmail(dto.Email)
		if err == nil && existente.ID != id {
			return UsuarioRespostaDTO{}, fmt.Errorf("este e-mail já está em uso por outro usuário")
		}
		atual.Email = dto.Email
	}

	// A role NÃO é alterável por aqui de propósito (anti-escalonamento). Mudança de
	// papel só por fluxos controlados (cadastro selado, convite, gestão pelo RH).

	atualizado, err := uc.repo.Atualizar(atual)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao atualizar usuário: %w", err)
	}

	return ParaRespostaDTO(atualizado, uc.gerarFotoURL(atualizado.FotoKey)), nil
}

// Deletar verifica a existência do usuário e delega a exclusão lógica ao repositório
func (uc *useCaseImpl) Deletar(id string, deletadoPor string) error {
	// Self-service: só o próprio dono exclui sua conta. (Gestão de gestores pelo RH e
	// de liderados pelo gestor vivem em outros módulos, com suas próprias regras.)
	if id != deletadoPor {
		return ErrAcessoNegado
	}
	// Confirma que o usuário existe e está ativo antes de deletar
	if _, err := uc.repo.BuscarPorId(id); err != nil {
		return fmt.Errorf("usuário não encontrado: %w", err)
	}
	return uc.repo.DeletarSoft(id, deletadoPor)
}

// Lockout de login: após maxFalhasLogin falhas consecutivas, a conta é bloqueada por
// bloqueioLogin (segunda linha de defesa, por conta, além do rate-limit por IP).
const (
	maxFalhasLogin  = 5
	bloqueioMinutos = 15
)

// Login verifica as credenciais informadas e emite um token JWT em caso de sucesso.
// Usa mensagem genérica em caso de falha para não revelar se o e-mail existe no sistema.
func (uc *useCaseImpl) Login(dto LoginDTO) (LoginRespostaDTO, error) {
	// Normaliza para casar com o e-mail gravado em forma canônica (minúsculo).
	dto.Email = texto.NormalizarEmail(dto.Email)
	// Busca o usuário pelo e-mail — falha silenciosa para não revelar cadastros
	u, err := uc.repo.BuscarPorEmail(dto.Email)
	if err != nil {
		return LoginRespostaDTO{}, fmt.Errorf("credenciais inválidas")
	}

	// Lockout: conta temporariamente bloqueada após muitas falhas → mensagem genérica
	// (não revela o bloqueio nem a existência do e-mail).
	if u.BloqueadoAte != nil && u.BloqueadoAte.After(time.Now()) {
		return LoginRespostaDTO{}, fmt.Errorf("credenciais inválidas")
	}

	// Compara a senha informada com o hash bcrypt armazenado no banco
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(dto.Password)); err != nil {
		// Senha errada: incrementa e, ao atingir o limite, bloqueia — numa única instrução
		// atômica (sem corrida sob requisições paralelas).
		_ = uc.repo.RegistrarFalhaLogin(u.ID, maxFalhasLogin, bloqueioMinutos)
		return LoginRespostaDTO{}, fmt.Errorf("credenciais inválidas")
	}

	// Login OK: zera o contador de falhas e o bloqueio.
	_ = uc.repo.ZerarFalhaLogin(u.ID)

	// Emite o JWT (mesma emissão usada pelo login com Google).
	return uc.gerarTokenLogin(u)
}

// gerarTokenLogin monta os claims, assina o JWT (HS256) e devolve o DTO de login.
// Centraliza a emissão do token para ser reaproveitada pelo login por senha e pelo
// login com Google — os dois entregam exatamente o mesmo token.
func (uc *useCaseImpl) gerarTokenLogin(u Usuario) (LoginRespostaDTO, error) {
	claims := middleware.ClaimsJWT{
		UsuarioID: u.ID,
		Role:      u.Role,
		Versao:    u.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(time.Duration(uc.cfg.JWTExpiracaoHoras) * time.Hour),
			),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	// Assina o token com HS256 usando a chave secreta definida em JWT_SECRET no .env
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(uc.cfg.JWTSecret))
	if err != nil {
		return LoginRespostaDTO{}, fmt.Errorf("erro ao gerar token de autenticação: %w", err)
	}

	return LoginRespostaDTO{
		Token:   tokenString,
		Usuario: ParaRespostaDTO(u, uc.gerarFotoURL(u.FotoKey)),
	}, nil
}

// LoginGoogle autentica via Google (OAuth). Valida o ID token do Google e então:
//   - se já existe uma conta ATIVA com aquele e-mail → faz login (reutiliza a conta);
//   - se não existe → cria uma conta nova de Gestor (LIDER), com senha aleatória
//     inutilizável (a coluna é NOT NULL; login por senha fica impossível para ela).
//
// Emite o MESMO JWT do login por senha. Respeita a regra "1 e-mail = 1 conta" e a
// reserva do e-mail de ADMIN. Exige GOOGLE_CLIENT_ID configurado e e-mail verificado.
func (uc *useCaseImpl) LoginGoogle(dto LoginGoogleDTO) (LoginRespostaDTO, error) {
	if uc.cfg == nil || uc.cfg.GoogleClientID == "" {
		return LoginRespostaDTO{}, fmt.Errorf("login com Google não está configurado")
	}

	// Valida assinatura + emissor + audiência + prazo do token do Google.
	dados, err := validarIDTokenGoogle(dto.Credential, uc.cfg.GoogleClientID)
	if err != nil {
		return LoginRespostaDTO{}, fmt.Errorf("credenciais inválidas")
	}
	// Só aceitamos e-mail JÁ VERIFICADO pelo Google — senão daria para forjar o dono.
	if !dados.EmailVerified {
		return LoginRespostaDTO{}, fmt.Errorf("credenciais inválidas")
	}

	emailNorm := texto.NormalizarEmail(dados.Email)

	// O e-mail do ADMIN é reservado — nunca cria/loga por aqui (anti-escalonamento).
	if uc.emailReservado(emailNorm) {
		return LoginRespostaDTO{}, fmt.Errorf("credenciais inválidas")
	}

	// Já existe conta ativa com esse e-mail? → login (reutiliza a conta).
	if u, err := uc.repo.BuscarPorEmail(emailNorm); err == nil {
		return uc.gerarTokenLogin(u)
	}

	// Não existe → cria conta nova de Gestor (LIDER) com senha aleatória inutilizável.
	senhaAleatoria := uuid.New().String() + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(senhaAleatoria), 12)
	if err != nil {
		return LoginRespostaDTO{}, fmt.Errorf("erro ao processar conta: %w", err)
	}

	nome := dados.Nome
	if nome == "" {
		nome = emailNorm
	}

	criado, err := uc.repo.Criar(Usuario{
		ID:       uuid.New().String(),
		Nome:     nome,
		Email:    emailNorm,
		Password: string(hash),
		Role:     "LIDER", // conta nova via Google nasce como Gestor solo (rh_id nulo)
		CriadoEm: time.Now(),
	})
	if err != nil {
		return LoginRespostaDTO{}, fmt.Errorf("erro ao criar conta: %w", err)
	}

	// Boas-vindas (mesmo template do auto-cadastro); assíncrono e dormente sem SMTP.
	if uc.emailSvc != nil {
		assunto, html := email.TemplateBoasVindas(criado.Nome, uc.cfg.AppURL)
		go func() { _ = uc.emailSvc.EnviarHTML([]string{criado.Email}, assunto, html) }()
	}

	return uc.gerarTokenLogin(criado)
}

// UploadFoto envia o arquivo para o S3 usando a chave padrão "usuarios/{id}/foto.{ext}",
// persiste a chave no banco e retorna o DTO atualizado com a nova URL presignada.
func (uc *useCaseImpl) UploadFoto(id string, solicitanteID string, arquivo io.Reader, tamanho int64, tipoConteudo string) (UsuarioRespostaDTO, error) {
	// Self-service: só o próprio dono troca a foto da sua conta.
	if id != solicitanteID {
		return UsuarioRespostaDTO{}, ErrAcessoNegado
	}
	if _, err := uc.repo.BuscarPorId(id); err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("usuário não encontrado: %w", err)
	}

	// A extensão é derivada do Content-Type para manter o objeto identificável no S3
	ext := extensaoPorTipo(tipoConteudo)
	caminho := fmt.Sprintf("usuarios/%s/foto%s", id, ext)
	chave := uc.armazenamento.ChaveCompleta(caminho)

	if err := uc.armazenamento.Upload(chave, arquivo, tamanho, tipoConteudo); err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao enviar foto: %w", err)
	}

	if err := uc.repo.AtualizarFoto(id, chave); err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao salvar chave da foto: %w", err)
	}

	atualizado, err := uc.repo.BuscarPorId(id)
	if err != nil {
		return UsuarioRespostaDTO{}, fmt.Errorf("erro ao buscar usuário após upload: %w", err)
	}

	return ParaRespostaDTO(atualizado, uc.gerarFotoURL(atualizado.FotoKey)), nil
}

// extensaoPorTipo retorna a extensão de arquivo correspondente ao Content-Type informado
func extensaoPorTipo(tipoConteudo string) string {
	switch tipoConteudo {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}
