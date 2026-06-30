// Pacote: internal/recuperacao
// Arquivo: usecase.go
// Descrição: Regras do "esqueci minha senha". Gera link (token) + código, manda por
//            e-mail, valida o token+código e troca a senha (delegando ao módulo usuario,
//            que aplica a complexidade). Segurança: anti-enumeração (não revela se o
//            e-mail existe), código cifrado, validade de 15 min e uso único.
// Autor: OneByOne API
// Criado em: 2026

package recuperacao

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"onebyone-api/internal/usuario"
	"onebyone-api/pkg/email"
)

// ErrTokenInvalido: link inexistente, expirado ou já usado.
var ErrTokenInvalido = errors.New("link inválido ou expirado")

// ErrCodigoInvalido: código (contra-senha) incorreto.
var ErrCodigoInvalido = errors.New("código inválido")

// maxTentativas: após esse número de códigos errados, o token é invalidado (anti-brute-force).
const maxTentativas = 5

// validadeLink: por quanto tempo o link + código de recuperação valem. Janela curta
// (15 min) por segurança — reduz o tempo em que um e-mail interceptado seria útil e
// estreita ainda mais a janela de brute-force do código de 6 dígitos. O texto do e-mail
// usa este mesmo número (minutos), então nunca fica fora de sincronia com a regra.
const validadeLink = 15 * time.Minute

// UseCase define as operações de recuperação de senha.
type UseCase interface {
	// Solicitar gera o pedido e envia o e-mail. SEMPRE devolve nil (anti-enumeração).
	Solicitar(dto SolicitarDTO) error
	// ValidarToken diz se o link ainda é válido (para o front mostrar o formulário).
	ValidarToken(token string) (bool, error)
	// Redefinir valida token + código e troca a senha (com checagem de complexidade).
	Redefinir(token string, dto RedefinirDTO) error
}

type useCaseImpl struct {
	repo      Repositorio
	usuarioUC usuario.UseCase
	emailSvc  email.Servico
	appURL    string
}

// NovoUseCase cria o UseCase de recuperação.
func NovoUseCase(repo Repositorio, usuarioUC usuario.UseCase, emailSvc email.Servico, appURL string) UseCase {
	return &useCaseImpl{repo: repo, usuarioUC: usuarioUC, emailSvc: emailSvc, appURL: appURL}
}

func (uc *useCaseImpl) Solicitar(dto SolicitarDTO) error {
	// Anti-enumeração: se o e-mail não existe, não fazemos nada e mesmo assim o controller
	// responde "se existir, enviamos" — não dá para descobrir quais e-mails têm conta.
	u, err := uc.usuarioUC.BuscarPorEmail(dto.Email)
	if err != nil {
		return nil
	}

	// Um pedido novo invalida os anteriores ainda pendentes.
	_ = uc.repo.InvalidarPendentesDoUsuario(u.ID)

	codigo := gerarCodigo()
	hash, err := bcrypt.GenerateFromPassword([]byte(codigo), 10)
	if err != nil {
		return nil
	}
	token := uuid.New().String()
	rec := Recuperacao{
		ID:         token,
		UsuarioID:  u.ID,
		CodigoHash: string(hash),
		Status:     StatusPendente,
		ExpiraEm:   time.Now().Add(validadeLink),
		CriadoEm:   time.Now(),
	}
	if err := uc.repo.Criar(rec); err != nil {
		return nil
	}

	// E-mail com o link + código (dormente se o SMTP não estiver configurado).
	if uc.emailSvc != nil {
		link := uc.appURL + "/redefinir-senha/" + token
		assunto, html := email.TemplateRecuperacaoSenha(u.Nome, link, codigo, int(validadeLink.Minutes()))
		go func() { _ = uc.emailSvc.EnviarHTML([]string{u.Email}, assunto, html) }()
	}
	return nil
}

func (uc *useCaseImpl) ValidarToken(token string) (bool, error) {
	rec, err := uc.repo.BuscarPorToken(token)
	if err != nil {
		return false, nil
	}
	return ehValido(rec), nil
}

func (uc *useCaseImpl) Redefinir(token string, dto RedefinirDTO) error {
	rec, err := uc.repo.BuscarPorToken(token)
	if err != nil || !ehValido(rec) {
		return ErrTokenInvalido
	}
	if err := bcrypt.CompareHashAndPassword([]byte(rec.CodigoHash), []byte(dto.Codigo)); err != nil {
		// Código errado: conta a tentativa e, ao atingir o limite, invalida o token —
		// trava o brute-force do código de 6 dígitos dentro da janela de 15 minutos.
		_ = uc.repo.IncrementarTentativa(rec.ID)
		if rec.Tentativas+1 >= maxTentativas {
			_ = uc.repo.MarcarUsado(rec.ID, time.Now())
			return ErrTokenInvalido
		}
		return ErrCodigoInvalido
	}
	// Troca a senha (o módulo usuario valida a complexidade e aplica o bcrypt).
	if err := uc.usuarioUC.RedefinirSenha(rec.UsuarioID, dto.NovaSenha); err != nil {
		return err
	}
	_ = uc.repo.MarcarUsado(rec.ID, time.Now())
	return nil
}

// ehValido confere status pendente e validade.
func ehValido(rec Recuperacao) bool {
	return rec.Status == StatusPendente && time.Now().Before(rec.ExpiraEm)
}

// gerarCodigo devolve um código numérico de 6 dígitos, com aleatoriedade criptográfica.
func gerarCodigo() string {
	const digitos = "0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digitos))))
		if err != nil {
			n = big.NewInt(0)
		}
		b[i] = digitos[n.Int64()]
	}
	return string(b)
}
