// Pacote: internal/convite
// Arquivo: usecase.go
// Descrição: Regras de negócio dos convites de liderado. Gera o convite (link +
//            código), valida o acesso público e processa o aceite — criando ou
//            reutilizando a conta do liderado e vinculando-a ao colaborador.
//            Reaproveita os módulos usuario (criar conta + login/JWT) e
//            colaborador (buscar + vincular usuario_id).
// Autor: OneByOne API
// Criado em: 2025

package convite

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"onebyone-api/internal/colaborador"
	"onebyone-api/internal/usuario"
	"onebyone-api/pkg/email"
)

// Validade padrão de um convite (7 dias).
const validadeConvite = 7 * 24 * time.Hour

// Alfabeto do código sem caracteres ambíguos (sem O/0, I/1) para facilitar a digitação.
const alfabetoCodigo = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// UseCase define as operações de negócio dos convites.
type UseCase interface {
	// Gerar cria um convite para um colaborador do líder logado (usuarioID) e
	// devolve o token + código (uma vez). Valida a posse do colaborador.
	Gerar(colaboradorID string, usuarioID string) (ConviteGeradoDTO, error)
	// BuscarPublico devolve os dados públicos do convite (para a tela do liderado)
	BuscarPublico(token string) (ConvitePublicoDTO, error)
	// Aceitar valida o código, garante a conta do liderado, vincula ao colaborador
	// e devolve o login (token JWT) para acesso imediato
	Aceitar(token string, dto AceitarConviteDTO) (usuario.LoginRespostaDTO, error)
}

type useCaseImpl struct {
	repo          Repositorio
	usuarioUC     usuario.UseCase
	colaboradorUC colaborador.UseCase
	emailSvc      email.Servico
	appURL        string
}

// NovoUseCase cria o UseCase de convites com as dependências injetadas.
func NovoUseCase(
	repo Repositorio,
	usuarioUC usuario.UseCase,
	colaboradorUC colaborador.UseCase,
	emailSvc email.Servico,
	appURL string,
) UseCase {
	return &useCaseImpl{repo: repo, usuarioUC: usuarioUC, colaboradorUC: colaboradorUC, emailSvc: emailSvc, appURL: appURL}
}

// gerarCodigo gera um código aleatório de n caracteres usando crypto/rand.
func gerarCodigo(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = alfabetoCodigo[int(bytes[i])%len(alfabetoCodigo)]
	}
	return string(bytes), nil
}

