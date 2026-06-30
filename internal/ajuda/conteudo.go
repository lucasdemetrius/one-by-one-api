// Pacote: internal/ajuda
// Arquivo: conteudo.go
// Descrição: Conteúdo CURADO da Central de Ajuda — tópicos (artigos), tour de boas-vindas
//            e a base de conhecimento que orienta o assistente de IA. Tudo aqui funciona
//            SEM IA (a IA é um extra). É a fonte única da verdade do "como usar o OneByOne",
//            então quando uma regra/rota mudar, atualize o tópico correspondente aqui.
// Autor: OneByOne API
// Criado em: 2026

package ajuda

// Papéis usados para filtrar a visibilidade dos tópicos/tour (mesmos valores do tb_usuarios).
const (
	papelGestor   = "LIDER"
	papelLiderado = "COLABORADOR"
	papelRH       = "RH"
	papelAdmin    = "ADMIN"
)

// Topico é um artigo da Central de Ajuda.
type Topico struct {
	ID        string   `json:"id"`
	Titulo    string   `json:"titulo"`
	Icone     string   `json:"icone"`     // emoji para o card
	Categoria string   `json:"categoria"` // agrupador na UI
	Papeis    []string `json:"papeis"`    // vazio = todos os papéis
	Resumo    string   `json:"resumo"`    // uma linha (para o card)
	Conteudo  string   `json:"conteudo"`  // markdown (corpo do artigo)
}

// PassoTour é uma etapa do tour de boas-vindas.
type PassoTour struct {
	Ordem  int    `json:"ordem"`
	Icone  string `json:"icone"`
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
}

