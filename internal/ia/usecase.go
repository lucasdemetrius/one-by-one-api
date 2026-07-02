// Pacote: internal/ia
// Arquivo: usecase.go
// Descrição: Regras de negócio da IA por gestor (BYOK). Guarda/lê a config
//            (provedor + chave cifrada) e oferece `Completar` — a primitiva que
//            os recursos de IA (chat, overview, sugestões, PDI) usam para falar
//            com o provedor escolhido pelo gestor. A chave é decifrada só na hora.
// Autor: OneByOne API
// Criado em: 2026

package ia

import (
	"errors"
	"fmt"

	"onebyone-api/pkg/cripto"
)

// ErrSemConfig: o gestor ainda não configurou a IA no perfil.
var ErrSemConfig = errors.New("configure sua IA no perfil para usar este recurso")

// ErrProvedorInvalido: o provedor de IA informado não é um dos suportados.
// É erro de negócio (mensagem amigável), comparável via errors.Is no controller.
var ErrProvedorInvalido = errors.New("provedor de IA inválido")

// UseCase define as operações de IA.
type UseCase interface {
	// ObterConfig devolve o provedor e se há chave (nunca a chave em si).
	ObterConfig(usuarioID string) (ConfigIARespostaDTO, error)
	// SalvarConfig grava o provedor e (se enviada) cifra e guarda a chave.
	SalvarConfig(usuarioID string, dto SalvarConfigDTO) error
	// Chat responde uma pergunta livre do gestor.
	Chat(usuarioID, mensagem string) (string, error)
	// Completar é a primitiva usada pelos recursos de IA (sistema + prompt → texto).
	Completar(usuarioID, sistema, prompt string) (string, error)
}

type useCaseImpl struct {
	repo Repositorio
	// segredo do servidor usado para CIFRAR (e como 1ª tentativa ao decifrar) a chave de API.
	segredo string
	// segredoFallback: 2ª tentativa ao decifrar. Permite migrar do JWT_SECRET para um
	// segredo dedicado sem quebrar chaves já salvas (as antigas decifram pelo fallback;
	// ao serem regravadas passam a usar o segredo novo).
	segredoFallback string
}

// NovoUseCase cria o UseCase de IA. `segredo` é o usado para cifrar; `segredoFallback`
// (opcional) só é tentado ao decifrar valores antigos. Ambos devem ser estáveis.
func NovoUseCase(repo Repositorio, segredo, segredoFallback string) UseCase {
	return &useCaseImpl{repo: repo, segredo: segredo, segredoFallback: segredoFallback}
}

func (uc *useCaseImpl) ObterConfig(usuarioID string) (ConfigIARespostaDTO, error) {
	prov, chave, err := uc.repo.ObterConfig(usuarioID)
	if err != nil {
		return ConfigIARespostaDTO{}, err
	}
	dto := ConfigIARespostaDTO{TemChave: chave != nil && *chave != ""}
	if prov != nil {
		dto.Provedor = *prov
	}
	// Sem config própria completa? Verifica se está herdando a IA do RH.
	if !dto.TemChave {
		if _, _, herdada, e := uc.repo.ObterConfigEfetiva(usuarioID); e == nil {
			dto.HerdadaDoRH = herdada
		}
	}
	return dto, nil
}

func (uc *useCaseImpl) SalvarConfig(usuarioID string, dto SalvarConfigDTO) error {
	if !ProvedorValido(dto.Provedor) {
		return ErrProvedorInvalido
	}
	var chaveCifrada *string
	if dto.Chave != "" {
		cif, err := cripto.Cifrar(dto.Chave, uc.segredo)
		if err != nil {
			return fmt.Errorf("erro ao proteger a chave: %w", err)
		}
		chaveCifrada = &cif
	}
	return uc.repo.SalvarConfig(usuarioID, dto.Provedor, chaveCifrada)
}

func (uc *useCaseImpl) Completar(usuarioID, sistema, prompt string) (string, error) {
	// Config EM VIGOR: a própria do gestor, ou — se ele não tiver — a do RH dono dele.
	prov, chaveCif, _, err := uc.repo.ObterConfigEfetiva(usuarioID)
	if err != nil {
		return "", err
	}
	if prov == nil || *prov == "" || chaveCif == nil || *chaveCif == "" {
		return "", ErrSemConfig
	}
	chave, err := cripto.DecifrarComFallback(*chaveCif, uc.segredo, uc.segredoFallback)
	if err != nil {
		return "", fmt.Errorf("erro ao ler a chave de IA: %w", err)
	}
	return completar(*prov, chave, sistema, prompt)
}

func (uc *useCaseImpl) Chat(usuarioID, mensagem string) (string, error) {
	const sistema = "Você é o assistente do OneByOne, um app de reuniões 1:1 entre " +
		"gestor e liderado. Ajude o gestor a conduzir 1:1s melhores, desenvolver o time " +
		"e dar feedback. Responda em português do Brasil, de forma prática, gentil e objetiva."
	return uc.Completar(usuarioID, sistema, mensagem)
}