func (uc *useCaseImpl) Gerar(colaboradorID string, usuarioID string) (ConviteGeradoDTO, error) {
	// POSSE: o colaborador precisa pertencer à estrutura do líder logado, senão
	// um líder poderia gerar convite (e tomar a conta) de liderado de outro líder.
	dono, err := uc.colaboradorUC.PertenceAoLider(colaboradorID, usuarioID)
	if err != nil {
		return ConviteGeradoDTO{}, err
	}
	if !dono {
		return ConviteGeradoDTO{}, fmt.Errorf("colaborador não encontrado")
	}

	// Cancela convites pendentes anteriores para manter só o mais recente válido.
	_ = uc.repo.CancelarPendentesDoColaborador(colaboradorID)

	codigo, err := gerarCodigo(6)
	if err != nil {
		return ConviteGeradoDTO{}, fmt.Errorf("erro ao gerar código: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(codigo), 12)
	if err != nil {
		return ConviteGeradoDTO{}, fmt.Errorf("erro ao processar código: %w", err)
	}

	novo := Convite{
		ID:            uuid.New().String(),
		ColaboradorID: colaboradorID,
		CodigoHash:    string(hash),
		Status:        StatusPendente,
		ExpiraEm:      time.Now().Add(validadeConvite),
		CriadoEm:      time.Now(),
	}

	if _, err := uc.repo.Criar(novo); err != nil {
		return ConviteGeradoDTO{}, fmt.Errorf("erro ao criar convite: %w", err)
	}

	// Envia o convite por e-mail ao liderado (dormente se o SMTP não estiver
	// ligado). Não bloqueia: o gestor também recebe link+código para compartilhar.
	if col, errCol := uc.colaboradorUC.BuscarInternoPorId(colaboradorID); errCol == nil && col.Email != "" {
		assunto, html := email.TemplateConvite(col.Nome, uc.appURL+"/convite/"+novo.ID, codigo)
		_ = uc.emailSvc.EnviarHTML([]string{col.Email}, assunto, html)
	}

	return ConviteGeradoDTO{
		Token:    novo.ID,
		Codigo:   codigo,
		Link:     "/convite/" + novo.ID,
		ExpiraEm: novo.ExpiraEm,
	}, nil
}

func (uc *useCaseImpl) BuscarPublico(token string) (ConvitePublicoDTO, error) {
	c, err := uc.repo.BuscarPorToken(token)
	if err != nil {
		return ConvitePublicoDTO{}, fmt.Errorf("convite não encontrado")
	}

	valido := c.Status == StatusPendente && time.Now().Before(c.ExpiraEm)

	// Leitura interna: o fluxo público é autorizado pelo token do convite.
	col, err := uc.colaboradorUC.BuscarInternoPorId(c.ColaboradorID)
	if err != nil {
		// Convite existe mas o colaborador sumiu — trata como inválido.
		return ConvitePublicoDTO{Token: token, Valido: false}, nil
	}

	return ConvitePublicoDTO{
		Token:           token,
		Valido:          valido,
		ColaboradorNome: col.Nome,
		Email:           col.Email,
	}, nil
}

func (uc *useCaseImpl) Aceitar(token string, dto AceitarConviteDTO) (usuario.LoginRespostaDTO, error) {
	vazio := usuario.LoginRespostaDTO{}

	c, err := uc.repo.BuscarPorToken(token)
	if err != nil {
		return vazio, fmt.Errorf("convite inválido")
	}
	if c.Status != StatusPendente {
		return vazio, fmt.Errorf("este convite já foi usado ou cancelado")
	}
	if time.Now().After(c.ExpiraEm) {
		return vazio, fmt.Errorf("este convite expirou")
	}

	// Confere o código (contra-senha) contra o hash.
	if err := bcrypt.CompareHashAndPassword([]byte(c.CodigoHash), []byte(dto.Codigo)); err != nil {
		return vazio, fmt.Errorf("código inválido")
	}

	// Leitura interna: o aceite é autorizado pelo token + código do convite.
	col, err := uc.colaboradorUC.BuscarInternoPorId(c.ColaboradorID)
	if err != nil {
		return vazio, fmt.Errorf("colaborador do convite não encontrado")
	}

	// Garante a conta do liderado:
	// - se já existe e a senha confere → usa (caso de troca de gestor/empresa);
	// - senão → cria uma conta nova com a senha informada.
	login, errLogin := uc.usuarioUC.Login(usuario.LoginDTO{Email: col.Email, Password: dto.Senha})
	if errLogin != nil {
		if _, errCriar := uc.usuarioUC.Criar(usuario.CriarUsuarioDTO{
			Nome:     col.Nome,
			Email:    col.Email,
			Password: dto.Senha,
			Role:     "COLABORADOR",
		}); errCriar != nil {
			// Se a criação falhou por e-mail já existente, é porque a senha estava errada.
			return vazio, fmt.Errorf("não foi possível acessar: se você já tem conta, use sua senha atual")
		}
		login, err = uc.usuarioUC.Login(usuario.LoginDTO{Email: col.Email, Password: dto.Senha})
		if err != nil {
			return vazio, fmt.Errorf("erro ao autenticar após criar o acesso")
		}
	}

	// Vincula a conta de usuário ao colaborador (liderado) via método dedicado.
	// O Atualizar geral não mexe mais em usuario_id (anti-sequestro de identidade).
	usuarioID := login.Usuario.ID
	if err := uc.colaboradorUC.VincularConta(c.ColaboradorID, usuarioID); err != nil {
		return vazio, fmt.Errorf("erro ao vincular a conta ao colaborador: %w", err)
	}

	// Marca o convite como aceito.
	_ = uc.repo.MarcarAceito(c.ID, time.Now())

	return login, nil
}
