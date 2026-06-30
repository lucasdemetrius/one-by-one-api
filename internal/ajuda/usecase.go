// Pacote: internal/ajuda
// Arquivo: usecase.go
// Descrição: Regras da Central de Ajuda. Serve o conteúdo curado (tópicos/tour) e o
//            assistente de IA. A IA resolve a chave em cascata: PLATAFORMA (chave do .env,
//            vale para todos) → BYOK (chave do próprio gestor/RH) → indisponível (cai no
//            conteúdo curado). Assim a ajuda básica funciona SEMPRE, com ou sem IA.
// Autor: OneByOne API
// Criado em: 2026

package ajuda

import (
	"errors"
	"strings"

	"onebyone-api/internal/ia"
)

// ErrTopicoNaoEncontrado: id de tópico inexistente.
var ErrTopicoNaoEncontrado = errors.New("tópico não encontrado")

// mensagemSemIA é a resposta amigável quando não há IA disponível (sem chave de plataforma
// nem BYOK). Orienta a pessoa ao conteúdo curado em vez de devolver um erro seco.
const mensagemSemIA = "O assistente de IA ainda não está disponível para a sua conta. " +
	"Enquanto isso, dê uma olhada nos tópicos da Central de Ajuda — eles cobrem o passo a passo " +
	"do OneByOne. Se precisar, fale com o seu gestor ou com o suporte. 💜"

// UseCase define as operações da Central de Ajuda.
type UseCase interface {
	// ListarTopicos devolve os tópicos visíveis para o papel informado.
	ListarTopicos(role string) TopicosDTO
	// ObterTopico devolve um tópico pelo id (independente de papel).
	ObterTopico(id string) (Topico, error)
	// Tour devolve as etapas do tour de boas-vindas adequadas ao papel.
	Tour(role string) TourDTO
	// Perguntar responde uma pergunta livre usando IA (plataforma → BYOK → indisponível).
	Perguntar(usuarioID, role, pergunta string) RespostaIADTO
	// IADisponivelPara diz se há IA utilizável (plataforma ou BYOK) para o usuário.
	IADisponivelPara(usuarioID string) bool
}

type useCaseImpl struct {
	iaUC ia.UseCase
	// Chave de IA da PLATAFORMA (opcional). Quando preenchida, atende todos os usuários.
	plataformaProvedor string
	plataformaChave    string
}

// NovoUseCase cria o UseCase da Ajuda. `iaUC` é usado para o fallback BYOK; o par
// plataforma(provedor, chave) vem do .env e, quando presente, atende qualquer usuário.
func NovoUseCase(iaUC ia.UseCase, plataformaProvedor, plataformaChave string) UseCase {
	return &useCaseImpl{
		iaUC:               iaUC,
		plataformaProvedor: plataformaProvedor,
		plataformaChave:    plataformaChave,
	}
}

func (uc *useCaseImpl) ListarTopicos(role string) TopicosDTO {
	return TopicosDTO{Itens: topicosVisiveis(role)}
}

func (uc *useCaseImpl) ObterTopico(id string) (Topico, error) {
	t, ok := acharTopico(id)
	if !ok {
		return Topico{}, ErrTopicoNaoEncontrado
	}
	return t, nil
}

func (uc *useCaseImpl) Tour(role string) TourDTO {
	return TourDTO{Passos: tourPara(role)}
}

// temChavePlataforma indica se a IA de plataforma está configurada no .env.
func (uc *useCaseImpl) temChavePlataforma() bool {
	return uc.plataformaProvedor != "" && uc.plataformaChave != ""
}

func (uc *useCaseImpl) IADisponivelPara(usuarioID string) bool {
	if uc.temChavePlataforma() {
		return true
	}
	// Sem chave de plataforma: só há IA se o usuário (ou o RH dele) tiver BYOK configurada.
	// Detectamos isso tentando uma resolução barata via ObterConfig do módulo ia.
	cfg, err := uc.iaUC.ObterConfig(usuarioID)
	return err == nil && (cfg.TemChave || cfg.HerdadaDoRH)
}

// Perguntar responde com IA, escolhendo a chave em cascata. Nunca devolve erro técnico ao
// chamador: em qualquer falha de IA cai na mensagem amigável (fonte "indisponivel").
func (uc *useCaseImpl) Perguntar(usuarioID, role, pergunta string) RespostaIADTO {
	pergunta = strings.TrimSpace(pergunta)
	if pergunta == "" {
		return RespostaIADTO{Resposta: mensagemSemIA, Fonte: "indisponivel", IADisponivel: false}
	}
	sistema := baseConhecimento + contextoDePapel(role)

	// 1) Chave de PLATAFORMA (atende todos).
	if uc.temChavePlataforma() {
		if resp, err := ia.CompletarComChave(uc.plataformaProvedor, uc.plataformaChave, sistema, pergunta); err == nil {
			return RespostaIADTO{Resposta: resp, Fonte: "plataforma", IADisponivel: true}
		}
		// Se a plataforma falhar (ex.: cota/erro do provedor), ainda tentamos o BYOK abaixo.
	}

	// 2) BYOK do próprio usuário (ou herdada do RH).
	resp, err := uc.iaUC.Completar(usuarioID, sistema, pergunta)
	if err == nil {
		return RespostaIADTO{Resposta: resp, Fonte: "byok", IADisponivel: true}
	}

	// 3) Sem IA utilizável → mensagem amigável (o conteúdo curado segue disponível).
	return RespostaIADTO{Resposta: mensagemSemIA, Fonte: "indisponivel", IADisponivel: false}
}

// contextoDePapel adiciona ao prompt de sistema quem está perguntando, para a IA ajustar a
// resposta (um liderado e um RH têm dúvidas diferentes).
func contextoDePapel(role string) string {
	switch role {
	case papelGestor:
		return "\n\nO usuário que está perguntando é um GESTOR (conduz os 1:1 do time)."
	case papelLiderado:
		return "\n\nO usuário que está perguntando é um LIDERADO (participa dos 1:1 com o gestor dele)."
	case papelRH:
		return "\n\nO usuário que está perguntando é do RH (vê o consolidado e cadastra gestores)."
	case papelAdmin:
		return "\n\nO usuário que está perguntando é um ADMINISTRADOR da plataforma."
	default:
		return ""
	}
}
