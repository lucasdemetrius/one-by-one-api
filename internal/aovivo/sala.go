// Pacote: internal/aovivo
// Arquivo: sala.go
// Descrição: Infraestrutura de tempo real (WebSocket) do 1:1 ao vivo. Um Hub
//            gerencia várias Salas (uma por 1:1, identificada pelo colaborador).
//            Cada Sala mantém os participantes conectados, retransmite cursores
//            e movimentos do tabuleiro, e guarda o último estado para quem entra.
// Autor: OneByOne API
// Criado em: 2025

package aovivo

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Participante são os dados públicos de quem está na sala.
type Participante struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Papel string `json:"papel"`
	Cor   string `json:"cor"`
}

// Cliente é uma conexão WebSocket dentro de uma sala.
type Cliente struct {
	ID    string
	Nome  string
	Papel string
	Cor   string
	conn  *websocket.Conn
	envio chan []byte
	sala  *Sala
}

// Sala é uma sessão de 1:1 ao vivo.
type Sala struct {
	ID        string
	mu        sync.Mutex
	clientes  map[*Cliente]bool
	estado    []byte // último {tipo:"tabuleiro",...} para enviar a quem entra
	encerrado []byte // se o 1:1 foi encerrado, guarda o sinal p/ quem entrar depois
}

// Hub gerencia todas as salas ativas.
type Hub struct {
	mu    sync.Mutex
	salas map[string]*Sala
}

// NovoHub cria um hub vazio.
func NovoHub() *Hub {
	return &Hub{salas: make(map[string]*Sala)}
}

// obterSala devolve (criando se preciso) a sala com o id informado.
func (h *Hub) obterSala(id string) *Sala {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.salas[id]
	if !ok {
		s = &Sala{ID: id, clientes: make(map[*Cliente]bool)}
		h.salas[id] = s
	}
	return s
}

// removerSeVazia descarta a sala quando não há mais ninguém.
func (h *Hub) removerSeVazia(s *Sala) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s.mu.Lock()
	vazia := len(s.clientes) == 0
	s.mu.Unlock()
	if vazia {
		delete(h.salas, s.ID)
	}
}

func (s *Sala) entrar(c *Cliente) {
	s.mu.Lock()
	s.clientes[c] = true
	estado := s.estado
	encerrado := s.encerrado
	s.mu.Unlock()

	// Envia o estado atual do tabuleiro ao recém-chegado (se já existir).
	if estado != nil {
		select {
		case c.envio <- estado:
		default:
		}
	}
	// Se o 1:1 já foi encerrado, avisa quem acabou de entrar (modo consulta).
	if encerrado != nil {
		select {
		case c.envio <- encerrado:
		default:
		}
	}
	// Avisa todos da presença atualizada.
	s.transmitirPresenca()
}

func (s *Sala) sair(c *Cliente) {
	s.mu.Lock()
	if _, ok := s.clientes[c]; !ok {
		s.mu.Unlock()
		return
	}
	delete(s.clientes, c)
	close(c.envio)
	s.mu.Unlock()
	s.transmitirPresenca()
}

func (s *Sala) participantes() []Participante {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps := make([]Participante, 0, len(s.clientes))
	for c := range s.clientes {
		ps = append(ps, Participante{ID: c.ID, Nome: c.Nome, Papel: c.Papel, Cor: c.Cor})
	}
	return ps
}

// transmitir envia uma mensagem a todos os clientes (menos `exceto`, se informado).
func (s *Sala) transmitir(msg []byte, exceto *Cliente) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.clientes {
		if c == exceto {
			continue
		}
		// Envio não-bloqueante: se o buffer do cliente está cheio, descarta.
		select {
		case c.envio <- msg:
		default:
		}
	}
}

func (s *Sala) transmitirPresenca() {
	msg, _ := json.Marshal(map[string]any{
		"tipo":          "presenca",
		"participantes": s.participantes(),
	})
	s.transmitir(msg, nil)
}

// receber processa uma mensagem vinda de um cliente.
func (s *Sala) receber(remetente *Cliente, msg []byte) {
	var base struct {
		Tipo string `json:"tipo"`
	}
	if err := json.Unmarshal(msg, &base); err != nil {
		return
	}

	switch base.Tipo {
	case "cursor":
		// Injeta quem enviou e retransmite aos outros.
		var c struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		}
		_ = json.Unmarshal(msg, &c)
		out, _ := json.Marshal(map[string]any{
			"tipo": "cursor", "de": remetente.ID, "x": c.X, "y": c.Y,
		})
		s.transmitir(out, remetente)

	case "tabuleiro":
		// Guarda o estado (para quem entrar depois) e retransmite aos outros.
		s.mu.Lock()
		s.estado = msg
		s.mu.Unlock()
		s.transmitir(msg, remetente)

	case "tema-atualizado":
		// Sinal transitório: alguém adicionou/removeu um bloco de conteúdo de um
		// tema. Apenas retransmite aos outros para que recarreguem os blocos
		// daquele tema (o conteúdo em si vem do banco via REST, não daqui).
		// Não guardamos estado: é só um "avise os outros que mudou".
		s.transmitir(msg, remetente)

	case "apresentacao":
		// Sinal transitório: alguém entrou/saiu do MODO APRESENTAÇÃO de um tema
		// ({tipo:"apresentacao", tema, ativo}). Retransmite para o outro abrir/
		// fechar a mesma apresentação ao vivo. O conteúdo vem do banco via REST.
		s.transmitir(msg, remetente)

	case "encerrado":
		// O gestor ENCERROU o 1:1 ({tipo:"encerrado", resumo}). Guarda o sinal
		// (para quem entrar depois cair direto no modo consulta) e avisa TODOS —
		// inclusive quem enviou — para a tela virar somente-leitura nos dois lados.
		s.mu.Lock()
		s.encerrado = msg
		s.mu.Unlock()
		s.transmitir(msg, nil)
	}
}
