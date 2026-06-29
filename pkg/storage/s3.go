// Pacote: pkg/storage
// Arquivo: s3.go
// Descrição: Implementação do serviço de armazenamento de arquivos usando AWS S3.
//            Todos os objetos são privados — o acesso é concedido exclusivamente
//            via URLs presignadas com validade configurável, geradas sob demanda.
//            As chaves são prefixadas com o prefixo configurado no .env para
//            isolar os arquivos deste projeto dentro do bucket compartilhado.
// Autor: OneByOne API
// Criado em: 2025

package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appcfg "onebyone-api/pkg/config"
)

// ExpiracaoURLFoto define o tempo de validade padrão das URLs presignadas de foto.
// Após esse período, o frontend precisará chamar novamente a API para obter uma nova URL.
const ExpiracaoURLFoto = 2 * time.Hour

// Armazenamento define o contrato para upload e acesso a arquivos no S3
type Armazenamento interface {
	// Upload envia um arquivo para o S3 com a chave informada
	Upload(chave string, arquivo io.Reader, tamanho int64, tipoConteudo string) error
	// GerarURLPresignada gera uma URL temporária e privada para acesso ao arquivo
	GerarURLPresignada(chave string, expiracao time.Duration) (string, error)
	// Deletar remove um arquivo do S3 pela sua chave
	Deletar(chave string) error
	// ChaveCompleta retorna a chave com o prefixo do projeto aplicado
	ChaveCompleta(caminho string) string
}

// s3Armazenamento é a implementação concreta do Armazenamento usando AWS S3
type s3Armazenamento struct {
	// cliente é o cliente S3 do SDK AWS v2
	cliente *s3.Client
	// presign é o cliente responsável por assinar URLs temporárias
	presign *s3.PresignClient
	// bucket é o nome do bucket S3 onde os arquivos são armazenados
	bucket string
	// prefixo isola os arquivos deste projeto dentro do bucket (ex: "one-by-one")
	prefixo string
}

// NovoArmazenamentoS3 cria e valida uma nova instância do serviço de armazenamento S3
func NovoArmazenamentoS3(cfg *appcfg.Config) (Armazenamento, error) {
	// Carrega a configuração AWS com as credenciais estáticas definidas no .env
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.AWSRegion),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AWSAccessKeyID,
				cfg.AWSSecretAccessKey,
				"", // session token — vazio para credenciais de usuário IAM permanente
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao configurar credenciais AWS: %w", err)
	}

	cliente := s3.NewFromConfig(awsCfg)

	return &s3Armazenamento{
		cliente: cliente,
		presign: s3.NewPresignClient(cliente),
		bucket:  cfg.AWSBucket,
		prefixo: cfg.AWSPrefixo,
	}, nil
}

// ChaveCompleta retorna o caminho completo do objeto no S3 incluindo o prefixo do projeto.
// Exemplo: "usuarios/abc-123/foto.jpg" → "one-by-one/usuarios/abc-123/foto.jpg"
func (s *s3Armazenamento) ChaveCompleta(caminho string) string {
	if s.prefixo == "" {
		return caminho
	}
	return fmt.Sprintf("%s/%s", s.prefixo, caminho)
}

// Upload envia o arquivo para o S3 com a chave completa (prefixo + caminho).
// Os objetos são sempre privados — nunca recebem ACL pública.
func (s *s3Armazenamento) Upload(chave string, arquivo io.Reader, tamanho int64, tipoConteudo string) error {
	_, err := s.cliente.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(chave),
		Body:          arquivo,
		ContentLength: aws.Int64(tamanho),
		ContentType:   aws.String(tipoConteudo),
		// Sem ACL pública — o acesso é feito exclusivamente via URL presignada
	})
	if err != nil {
		return fmt.Errorf("erro ao enviar arquivo para o S3 (chave: %s): %w", chave, err)
	}
	return nil
}

// GerarURLPresignada cria uma URL temporária autenticada para o objeto privado.
// A URL expira após o período informado — o frontend deve renovar chamando a API.
func (s *s3Armazenamento) GerarURLPresignada(chave string, expiracao time.Duration) (string, error) {
	// PresignGetObject assina criptograficamente a URL usando as credenciais IAM;
	// nenhuma chamada de rede é feita — a operação é puramente local
	req, err := s.presign.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(chave),
		},
		s3.WithPresignExpires(expiracao),
	)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar URL presignada (chave: %s): %w", chave, err)
	}
	return req.URL, nil
}

// Deletar remove permanentemente o arquivo do S3 pela sua chave completa
func (s *s3Armazenamento) Deletar(chave string) error {
	_, err := s.cliente.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(chave),
	})
	if err != nil {
		return fmt.Errorf("erro ao deletar arquivo do S3 (chave: %s): %w", chave, err)
	}
	return nil
}
