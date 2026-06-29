// Pacote: pkg/config
// Arquivo: config.go
// Descrição: Responsável por carregar e expor todas as variáveis de ambiente
//            necessárias para o funcionamento da aplicação. Utiliza godotenv
//            para ler o arquivo .env em desenvolvimento.
// Autor: OneByOne API
// Criado em: 2025

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config armazena todas as configurações da aplicação carregadas via variáveis de ambiente
type Config struct {
	// DBHost é o endereço do servidor MySQL (ex.: localhost ou IP do container)
	DBHost string
	// DBPort é a porta de acesso ao servidor MySQL (padrão: 3306)
	DBPort string
	// DBUser é o nome do usuário de acesso ao banco de dados
	DBUser string
	// DBPassword é a senha do usuário de acesso ao banco de dados
	DBPassword string
	// DBName é o nome do schema/database a ser utilizado
	DBName string
	// JWTSecret é a chave secreta usada para assinar e verificar tokens JWT
	JWTSecret string
	// JWTExpiracaoHoras define quantas horas um token JWT permanece válido após a emissão
	JWTExpiracaoHoras int
	// PortaAPI é a porta TCP em que o servidor HTTP irá escutar as requisições
	PortaAPI string
	// AWSAccessKeyID é a chave de acesso do usuário IAM com permissão no bucket S3
	AWSAccessKeyID string
	// AWSSecretAccessKey é a chave secreta correspondente ao AWSAccessKeyID
	AWSSecretAccessKey string
	// AWSRegion é a região AWS onde o bucket S3 está hospedado (ex: us-east-1)
	AWSRegion string
	// AWSBucket é o nome do bucket S3 onde as fotos serão armazenadas
	AWSBucket string
	// AWSPrefixo é o prefixo de pasta dentro do bucket que isola os arquivos deste projeto
	AWSPrefixo string
	// ── E-mail (SMTP) — opcional. Vazio = envio de e-mail desligado (dormente). ──
	// SMTPHost é o servidor SMTP (ex.: email-smtp.us-east-1.amazonaws.com no AWS SES)
	SMTPHost string
	// SMTPPort é a porta SMTP (ex.: 587)
	SMTPPort string
	// SMTPUser e SMTPPassword são as credenciais SMTP
	SMTPUser     string
	SMTPPassword string
	// SMTPRemetente é o e-mail "De" (precisa ser verificado no provedor, ex.: SES)
	SMTPRemetente string
	// AppURL é a URL pública do app (usada nos links dos e-mails)
	AppURL string
	// Ambiente: "producao" liga as travas de segurança (JWT forte, origem do WS restrita,
	// APP_URL https obrigatório). Em "desenvolvimento" (padrão) as checagens são brandas.
	Ambiente string
	// reCAPTCHA (anti-bot). Vazios = DESLIGADO (dormente). Quando preenchidos, ativam a
	// verificação no login/cadastro/recuperação. SiteKey é público (vai pro front); Secret
	// fica só no servidor.
	RecaptchaSiteKey string
	RecaptchaSecret  string
}

// Carregar lê as variáveis de ambiente do arquivo .env (ou do ambiente do sistema)
// e retorna uma instância de Config preenchida com os valores encontrados.
// Valores ausentes recebem defaults razoáveis para desenvolvimento local.
func Carregar() (*Config, error) {
	// Perfil de ambiente — no espírito do `spring.profiles.active` do Spring Boot.
	// O AMBIENTE vem do SO / docker-compose (NUNCA do arquivo, senão seria circular).
	// Carregamos o arquivo do perfil:
	//   • desenvolvimento → .env.dev  (como o application-dev.yml)
	//   • produção        → .env      (como o application.yml)
	// O godotenv NÃO sobrescreve variáveis já definidas no ambiente, então o que vier do
	// container/compose sempre vence o arquivo. Se o arquivo do perfil não existir (ex.:
	// em produção tudo vem do ambiente), caímos no .env genérico (retrocompatível).
	ambiente := getEnv("AMBIENTE", "desenvolvimento")
	arquivoPerfil := ".env.dev"
	if ambiente == "producao" {
		arquivoPerfil = ".env"
	}
	if err := godotenv.Load(arquivoPerfil); err != nil {
		_ = godotenv.Load(".env")
	}

	// Converte JWT_EXPIRACAO_HORAS para inteiro; usa 24h como padrão caso inválido
	expiracaoHoras, err := strconv.Atoi(getEnv("JWT_EXPIRACAO_HORAS", "24"))
	if err != nil {
		// Valor inválido na variável de ambiente — aplica o padrão seguro de 24 horas
		expiracaoHoras = 24
	}

	cfg := &Config{
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "3306"),
		DBUser:             getEnv("DB_USER", "root"),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", "onebyone"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpiracaoHoras:  expiracaoHoras,
		PortaAPI:           getEnv("PORTA_API", "8080"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSBucket:          getEnv("AWS_BUCKET", "controleazul"),
		AWSPrefixo:         getEnv("AWS_PREFIXO", "one-by-one"),
		SMTPHost:           getEnv("SMTP_HOST", ""),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		SMTPUser:           getEnv("SMTP_USER", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		SMTPRemetente:      getEnv("SMTP_REMETENTE", ""),
		AppURL:             getEnv("APP_URL", "http://localhost:3100"),
		Ambiente:           getEnv("AMBIENTE", "desenvolvimento"),
		RecaptchaSiteKey:   getEnv("RECAPTCHA_SITE_KEY", ""),
		RecaptchaSecret:    getEnv("RECAPTCHA_SECRET", ""),
	}

	// Travas de segurança (fail-fast): o app NÃO sobe com configuração insegura.
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET é obrigatório — defina no .env uma chave longa e aleatória")
	}
	if cfg.Ambiente == "producao" {
		if strings.Contains(strings.ToLower(cfg.JWTSecret), "troque") {
			return nil, fmt.Errorf("JWT_SECRET ainda é o valor de exemplo — gere um real (ex.: openssl rand -base64 48)")
		}
		if len(cfg.JWTSecret) < 32 {
			return nil, fmt.Errorf("JWT_SECRET fraco para produção: use >= 32 caracteres aleatórios (ex.: openssl rand -base64 48)")
		}
		if !strings.HasPrefix(cfg.AppURL, "https://") {
			return nil, fmt.Errorf("APP_URL precisa ser https em produção (ex.: https://seudominio.com)")
		}
	}

	return cfg, nil
}

// getEnv retorna o valor da variável de ambiente identificada por chave,
// ou valorPadrao caso a variável não exista ou esteja vazia
func getEnv(chave, valorPadrao string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return valorPadrao
}