// topicos é o catálogo curado. A ordem aqui é a ordem de exibição.
var topicos = []Topico{
	{
		ID: "primeiros-passos", Titulo: "Primeiros passos no OneByOne", Icone: "🚀",
		Categoria: "Começar", Papeis: nil,
		Resumo: "Entenda o que é o OneByOne e por onde começar.",
		Conteudo: "O **OneByOne** é onde gestor e liderado cuidam dos 1:1 (reuniões um a um) — " +
			"o ritual mais importante para desenvolver pessoas.\n\n" +
			"A hierarquia é: **RH → Gestor → Liderado**. O gestor organiza sua estrutura " +
			"(**Organização → Equipe → Colaborador**), define a **pauta**, conduz o 1:1 e registra o " +
			"que foi conversado.\n\n" +
			"**Por onde começar (gestor):** crie sua organização, uma equipe, adicione/convide um " +
			"liderado, escolha um template de pauta e agende o primeiro 1:1.",
	},
	{
		ID: "montar-estrutura", Titulo: "Organização, equipes e colaboradores", Icone: "🏗️",
		Categoria: "Estrutura", Papeis: []string{papelGestor, papelRH},
		Resumo: "Monte a árvore Organização → Equipe → Colaborador.",
		Conteudo: "1. **Organização** é o topo da sua estrutura (ex.: a empresa ou área).\n" +
			"2. Dentro dela, crie **Equipes** (times).\n" +
			"3. Em cada equipe, adicione **Colaboradores** (liderados).\n\n" +
			"O e-mail de cada liderado é único **por gestor**. Os dados são privados: cada gestor só " +
			"enxerga a própria estrutura.",
	},
	{
		ID: "convidar-liderado", Titulo: "Convidar um liderado", Icone: "✉️",
		Categoria: "Estrutura", Papeis: []string{papelGestor, papelRH},
		Resumo: "Envie o convite com link + código para o liderado criar o acesso.",
		Conteudo: "No colaborador, use **Convidar**. O liderado recebe um e-mail com um **link** e um " +
			"**código** de convite. Ao aceitar, ele cria a conta dele e passa a ver o próprio 1:1.\n\n" +
			"Se a pessoa troca de empresa, ao aceitar um novo convite ela perde o acesso ao 1:1 da " +
			"empresa anterior (o histórico continua com o gestor de lá).",
	},
	{
		ID: "templates-pauta", Titulo: "Templates de pauta", Icone: "📋",
		Categoria: "Pauta", Papeis: []string{papelGestor, papelRH},
		Resumo: "Crie modelos de pauta reutilizáveis com blocos.",
		Conteudo: "Um **template** é um modelo de pauta formado por **blocos** (texto, lista, imagem, " +
			"destaque). O template pode ser definido por colaborador, por equipe ou pela organização — " +
			"vale o mais específico (**colaborador → equipe → organização → padrão**).\n\n" +
			"Assim cada 1:1 já abre com a pauta certa, sem começar do zero.",
	},
	{
		ID: "conduzir-1a1", Titulo: "Conduzir um 1:1", Icone: "🤝",
		Categoria: "Reunião", Papeis: []string{papelGestor, papelRH, papelLiderado},
		Resumo: "Faça o 1:1 ao vivo, com pauta e tabuleiro colaborativo.",
		Conteudo: "No 1:1 **ao vivo**, gestor e liderado compartilham presença, cursores e um **tabuleiro** " +
			"colaborativo da pauta (em tempo real). É o momento de conversar sobre entregas, " +
			"desenvolvimento, feedbacks e combinados.\n\n" +
			"Ao final, **encerre o 1:1** para marcá-lo como **REALIZADO** — é isso que alimenta a saúde " +
			"do 1:1 e o streak.",
	},
	{
		ID: "registrar-1a1", Titulo: "Registrar o que foi conversado", Icone: "📝",
		Categoria: "Reunião", Papeis: []string{papelGestor},
		Resumo: "Guarde as respostas de cada bloco da pauta.",
		Conteudo: "Cada 1:1 gera um **registro** com as **respostas** preenchidas por bloco da pauta. " +
			"É o histórico estruturado da conversa — fica fácil retomar no próximo encontro e " +
			"acompanhar a evolução ao longo do tempo.",
	},
	{
		ID: "pdi", Titulo: "Plano de Desenvolvimento Individual (PDI)", Icone: "🎯",
		Categoria: "Desenvolvimento", Papeis: []string{papelGestor, papelRH, papelLiderado},
		Resumo: "Defina objetivos com prazos e acompanhe a conclusão.",
		Conteudo: "O **PDI** lista os objetivos de desenvolvimento do liderado, cada um com **prazo** e " +
			"status de conclusão. Use o 1:1 para revisar o PDI: o que avançou, o que travou e qual o " +
			"próximo passo.",
	},
	{
		ID: "nove-box", Titulo: "Matriz 9-box (desempenho × potencial)", Icone: "🟦",
		Categoria: "Desenvolvimento", Papeis: []string{papelGestor, papelRH},
		Resumo: "Posicione cada liderado na matriz de talentos.",
		Conteudo: "A **9-box** classifica o liderado em duas dimensões — **desempenho** e **potencial** — " +
			"resultando em 9 quadrantes. Ajuda a enxergar talentos, planejar sucessão e direcionar o " +
			"desenvolvimento de cada pessoa.",
	},
	{
		ID: "acompanhamento", Titulo: "Acompanhar humor, entregas e feedbacks", Icone: "💗",
		Categoria: "Desenvolvimento", Papeis: []string{papelGestor, papelRH},
		Resumo: "Tudo do liderado num lugar só, ao longo do tempo.",
		Conteudo: "No **acompanhamento** você registra **sentimento (humor)**, **entregas**, **feedbacks** " +
			"e **estudos** do liderado. Com o tempo, dá para ver a **tendência** de humor e o quanto a " +
			"pessoa está evoluindo — o que importa de verdade, mais do que a quantidade de reuniões.",
	},
	{
		ID: "agenda-lembretes", Titulo: "Agenda e lembretes de 1:1", Icone: "🗓️",
		Categoria: "Reunião", Papeis: []string{papelGestor},
		Resumo: "Agende com recorrência e receba lembretes por e-mail.",
		Conteudo: "Agende seus 1:1 com **recorrência** (semanal, quinzenal, mensal). O sistema envia " +
			"**lembretes por e-mail** e gera **notificações** no sino conforme a data se aproxima " +
			"(1 dia antes, no dia, 1 hora antes), respeitando suas preferências.",
	},
	{
		ID: "saude-1a1", Titulo: "Saúde do 1:1 e streak 🔥", Icone: "🔥",
		Categoria: "Indicadores", Papeis: []string{papelGestor, papelRH},
		Resumo: "Acompanhe sua cadência: em dia, atrasados e sequência.",
		Conteudo: "A **saúde do 1:1** mostra a sua cadência: quantos liderados estão **em dia**, quantos " +
			"**atrasados**, os **realizados** nos últimos 30 dias e a sua **sequência (streak)** de " +
			"constância. Encerrar o 1:1 (marcar como realizado) é o que mantém o streak aceso.",
	},
	{
		ID: "ia-assistente", Titulo: "Configurar sua IA (BYOK)", Icone: "🤖",
		Categoria: "Recursos", Papeis: []string{papelGestor, papelRH},
		Resumo: "Use sua própria chave de IA (Claude, OpenAI, DeepSeek ou Grok).",
		Conteudo: "No seu perfil, conecte uma **chave de IA própria** (BYOK) de um provedor suportado " +
			"(**Claude, OpenAI, DeepSeek ou Grok**). A chave é guardada **cifrada** e usada para os " +
			"recursos de IA do gestor. Um RH pode configurar a IA do tenant e os gestores herdam.\n\n" +
			"A **Central de Ajuda** também pode responder com IA (este assistente).",
	},
	{
		ID: "notificacoes", Titulo: "Notificações e preferências", Icone: "🔔",
		Categoria: "Recursos", Papeis: nil,
		Resumo: "Avisos no sino in-app e por e-mail, do seu jeito.",
		Conteudo: "O **sino** mostra avisos in-app (ex.: 1:1 chegando). Você ajusta nas **preferências** " +
			"o que quer receber e por qual canal. Os lembretes da agenda respeitam essas escolhas.",
	},
	{
		ID: "painel-rh", Titulo: "Visão consolidada do RH", Icone: "🏢",
		Categoria: "RH", Papeis: []string{papelRH},
		Resumo: "Cadastre gestores e acompanhe a evolução do time todo.",
		Conteudo: "Como **RH** você cadastra **gestores** e enxerga o tenant inteiro: a **agenda** " +
			"consolidada, a **9-box** de todos os liderados, o **acompanhamento** (evolução/humor) e os " +
			"KPIs de cada gestor. O foco é **qualidade e evolução**, não ranking de reuniões.",
	},
	{
		ID: "guia-liderado", Titulo: "Guia do liderado", Icone: "🌱",
		Categoria: "Começar", Papeis: []string{papelLiderado},
		Resumo: "Aceite o convite, prepare a pauta e participe do seu 1:1.",
		Conteudo: "Você foi convidado pelo seu gestor. Depois de **aceitar o convite** e criar o acesso, " +
			"você acompanha seus **1:1**, contribui na **pauta** (tabuleiro ao vivo), acompanha seu " +
			"**PDI** e o histórico das conversas. É o seu espaço de desenvolvimento.",
	},
	{
		ID: "recuperar-senha", Titulo: "Esqueci minha senha", Icone: "🔑",
		Categoria: "Conta", Papeis: nil,
		Resumo: "Receba um link + código para criar uma nova senha.",
		Conteudo: "Na tela de login, use **Esqueci minha senha** e informe seu e-mail. Se houver conta, " +
			"você recebe um **link** e um **código de segurança** (de 6 dígitos) por e-mail.\n\n" +
			"Abra o link, digite o código (um dígito por campo) e defina a nova senha. Por segurança, " +
			"**o link e o código valem por 15 minutos** e só podem ser usados uma vez.",
	},
}

// tourGestor é o tour de boas-vindas para o gestor (o caminho feliz completo).
var tourGestor = []PassoTour{
	{1, "🏗️", "Monte sua estrutura", "Crie sua Organização, uma Equipe e adicione seus liderados."},
	{2, "✉️", "Convide os liderados", "Envie o convite por e-mail (link + código) para cada pessoa criar o acesso."},
	{3, "📋", "Defina a pauta", "Escolha ou crie um template de pauta com os blocos que importam."},
	{4, "🗓️", "Agende o 1:1", "Marque o primeiro encontro, com recorrência se quiser. Você recebe lembretes."},
	{5, "🤝", "Conduza o 1:1", "Faça a reunião ao vivo, registre o que foi conversado e encerre como realizado."},
	{6, "🔥", "Acompanhe a evolução", "Veja a saúde do 1:1, o streak, o PDI, a 9-box e o humor do time."},
}

// tourRH é o tour para o RH.
var tourRH = []PassoTour{
	{1, "🏢", "Cadastre seus gestores", "Crie as contas dos gestores do seu tenant."},
	{2, "📊", "Acompanhe o consolidado", "Veja a agenda, a 9-box e a evolução dos liderados de cada gestor."},
	{3, "🤖", "Configure a IA do tenant", "Conecte uma chave de IA — os gestores herdam para os recursos de IA."},
	{4, "💗", "Foque na evolução", "Priorize qualidade e desenvolvimento das pessoas, não a quantidade de reuniões."},
}

// tourLiderado é o tour para o liderado.
var tourLiderado = []PassoTour{
	{1, "✅", "Aceite o convite", "Use o link e o código do e-mail para criar seu acesso."},
	{2, "📋", "Prepare a pauta", "Contribua nos pontos que você quer levar para o 1:1."},
	{3, "🤝", "Participe do 1:1", "Converse com seu gestor sobre entregas, feedbacks e desenvolvimento."},
	{4, "🎯", "Acompanhe seu PDI", "Veja seus objetivos de desenvolvimento e o histórico das conversas."},
}

// topicosVisiveis devolve os tópicos visíveis para um papel. ADMIN e papéis desconhecidos
// veem todos (a ajuda nunca esconde conteúdo desnecessariamente).
func topicosVisiveis(role string) []Topico {
	out := make([]Topico, 0, len(topicos))
	for _, t := range topicos {
		if topicoVisivelPara(t, role) {
			out = append(out, t)
		}
	}
	return out
}

// topicoVisivelPara diz se um tópico aparece para o papel informado.
func topicoVisivelPara(t Topico, role string) bool {
	if len(t.Papeis) == 0 || role == papelAdmin || role == "" {
		return true
	}
	for _, p := range t.Papeis {
		if p == role {
			return true
		}
	}
	return false
}

// acharTopico localiza um tópico pelo id.
func acharTopico(id string) (Topico, bool) {
	for _, t := range topicos {
		if t.ID == id {
			return t, true
		}
	}
	return Topico{}, false
}

// tourPara devolve o tour adequado ao papel (gestor é o padrão).
func tourPara(role string) []PassoTour {
	switch role {
	case papelRH:
		return tourRH
	case papelLiderado:
		return tourLiderado
	default:
		return tourGestor
	}
}

// baseConhecimento é a instrução de sistema do assistente de IA: descreve o OneByOne com
// precisão para que as respostas sejam corretas, específicas do produto e no tom da marca.
// Atualize junto com os tópicos quando o produto mudar.
const baseConhecimento = `Você é o assistente da Central de Ajuda do OneByOne — um aplicativo de reuniões 1:1 (um a um) entre gestor e liderado, cujo objetivo é desenvolver pessoas.

CONHECIMENTO DO PRODUTO (use para responder com precisão; não invente recursos que não existem):
- Hierarquia: RH → Gestor → Liderado. O RH é o topo do tenant (cadastra gestores e vê o consolidado). O Gestor (papel LIDER) conduz os 1:1. O Liderado (papel COLABORADOR) participa e acompanha o próprio desenvolvimento.
- Estrutura do gestor: Organização → Equipe → Colaborador (liderado). Cada gestor só enxerga a própria estrutura (dados privados e isolados).
- Convite: o gestor convida o liderado por e-mail (link + código); o liderado aceita e cria o acesso.
- Pauta: templates de pauta formados por blocos (texto, lista, imagem, destaque). O template vale por prioridade: colaborador → equipe → organização → padrão.
- Reunião: 1:1 ao vivo com presença, cursores e um tabuleiro colaborativo em tempo real. Ao final, encerrar marca o 1:1 como REALIZADO. Cada 1:1 gera um registro com respostas por bloco.
- Desenvolvimento: PDI (objetivos com prazo), matriz 9-box (desempenho × potencial), acompanhamento (humor/sentimento, entregas, feedbacks, estudos).
- Agenda: 1:1 com recorrência (semanal/quinzenal/mensal) + lembretes por e-mail e notificações no sino (1 dia antes, no dia, 1 hora antes), conforme as preferências.
- Saúde do 1:1: cadência (em dia, atrasados, realizados em 30 dias) e streak (sequência de constância).
- IA: cada gestor pode conectar a própria chave (BYOK) de Claude, OpenAI, DeepSeek ou Grok; o RH pode configurar a IA do tenant e os gestores herdam.
- Senha: "esqueci minha senha" envia link + código de 6 dígitos, válidos por 15 minutos e de uso único.

COMO RESPONDER:
- Sempre em português do Brasil, de forma prática, gentil e objetiva. Prefira passos curtos e numerados quando fizer sentido.
- Foque em ajudar a pessoa a USAR o OneByOne e a fazer 1:1 melhores. Pode dar boas práticas de gestão de pessoas quando ajudar.
- Se a pergunta for sobre algo que o produto não faz, diga isso com honestidade e sugira a alternativa mais próxima dentro do app.
- Não fale de configurações técnicas internas, segredos, chaves ou detalhes de implementação. Se for um problema de conta/acesso que você não resolve, oriente a procurar o gestor ou o suporte.`
