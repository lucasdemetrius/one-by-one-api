# OneByOne — Catálogo completo de funcionalidades, regras e validações

> Documento companheiro do [CLAUDE.md](../CLAUDE.md) (seção 12) e **referência única**:
> todas as rotas, regras de negócio e validações de cada módulo, **geradas a partir do
> código**. Os `README.md` de cada módulo trazem o mesmo, focado em um módulo só — **ao
> mudar regra/rota/validação, atualize aqui e no README.** Acesso: salvo indicação, toda
> rota exige JWT e o dono do recurso é o **LÍDER** (ver CLAUDE.md §7.1 — posse/IDOR).

**Índice desta seção:** 12.1 Usuário · 12.2 Organização · 12.3 Equipe · 12.4 Colaborador (liderado) · 12.5 Convite · 12.6 Template · 12.7 Template-bloco · 12.8 OneByOne (a reunião) · 12.8b Saúde do 1:1 + Streak · 12.9 Registro do 1:1 · 12.10 Valor de registro (respostas) · 12.11 1:1 ao vivo (WebSocket) · 12.12 Tabuleiro da pauta · 12.13 Bloco de tema · 12.14 Classificação 9-box · 12.15 PDI — Plano de Desenvolvimento Individual · 12.16 Acompanhamento (sentimento/entregas/feedbacks/estudos) · 12.17 Agendamento · 12.18 Notificação (sino + cron) · 12.19 IA plugável (BYOK) · 12.20 Auditoria · 12.21 Infraestrutura (pkg/) e fiação (cmd/api/) · 12.22 Frontend — telas e fluxos

### Identidade & acesso

#### 12.1 Usuário (`usuario`)

Gerencia contas de usuário (cadastro, autenticação JWT, perfil, foto em S3)

**Tabela(s):** tb_usuarios: id (UUID PK), nome (VARCHAR 100), email (VARCHAR 150 UNIQUE), password (VARCHAR 255 hash bcrypt), role (ENUM LIDER/COLABORADOR, default COLABORADOR), foto_key (VARCHAR 500 NULL), criado_em (DATETIME), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL soft delete), deletado_por (VARCHAR 36 UUID NULL)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /auth/login` | Autentica um usuário com e-mail e senha; retorna token JWT + dados do usuário | público |
| `POST /auth/registrar` | Auto-cadastro público; cria novo usuário com role padrão COLABORADOR | público |
| `POST /usuarios` | Cria novo usuário (autenticado); permite definir role inclusive LIDER | JWT |
| `GET /usuarios` | Lista todos os usuários ativos, ordenados por nome | JWT |
| `GET /usuarios/:id` | Busca um usuário ativo pelo UUID | JWT |
| `PUT /usuarios/:id` | Atualiza parcialmente um usuário (só os campos enviados); sem alteração de senha | JWT |
| `DELETE /usuarios/:id` | Exclusão lógica (soft delete); registra quem deletou | JWT |
| `POST /usuarios/:id/foto` | Upload de foto de perfil para S3; retorna DTO com URL presignada temporária (~2h) | JWT |

**Regras de negócio:**

- Unicidade de e-mail na criação: verifica BuscarPorEmail antes de Criar
- Unicidade de e-mail na atualização: se muda email, valida que não pertence a outro usuário ativo
- Hash de senha bcrypt com custo 12 (GenerateFromPassword): nunca armazenada em texto puro
- Role padrão COLABORADOR: se não informado na criação, assume COLABORADOR
- UUID gerado no servidor: uuid.New().String() dentro do UseCase, não vem do cliente
- Atualização parcial: carrega estado atual, só sobrescreve campos não-vazios do DTO
- Senha não é alterável via PUT /usuarios/:id
- Soft delete (exclusão lógica): preenche deletado_em + deletado_por em vez de apagar linha
- Todas as queries de leitura filtram deletado_em IS NULL
- E-mail de usuário deletado pode ser reutilizado (filtro IS NULL no Buscar)
- Quem deletou é o usuário autenticado: controller passa JWT.UsuarioID como deletadoPor
- Login com mensagem genérica: tanto e-mail inexistente quanto senha errada retornam 'credenciais inválidas'
- JWT com expiração: cfg.JWTExpiracaoHoras, assinado HS256 com cfg.JWTSecret
- Claims JWT: UsuarioID + Role + RegisteredClaims (ExpiresAt, IssuedAt)
- Upload de foto: valida user existe, deriva extensão de Content-Type, monta chave 'usuarios/{id}/foto.{ext}', envia S3, salva chave no banco
- E-mail de boas-vindas assíncrono: enviado só para role LIDER ao criar, por email.Servico (pode estar dormente)
- URL presignada de foto: gerada dinamicamente via storage.ExpiracaoURLFoto (~2h); retorna nil silenciosamente em erro
- Mapper protege dados: ParaRespostaDTO omite password, deletado_em, deletado_por

**Validações:**

- CriarUsuarioDTO.nome: required, min=2, max=100
- CriarUsuarioDTO.email: required, email (RFC 5322), max=150
- CriarUsuarioDTO.password: required, min=6, max=100
- CriarUsuarioDTO.role: omitempty, oneof=LIDER COLABORADOR (opcional)
- AtualizarUsuarioDTO.nome: omitempty, min=2, max=100
- AtualizarUsuarioDTO.email: omitempty, email, max=150
- AtualizarUsuarioDTO.role: omitempty, oneof=LIDER COLABORADOR
- LoginDTO.email: required, email
- LoginDTO.password: required
- UploadFoto: Content-Type validado em tiposImagemPermitidos (image/jpeg, image/png, image/webp)
- UploadFoto: tamanho máximo 5MB (http.MaxBytesReader)
- Binding validação automática pelo Gin ShouldBindJSON: falha retorna 400

### Estrutura organizacional

#### 12.2 Organização (`organizacao`)

Gerencia organizações (empresas/áreas) do líder para agrupar equipes e colaboradores antes de conduzir reuniões 1:1.

**Tabela(s):** tb_organizacoes: id (VARCHAR 36, PK, UUID), usuario_id (VARCHAR 36, FK tb_usuarios, líder dono), template_id (VARCHAR 36, FK tb_template, anulável), nome (VARCHAR 100), criado_em (DATETIME default CURRENT_TIMESTAMP), alterado_em (DATETIME anulável), deletado_em (DATETIME anulável, soft delete), deletado_por (VARCHAR 36 anulável, quem deletou), foto_key (VARCHAR 500 anulável, chave S3)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /organizacoes` | Cria nova organização. Gera UUID v4, vincula ao líder autenticado (usuario_id do JWT), aceita template_id opcional. | Líder (JWT obrigatório) |
| `GET /organizacoes` | Lista todas as organizações ativas do líder autenticado, ordenadas por nome ASC. | Líder (JWT obrigatório) |
| `GET /organizacoes/:id` | Busca uma organização ativa pelo UUID. Retorna 404 se não existir ou estiver deletada. | Líder (JWT obrigatório) |
| `PUT /organizacoes/:id` | Atualiza parcialmente (nome e/ou template_id). Busca atual primeiro; se não existir, retorna 404. Campos omitidos são preservados. | Líder (JWT obrigatório) |
| `DELETE /organizacoes/:id` | Soft delete (exclusão lógica): preenche deletado_em e deletado_por. Valida existência antes de deletar. | Líder (JWT obrigatório) |
| `POST /organizacoes/:id/foto` | Upload de foto (multipart/form-data, campo 'foto'). Limita a 5MB, aceita JPEG/PNG/WebP. Envia ao S3, persiste chave, retorna organização com foto_url presignada (~2h). | Líder (JWT obrigatório) |

**Regras de negócio:**

- Geração de identidade: UUID v4 gerado no servidor (google/uuid) durante Criar; cliente não envia id.
- Vínculo ao líder: organização amarrada a usuario_id extraído do JWT no controller; cada líder cria organizações apenas para si (Cadeia A de posse).
- criado_em no servidor: timestamp definido com time.Now() no Criar.
- Atualização parcial (PATCH-like): PUT busca atual; Nome é atualizado se não vier vazio (""); TemplateID é atualizado apenas se ponteiro não for nil (enviar null preserva valor atual, sem forma de remover template).
- Verificação de existência antes de mutar: Atualizar, Deletar e UploadFoto chamam BuscarPorId primeiro; erro se não encontrada ou deletada.
- Soft delete: Deletar nunca remove linha; preenche deletado_em e deletado_por; todas as consultas filtram deletado_em IS NULL.
- alterado_em automático: repository define alterado_em = time.Now() em toda UPDATE (incluindo UploadFoto).
- Upload S3: arquivo limitado a 5MB, tipos JPEG/PNG/WebP validados no controller (mapa tiposImagemPermitidos). Chave segue padrão organizacoes/{id}/foto{ext}. Fluxo: upload S3 → persiste chave em foto_key → recarrega organização → retorna DTO com foto_url presignada.
- URL presignada temporária: gerada sob demanda via GerarURLPresignada(storage.ExpiracaoURLFoto ~2h); nunca exposta como link público. Se falhar geração ou armazenamento indisponível, campo retorna null (não derruba resposta).

**Validações:**

- CriarOrganizacaoDTO.Nome: binding required,min=2,max=100.
- CriarOrganizacaoDTO.TemplateID: binding omitempty (opcional).
- AtualizarOrganizacaoDTO.Nome: binding omitempty,min=2,max=100 (só validado se informado).
- AtualizarOrganizacaoDTO.TemplateID: binding omitempty (opcional).
- UploadFoto: validação Content-Type no controller contra mapa tiposImagemPermitidos (image/jpeg, image/png, image/webp); rejeita outros com erro 400.
- UploadFoto: limitador http.MaxBytesReader configurado para 5MB; arquivo maior retorna erro 400.
- UploadFoto: validação de existência da organização antes de upload (BuscarPorId).
- Todas as operações de leitura filtram deletado_em IS NULL (soft delete).
- UPDATE DELETE/Atualizar/UploadFoto executam com WHERE deletado_em IS NULL e checam RowsAffected para garantir posse/existência.

#### 12.3 Equipe (`equipe`)

Gerencia equipes (subgrupos de colaboradores) dentro de organizações: CRUD, upload de foto para S3 com URL presignada, herança de template, soft delete com auditoria de quem deletou.

**Tabela(s):** tb_equipes: id (VARCHAR 36, PK UUID), usuario_id (VARCHAR 36 FK→tb_usuarios, NOT NULL, líder dono), organizacao_id (VARCHAR 36 FK→tb_organizacoes, NOT NULL), template_id (VARCHAR 36 FK→tb_template, NULL, template padrão da equipe nível 2), nome (VARCHAR 100, NOT NULL), foto_key (VARCHAR 500, NULL, chave S3), criado_em (DATETIME, NOT NULL DEFAULT CURRENT_TIMESTAMP), alterado_em (DATETIME, NULL), deletado_em (DATETIME, NULL, soft delete), deletado_por (VARCHAR 36 FK→tb_usuarios, NULL)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /equipes` | Cria uma nova equipe vinculada ao líder autenticado, gerando UUID e persistindo em tb_equipes. Aceita organizacao_id, nome (2-100 chars) e template_id opcional. | LIDER (JWT obrigatório) |
| `GET /equipes` | Lista todas as equipes ativas (deletado_em IS NULL) do líder autenticado, ordenadas por nome. | LIDER (JWT obrigatório) |
| `GET /equipes/:id` | Busca uma equipe ativa pelo UUID, retornando dados + URL presignada de foto (2h de validade) se existir. | LIDER (JWT obrigatório) |
| `PUT /equipes/:id` | Atualização parcial: modifica nome e/ou template_id preservando demais campos. Nome é validado pela unicidade escopada ao líder (case-insensitive, trim). | LIDER (JWT obrigatório) |
| `DELETE /equipes/:id` | Soft delete: preenche deletado_em (timestamp) e deletado_por (usuarioID de quem deletou) sem remover registro fisicamente. | LIDER (JWT obrigatório) |
| `POST /equipes/:id/foto` | Upload de foto da equipe (multipart/form-data, campo 'foto'): aceita JPEG/PNG/WebP, máx 5MB. Envia ao S3, persiste chave em foto_key, retorna URL presignada. | LIDER (JWT obrigatório) |
| `GET /organizacoes/:id/equipes` | Lista todas as equipes ativas de uma organização específica, ordenadas por nome. | LIDER (JWT obrigatório) |

**Regras de negócio:**

- Soft delete: registros nunca são removidos fisicamente; deletado_em e deletado_por são preenchidos; todas as queries de leitura filtram deletado_em IS NULL
- Unicidade de nome escopada ao líder: o líder não pode ter duas equipes ATIVAS com o mesmo nome (normalizado: case-insensitive com LOWER/TRIM); excetoID exclui a própria equipe no UPDATE
- Herança de template (nível 2): equipe.template_id, quando preenchido, sobrescreve template da organização; quando nulo, segue template da organização
- Geração de UUID: Equipe.ID é UUID v4 gerado na aplicação (google/uuid); CriadoEm recebe timestamp atual
- Vínculo imutável ao líder: usuario_id da equipe é preenchido automaticamente com o JWT na criação; DTO não permite sobrescrever
- Validação de existência antes de modificar/excluir: Atualizar, Deletar e UploadFoto chamam BuscarPorId; se não existir ou estiver deletada, retornam erro (404)
- Atualização parcial preserva campos: se Nome ou TemplateID não virem no DTO, os campos atuais são mantidos
- Upload de foto: extensão derivada do Content-Type (image/jpeg→.jpg, image/png→.png, image/webp→.webp); chave formatada equipes/{id}/foto{ext}; URL presignada gerada dinamicamente com validade 2h
- Sincronização de alterado_em: UPDATE set alterado_em = NOW() ao atualizar nome/template ou foto
- Ordenação padrão: listagens (ListarPorUsuario, ListarPorOrganizacao) retornam equipes ORDER BY nome ASC

**Validações:**

- CriarEquipeDTO: organizacao_id (required), nome (required, min=2, max=100), template_id (*string, omitempty)
- AtualizarEquipeDTO: nome (omitempty, min=2, max=100 se preenchido), template_id (*string, omitempty)
- Upload de foto: Content-Type validado (apenas image/jpeg, image/png, image/webp aceitos); tamanho máximo 5 MB (http.MaxBytesReader limita corpo); erro de tamanho disparado durante leitura, não antecipadamente
- Nome: espaços nas pontas (trim) removidos ao perseverar; normalização (LOWER+TRIM) usada na checagem de unicidade para colisão case-insensitive
- Erro de conflito (409): mapeado pela string fixa 'já existe uma equipe com este nome' comparada no Controller
- Recurso não encontrado (404): retornado quando equipe não existe ou está deletada (deletado_em IS NOT NULL); não revela existência (obedece padrão POSSE: recurso alheio → 404)

#### 12.4 Colaborador (liderado) (`colaborador`)

Gerencia colaboradores (liderados) de uma organização: CRUD, foto em S3, desligamento/reativação, com posse baseada em Cadeia B (equipe/organização dono).

**Tabela(s):** tb_colaboradores: id (VARCHAR 36 PK), usuario_id (VARCHAR 36 NULL FK -> tb_usuarios), organizacao_id (VARCHAR 36 NOT NULL FK -> tb_organizacoes), equipe_id (VARCHAR 36 NOT NULL FK -> tb_equipes), template_id (VARCHAR 36 NULL FK -> tb_template), nome (VARCHAR 100), email (VARCHAR 150), whatsapp (VARCHAR 20 NULL), data_nascimento (DATE NULL), foto_key (VARCHAR 500 NULL), criado_em (DATETIME), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL soft delete), deletado_por (VARCHAR 36 NULL), desligado_em (DATETIME NULL inativação)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /colaboradores` | Cria novo colaborador na equipe/organização; valida posse (equipe e org do líder logado), unicidade de e-mail no líder, rejeita e-mail do próprio gestor (anti-sequestro), gera UUID | LIDER (middleware.ApenasLider + posse no usecase) |
| `POST /importar-liderados` | Import em lote (CSV): cria vários liderados numa equipe. Corpo `{organizacao_id, equipe_id, itens:[{nome,email}]}` (máx. 500). Valida cada linha (nome/e-mail) e reusa Criar (posse + e-mail único + anti-gestor); retorna `{criados, erros:[{linha,nome,email,motivo}]}` sem abortar o lote. Rota top-level (evita colisão estático×:id no Gin). | LIDER (ApenasLider + posse da equipe/org) |
| `GET /colaboradores/:id` | Busca colaborador ativo por ID; permite líder dono OU o próprio liderado (self via usuario_id) | LIDER dono OU COLABORADOR (self) |
| `PUT /colaboradores/:id` | Atualiza campos (nome, email, equipe_id, template_id, whatsapp, data_nascimento) com validação de posse, unicidade de e-mail no líder, rejeita e-mail do gestor; usuario_id IGNORADO propositalmente (segurança) | LIDER (posse checada no usecase) |
| `DELETE /colaboradores/:id` | Soft delete: preenche deletado_em e deletado_por (ID do líder logado); preserva registro | LIDER (posse checada) |
| `POST /colaboradores/:id/foto` | Upload de foto (multipart/form-data, campo 'foto'); valida tipo (JPEG/PNG/WebP), tamanho máx 5MB; envia para S3, salva foto_key, retorna URL presignada | LIDER dono OU COLABORADOR (self) |
| `POST /colaboradores/:id/desligar` | Marca colaborador como inativo (desligado_em); data opcional no corpo (formato YYYY-MM-DD, default hoje ao meio-dia); preserva registro para histórico | LIDER (posse checada) |
| `POST /colaboradores/:id/reativar` | Limpa desligado_em, reativando o colaborador | LIDER (posse checada) |
| `GET /equipes/:id/colaboradores` | Lista colaboradores ativos de uma equipe (rota aninhada); posse: equipe do líder logado; retorna lista ordenada por nome ASC | LIDER (posse de equipe checada) |
| `GET /organizacoes/:id/colaboradores` | Lista colaboradores ativos de uma organização (rota aninhada); posse: organização do líder logado; retorna lista ordenada por nome ASC | LIDER (posse de organização checada) |
| `GET /meu-colaborador` | Retorna o registro de colaborador vinculado ao liderado logado (usuario_id == JWT); sem parâmetro; sem ApenasLider | COLABORADOR (self) ou LIDER com vínculo |

**Regras de negócio:**

- Soft delete: Deletar preenche deletado_em (time.Now()) e deletado_por (usuarioID do líder logado); todas as queries filtram deletado_em IS NULL
- Inativação (desligamento): desligado_em marca colaborador como inativo (saída de empresa/equipe) mas preserva registro para histórico/linha do tempo (diferente de soft delete); campo nulo = ativo
- Posse Cadeia B: colaborador pertence ao líder se equipe.usuario_id == líder OU organizacao.usuario_id == líder (query EXISTS com UNION); resource alheio retorna 404 (ErrAcessoNegado)
- Unicidade de e-mail escopada ao líder: líder não pode ter 2+ liderados ATIVOS com o mesmo e-mail; valida em Criar/Atualizar via ExisteEmailNoLider(email, usuarioID, excetoID); liderados deletados (deletado_em IS NULL) permitem reutilização do e-mail
- Anti-sequestro de conta: liderado não pode usar e-mail do próprio gestor (validado via EmailEhDoLider: SELECT EXISTS FROM tb_usuarios); bloqueia em Criar e Atualizar, retorna ErrEmailDoGestor (409)
- Vínculo de usuário (usuario_id) só pelo fluxo de aceite de convite: Criar ignora usuario_id do DTO (setado nil); Atualizar também ignora usuario_id (comentário de segurança); apenas VincularConta (método dedicado, chamado pelo convite) modifica usuario_id
- Acesso leitura (BuscarPorId, UploadFoto): líder dono OU o próprio liderado (self: col.UsuarioID == usuarioID do JWT)
- Data de nascimento: convertida de string YYYY-MM-DD para time.Time em Criar/Atualizar; erros de formato retornam mensagem fixa; data_nascimento pode ser nula
- Atualização parcial (patch): Atualizar carrega estado atual, sobrescreve só os campos enviados; strings vazias ('') são ignoradas; ponteiros (usuario_id, template_id, whatsapp) só são atualizados se != nil (permite enviar null para limpar)
- Upload de foto: validação dupla — controller: MaxBytesReader 5MB, tiposImagemPermitidos (image/jpeg, image/png, image/webp); usecase: extensão derivada do tipo, caminho colaboradores/{id}/foto{ext} transformado em chave completa via storage.ChaveCompleta(), salvo em foto_key
- URL presignada de foto: gerada sob demanda por gerarFotoURL; expira em 2h (storage.ExpiracaoURLFoto); retorna nil se foto_key nulo ou armazenamento não configurado; nunca retorna foto_key bruta (exposição controlada)

**Validações:**

- CriarColaboradorDTO.OrganizacaoID: required (binding)
- CriarColaboradorDTO.EquipeID: required (binding)
- CriarColaboradorDTO.Nome: required, min=2, max=100 (binding)
- CriarColaboradorDTO.Email: required, email, max=150 (binding)
- CriarColaboradorDTO.UsuarioID: omitempty (opcional; ignorado no usecase mesmo se enviado)
- CriarColaboradorDTO.TemplateID: omitempty (opcional)
- CriarColaboradorDTO.Whatsapp: omitempty, max=20 (opcional)
- CriarColaboradorDTO.DataNascimento: omitempty, convertido de string YYYY-MM-DD para time.Time no usecase; erro de formato retorna msg fixa
- AtualizarColaboradorDTO: todos os campos opcionais (omitempty), mesmas validações de binding acima para nome, email, whatsapp, data_nascimento
- AtualizarColaboradorDTO.UsuarioID: IGNORADO (comentário no código: segurança)
- UploadFoto: Content-Type validado em controller (tiposImagemPermitidos: image/jpeg, image/png, image/webp); tamanho máx 5MB (http.MaxBytesReader); sem validação MIME real, apenas header
- DesligarColaboradorDTO.DataDesligamento: omitempty, formato YYYY-MM-DD; parseado com time.ParseInLocation (meio-dia local, evita conversão de fuso); default time.Now() se vazio
- Posse: todas as operações de leitura/escrita validam no usecase: PertenceAoLider(colaboradorID, usuarioID) ou PodeAcessar(self check); falta de posse retorna ErrAcessoNegado (404)
- Unicidade de e-mail: validada no usecase via repo.ExisteEmailNoLider e repo.EmailEhDoLider; erros mapeados para 409 (ErroConflito) no controller

#### 12.5 Convite (`convite`)

Convite de liderado — o gestor gera um link (UUID) + código para o liderado criar/vincular sua conta e ganhar acesso ao sistema.

**Tabela(s):** **tb_convites** (migration 004): id (VARCHAR(36), PK, token do link), colaborador_id (VARCHAR(36), FK, índice), codigo_hash (VARCHAR(255), hash bcrypt), status (VARCHAR(20), default PENDENTE; valores: PENDENTE | ACEITO | CANCELADO, índice), expira_em (DATETIME), criado_em (DATETIME), aceito_em (DATETIME, nullable)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /colaboradores/:id/convite` | Gera um convite para um colaborador (liderado) do líder logado. Devolve token (UUID) + código (6 caracteres, texto puro — mostrado só uma vez) + link relativo + data de expiração. Cancela convites pendentes anteriores do mesmo colaborador. | Autenticado (JWT) como LIDER (middleware ApenasLider barra COLABORADOR) |
| `GET /convites/:token` | Visualiza dados públicos do convite (nome do liderado, e-mail, validade). Autorizado pelo token do link; retorna se ainda é válido (status PENDENTE e não expirado) ou se já foi cancelado/aceito. | Público (sem autenticação) |
| `POST /convites/:token/aceitar` | Aceita o convite validando código + senha. Se já existe usuário com aquele e-mail e a senha confere, usa a conta existente (caso de troca de gestor/empresa). Senão cria nova conta com role COLABORADOR, vincula colaborador.usuario_id à conta, marca convite como ACEITO e devolve login (token JWT) para acesso imediato. | Público (sem autenticação); autorização via token + código do convite |

**Regras de negócio:**

- Código aleatório de 6 caracteres (alfabeto sem O/0/I/1 para facilitar digitação), gerado com crypto/rand e persistido como hash bcrypt (nunca em texto puro no banco)
- Ao gerar um novo convite para um colaborador, cancela todos os convites pendentes anteriores desse mesmo colaborador (mantém apenas um válido)
- Posse validada via PertenceAoLider (colaboradorUseCase) — um líder não pode gerar convite para liderado de outro líder
- Validação de aceite: status obrigatoriamente PENDENTE, não expirado (7 dias), código bcrypt correto
- Reutilização de conta (troca de gestor/empresa): se usuário com aquele e-mail já existe e senha confere, Login retorna a conta existente; senão cria nova
- Vínculo de conta (anti-sequestro): a ligação colaborador → usuario_id é feita exclusivamente via VincularConta no fluxo de aceite, nunca por PUT genérico
- Envio de e-mail com link + código é não-bloqueante (dorme se SMTP não está ligado); o gestor recebe link+código também na resposta de gerar
- Auditoria automática via middleware global (todas as operações são registradas)

**Validações:**

- DTOs com binding: AceitarConviteDTO.Codigo (required), AceitarConviteDTO.Senha (required, min=6 caracteres)
- Controller valida JSON binding na rota Aceitar (ShouldBindJSON); erros de parse → 400 ErroRequisicao
- UseCase.Gerar valida posse via colaboradorUC.PertenceAoLider (bool); colaborador não encontrado ou não pertencente → 404 ErroNaoEncontrado
- UseCase.BuscarPublico valida existência e validade (status=PENDENTE ∧ agora < expira_em); convite não encontrado → 404
- UseCase.Aceitar valida sequencialmente: token válido, status=PENDENTE, não expirado, código bcrypt correto; falhas → 400 ErroRequisicao
- E-mail do colaborador lido do banco (via BuscarInternoPorId); deve estar preenchido para envio de convite por e-mail
- Colaborador ID é UUID (gerado e validado por módulos anteriores)

### Templates de pauta

#### 12.6 Template (`template`)

Módulo de gerenciamento de templates (modelos de formulários) para estruturar reuniões one-on-one, com CRUD completo escopo ao líder dono.

**Tabela(s):** tb_template: id (VARCHAR 36 UUID PK), usuario_id (VARCHAR 36 FK tb_usuarios), nome (VARCHAR 100), criado_em (DATETIME), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL soft-delete), deletado_por (VARCHAR 36 NULL)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /templates` | Criar novo template vinculado ao líder autenticado. Gera UUID no servidor, preenche UsuarioID do JWT e CriadoEm com time.Now(). | LIDER (authMiddleware + ApenasLider()) |
| `GET /templates` | Listar todos os templates ativos do líder autenticado, ordenados por criado_em ASC (primeiro criado é padrão por herança no módulo onebyone). | LIDER (authMiddleware + ApenasLider()) |
| `GET /templates/:id` | Buscar um template ativo pelo UUID; checa posse (UsuarioID == JWT) no UseCase, retorna 404 se não encontrado ou alheio. | LIDER dono (authMiddleware + ApenasLider(); posse checada no UseCase) |
| `PUT /templates/:id` | Atualizar (renomear) um template; checa posse no UseCase, preenche AlteradoEm automaticamente com time.Now(). | LIDER dono (authMiddleware + ApenasLider(); posse checada no UseCase; erro posse → 403) |
| `DELETE /templates/:id` | Soft delete (exclusão lógica) de um template; checa posse no UseCase, preenche deletado_em e deletado_por. | LIDER dono (authMiddleware + ApenasLider(); posse checada no UseCase; erro posse → 403) |

**Regras de negócio:**

- Soft delete: registros não são removidos fisicamente; deletado_em e deletado_por são preenchidos, e todas as queries filtram 'WHERE deletado_em IS NULL'
- Posse (Cadeia A): template.UsuarioID == JWT; validada em BuscarPorId, Atualizar e Deletar; recurso alheio retorna 404 (ErrAcessoNegado)
- Propriedade na criação: UsuarioID é extraído do JWT, não do corpo; UUID gerado no servidor via uuid.New().String()
- Timestamps automáticos: CriadoEm preenchido com time.Now() na criação; AlteradoEm preenchido com time.Now() em cada UPDATE no repositório
- Herança de template: primeiro template criado (ordenação por criado_em ASC) é o padrão do líder, usado pelo módulo onebyone em caso de nenhum template específico
- Validação de propriedade em PUT/DELETE: erro fixo 'você não tem permissão para alterar/excluir este template' mapeado para 403 no controller
- Defesa em profundidade: UPDATE e DELETE usam 'WHERE id = ? AND deletado_em IS NULL', verificam RowsAffected; erro se nenhuma linha afetada

**Validações:**

- CriarTemplateDTO.Nome: binding 'required,min=2,max=100' (obrigatório, 2 a 100 caracteres)
- AtualizarTemplateDTO.Nome: binding 'required,min=2,max=100' (obrigatório na atualização, 2 a 100 caracteres)
- UsuarioID no criar: NÃO vem do corpo; extraído do JWT (middleware.ChaveUsuarioID) e repassa ao UseCase
- ID no criar: gerado no UseCase via uuid.New().String(); cliente não envia
- Validação de posse (BuscarPorId, Atualizar, Deletar): template.UsuarioID comparado com usuarioID do JWT no UseCase; se diferente retorna ErrAcessoNegado → 404
- Falha de binding JSON: 400 (ErroRequisicao) com mensagem 'dados inválidos: <erro>'
- Erro de permissão (Atualizar/Deletar): mensagem fixa comparada no controller para responder 403 (ErroProibido)
- CriadoEm e AlteradoEm: preenchidas automaticamente no servidor; cliente não envia

#### 12.7 Template-bloco (`templatebloco`)

Gerencia blocos (campos de formulário) dentro de templates de reunião 1:1, permitindo criar, buscar, listar, atualizar e excluir campos estruturados por tipo (TEXT, IMAGE, LIST, HIGHLIGHT) e ordem de exibição.

**Tabela(s):** tb_template_blocos: id (VARCHAR 36, PK), template_id (VARCHAR 36, FK → tb_template), tipo (ENUM TEXT/IMAGE/LIST/HIGHLIGHT), posicao (INT, default 0), rotulo (VARCHAR 150), criado_em (DATETIME), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL), deletado_por (VARCHAR 36 NULL). Soft delete via deletado_em/deletado_por; todas as queries filtram deletado_em IS NULL.

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /template-blocos` | Cria um novo bloco dentro de um template existente. Gera UUID no servidor e timestamp criado_em. Exige posse do template (template → usuario_id == JWT). | LIDER (middleware ApenasLider) + posse via template pai |
| `GET /template-blocos/:id` | Busca um bloco ativo pelo UUID. Valida posse do template pai antes de retornar. | LIDER (middleware ApenasLider) + posse via template pai |
| `GET /templates/:id/blocos` | Lista todos os blocos ativos de um template, ordenados por posicao ASC. Valida posse do template antes de listar. | LIDER (middleware ApenasLider) + posse via template pai |
| `PUT /template-blocos/:id` | Atualização parcial: sobrescreve apenas os campos informados (tipo, posicao, rotulo), preservando os demais. Define alterado_em automaticamente. Valida posse do template pai. | LIDER (middleware ApenasLider) + posse via template pai |
| `DELETE /template-blocos/:id` | Soft delete (exclusão lógica): preenche deletado_em e deletado_por com o usuário autenticado. Valida existência antes de deletar (RowsAffected check). | LIDER (middleware ApenasLider) + posse via template pai |

**Regras de negócio:**

- Posse herdada do template pai: bloco só é acessível se template.usuario_id == JWT (validado em garantirPosseTemplate via templateUC.PertenceAoUsuario); recurso alheio → 404 (ErrAcessoNegado mapeado em responderErro)
- UUID gerado no servidor: Criar() chama uuid.New().String() para id, cliente não envia
- Soft delete: registros não são apagados fisicamente; DELETE preenche deletado_em (time.Now()) e deletado_por (usuarioID do JWT). Todas as queries filtram deletado_em IS NULL
- Atualização parcial: Atualizar() carrega estado atual, sobrescreve apenas campos informados (tipo se não vazio, rotulo se não vazio, posicao se ponteiro não nil), preserva demais
- alterado_em definido automaticamente: Atualizar() força alterado_em = time.Now(), cliente não controla
- Listagem sempre ordenada: ListarPorTemplate() retorna blocos ordenados por posicao ASC (ORDER BY posicao ASC)
- Validação de existência antes de deletar: Deletar() chama BuscarPorId primeiro; se não existe, retorna erro. DeletarSoft() valida RowsAffected() — se 0, retorna 'não encontrado ou já deletado'
- Tipo restrito a 4 valores: TEXT, IMAGE, LIST, HIGHLIGHT (ENUM no banco, validado com oneof na DTO)

**Validações:**

- CriarTemplateBlocoDTO.template_id: required (uuid do template de destino)
- CriarTemplateBlocoDTO.tipo: required,oneof=TEXT IMAGE LIST HIGHLIGHT (exatamente um dos 4 valores)
- CriarTemplateBlocoDTO.posicao: min=0 (ordem de exibição, não negativa)
- CriarTemplateBlocoDTO.rotulo: required,min=1,max=150 (label entre 1 e 150 chars)
- AtualizarTemplateBlocoDTO.tipo: omitempty,oneof=TEXT IMAGE LIST HIGHLIGHT (opcional, mas se enviado deve ser válido)
- AtualizarTemplateBlocoDTO.posicao: omitempty,min=0 (ponteiro para distinguir nil de 0; se enviado, >= 0)
- AtualizarTemplateBlocoDTO.rotulo: omitempty,min=1,max=150 (opcional, mas se enviado entre 1 e 150 chars)
- Controller ignora template_id no PUT Atualizar (só usa id do path, template já carregado via BuscarPorId)
- Controller ignora usuario_id, deletado_em, deletado_por no body (não sobrescrevíveis via API)

### Reunião 1:1

#### 12.8 OneByOne (a reunião) (`onebyone`)

Gerencia as reuniões one-on-one agendadas entre líderes e colaboradores, com suporte a recorrência e herança de template.

**Tabela(s):** tb_onebyone: id (VARCHAR 36 PK), usuario_id (VARCHAR 36 FK → tb_usuarios, dono), organizacao_id (VARCHAR 36 FK), equipe_id (VARCHAR 36 FK), colabor_id (VARCHAR 36 FK → tb_colaboradores), recorrencia (ENUM NENHUMA/MENSAL/QUINZENAL), status (ENUM AGENDADO/REALIZADO/PENDENTE, default AGENDADO), **realizado_em (DATETIME NULL — quando o 1:1 foi realizado, migration 015)**, data_agendada (DATE), criado_em (DATETIME), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL), deletado_por (VARCHAR 36 NULL). Soft delete ativo (deletado_em IS NULL filtra todas as queries). Relacionamentos: ResolverTemplate usa tb_colaboradores.template_id (prioridade 1), tb_equipes.template_id (prioridade 2), tb_organizacoes.template_id (prioridade 3), template mais antigo do líder (prioridade 4).

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /api/v1/onebyone` | Agenda uma nova reunião one-on-one. Recebe organizacao_id, equipe_id, colabor_id, recorrencia (opcional, padrão NENHUMA), data_agendada. Gera UUID, status inicial AGENDADO, usuario_id extraído do JWT. | LIDER autenticado (JWT) |
| `POST /api/v1/onebyone/encerrar` | Registra um 1:1 como REALIZADO no livro-razão. Recebe `{colabor_id}`; cria a linha com status=REALIZADO, data_agendada=hoje e realizado_em=NOW(). Idempotente por dia (BuscarRealizadoNoDia). Resolve organização/equipe e posse via colaborador (Cadeia B). Alimenta o módulo saude1a1. | LIDER dono do colaborador (ApenasLider + PertenceAoLider) |
| `GET /api/v1/onebyone` | Lista todas as reuniões ativas do líder autenticado, ordenadas por data_agendada DESC (mais recentes primeiro). | LIDER autenticado (JWT) |
| `GET /api/v1/onebyone/:id` | Busca uma reunião ativa pelo UUID. Valida posse: só retorna se usuario_id == JWT, senão 404. | LIDER dono da reunião (posse validada) |
| `PUT /api/v1/onebyone/:id` | Atualiza status, recorrencia e/ou data_agendada (campos opcionais, só informados são atualizados). Valida posse; preenche alterado_em com data/hora atual. | LIDER dono da reunião (posse validada) |
| `DELETE /api/v1/onebyone/:id` | Soft delete: preenche deletado_em (timestamp) e deletado_por (usuario_id do JWT). Valida posse; retorna erro 404 se não encontrado ou já deletado. | LIDER dono da reunião (posse validada) |

**Regras de negócio:**

- Posse (Cadeia A): todo one-on-one tem usuario_id que deve igualar o JWT. GET/:id, PUT/:id, DELETE/:id validam: BuscarPorId(id) → if reuniao.UsuarioID != usuarioID return ErrAcessoNegado (404).
- Soft delete: todas as queries de leitura/escrita filtram deletado_em IS NULL. DeletarSoft preenche deletado_em (NOW()) e deletado_por (usuarioID).
- Status inicial: ao criar, status sempre é AGENDADO (definido no usecase, não vem do DTO).
- Recorrência padrão: se não informada no POST, usecase atribui NENHUMA.
- Validação de data: data_agendada (string) convertida com layout 2006-01-02 (YYYY-MM-DD). Erro se formato inválido → mensagem orientando YYYY-MM-DD.
- Atualização parcial: PUT só altera campos não vazios do DTO. Repositório atualiza automaticamente alterado_em.
- Herança de template (ResolverTemplate/ResolverTemplateID): uma única query SQL com COALESCE resolve o template em ordem: colaboradores.template_id (prioridade 1) → equipes.template_id (prioridade 2) → organizacoes.template_id (prioridade 3) → template mais antigo do líder ORDER BY criado_em ASC LIMIT 1 (prioridade 4). Se nenhum encontrado, erro orientando criar template. Usado por registroonebyone ao abrir reunião.
- Reuso por módulos: usecase.PertenceAoUsuario (bool check) é reaproveitado por registroonebyone e valorregistro para validar posse via one-on-one pai.

**Validações:**

- DTO de criação (CriarOneByOneDTO): organizacao_id required, equipe_id required, colabor_id required, recorrencia omitempty,oneof=NENHUMA MENSAL QUINZENAL, data_agendada required.
- DTO de atualização (AtualizarOneByOneDTO): status omitempty,oneof=AGENDADO REALIZADO PENDENTE, recorrencia omitempty,oneof=NENHUMA MENSAL QUINZENAL, data_agendada omitempty.
- Formato de data: string YYYY-MM-DD, parseado com time.Parse(2006-01-02). Erro 400 se inválido com mensagem orientando formato.
- Campos ignorados: usuario_id nunca vem do corpo — sempre extraído do JWT via middleware.ChaveUsuarioID. ID gerado pelo servidor (uuid.New().String()), não aceita do cliente.

#### 12.8b Saúde do 1:1 + Streak (`saude1a1`)

"Saúde do 1:1" do gestor — resumo de cadência que fecha o ciclo de engajamento (card do `/painel`). Leitura agregada, somente leitura (sem mapper). Detalhe: [internal/saude1a1/README.md](../internal/saude1a1/README.md).

**Tabela(s):** nenhuma própria. LÊ `tb_onebyone` (realizados: status=REALIZADO + realizado_em) e `tb_agendamentos` (cadência esperada), sempre escopado por `usuario_id`. Índices da migration 015 ajudam o agregado.

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /api/v1/saude-1a1` | Retorna `{ percentual_em_dia, total_agendados, atrasados, realizados_ult_30, streak_semanas }` do gestor. | **ApenasLider** (escopo usuario_id do JWT) |

**Regras de negócio:**

- percentual_em_dia = (agendados − atrasados) / agendados × 100 (100 se não há agenda).
- atrasados = agendamentos ativos com data_hora < agora (ocorrência vencida).
- realizados_ult_30 = COUNT tb_onebyone REALIZADO com realizado_em nos últimos 30 dias.
- streak_semanas = semanas ISO consecutivas (de hoje para trás) com ≥1 realizado. Tolerante: semana atual ainda vazia não quebra. Calculado em Go (calcularStreak), coberto por usecase_test.go.

**Validações:**

- Sem corpo de entrada (só leitura). Escopo sempre pelo usuario_id do token; ApenasLider barra COLABORADOR.

#### 12.9 Registro do 1:1 (`registroonebyone`)

Abre, consulta e exclui registros (formulários preenchidos) de reuniões 1:1, resolvendo automaticamente qual template usar pela regra de herança.

**Tabela(s):** tb_registros_onebyone: id (PK, UUID), oneaone_id (FK→tb_onebyone), template_id (FK→tb_template), criado_em (DATETIME NOT NULL), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL soft delete), deletado_por (VARCHAR(36) NULL)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /registros-onebyone` | Abre um novo registro para uma reunião. O template é resolvido automaticamente pela regra de herança (colaborador→equipe→organização→padrão do líder); o cliente nunca informa o template. | LIDER (posse via onebyone.usuario_id) |
| `GET /registros-onebyone/:id` | Busca um registro ativo (deletado_em IS NULL) pelo UUID. | LIDER (posse via onebyone.usuario_id do registro) |
| `DELETE /registros-onebyone/:id` | Soft delete: preenche deletado_em e deletado_por (ID do usuário logado) sem remover a linha. | LIDER (posse via onebyone.usuario_id) |
| `GET /onebyone/:id/registros` | Lista todos os registros ativos de uma reunião, ordenados por criado_em DESC (mais recente primeiro). | LIDER (posse via onebyone.usuario_id da reunião) |

**Regras de negócio:**

- Resolução automática de template por herança: ao criar, chama onebyoneUC.ResolverTemplate(), não aceita template do cliente
- Soft delete: registros não são removidos fisicamente; deletado_em e deletado_por são preenchidos no UPDATE
- Consultas filtram deletado_em IS NULL: registros deletados logicamente são invisíveis
- Geração de ID no servidor: uuid.New().String(); cliente não controla ID nem CriadoEm
- Persistência com releitura: após INSERT, chama BuscarPorId para devolver o estado real do banco
- Posse via cadeia indireta: registro→oneaone→usuario_id; validada em todas as operações (Criar/BuscarPorId/ListarPorOneByOne/Deletar)
- Recurso alheio retorna 404: falta de posse mapeia ErrAcessoNegado para ErroNaoEncontrado (não revela existência)
- Validação de RowsAffected no DELETE: se nenhuma linha foi afetada, retorna 'registro não encontrado ou já deletado'

**Validações:**

- CriarRegistroOneByOneDTO.OneByOneID: binding:required (obrigatório, não pode estar vazio ou null no JSON)
- JSON binding no controller via ShouldBindJSON: se falhar, responde 400 'dados inválidos: <erro>'
- UUID do registro resolvido no servidor com uuid.New().String()
- Data CriadoEm preenchida no servidor com time.Now(); cliente não controla
- Sem campo de template no DTO de entrada: é resolvido automaticamente pela herança

#### 12.10 Valor de registro (respostas) (`valorregistro`)

Módulo que armazena as respostas preenchidas em cada bloco de formulário de uma reunião 1:1 — o conteúdo textual e estruturado (JSON) efetivo da reunião.

**Tabela(s):** `tb_valores_registro`: id (VARCHAR 36 PK, UUID), registro_id (VARCHAR 36 FK → tb_registros_onebyone, obrigatório), bloco_id (VARCHAR 36 FK → tb_template_blocos, obrigatório), valor_texto (TEXT NULL), valor_json (JSON NULL), criado_em (DATETIME NOT NULL), alterado_em (DATETIME NULL), deletado_em (DATETIME NULL, soft delete marker), deletado_por (VARCHAR 36 NULL, ID do usuário que deletou)

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /valores-registro` | Cria uma nova resposta para um bloco de template dentro de um registro de one-on-one. Valida que pelo menos um dos campos (valor_texto ou valor_json) está preenchido. Gera UUID no servidor e serializa JSON se fornecido. | LIDER (dono da reunião via cadeia: registro → onebyone → usuario_id) |
| `GET /valores-registro/:id` | Busca uma resposta (valor de registro) ativa pelo seu UUID. Retorna os dados completos incluindo conteúdo textual/JSON e timestamps. | LIDER (dono da reunião via cadeia: valor → registro → onebyone → usuario_id) |
| `PUT /valores-registro/:id` | Atualiza parcialmente uma resposta existente: permite modificar valor_texto e/ou valor_json. Preserva campos não enviados. Atualiza automaticamente o timestamp alterado_em. | LIDER (dono da reunião via cadeia) |
| `DELETE /valores-registro/:id` | Realiza soft delete de uma resposta: preenche deletado_em e deletado_por (ID do usuário autenticado). Não remove a linha do banco. | LIDER (dono da reunião via cadeia) |
| `GET /registros-onebyone/:id/valores` | Lista todas as respostas ativas de um registro específico, ordenadas por data de criação (ascendente). | LIDER (dono da reunião via cadeia) |

**Regras de negócio:**

- Posse herdada em cadeia: valor → registro → onebyone → usuario_id (resolvida por RegistroPertenceAoUsuario do registroonebyone.UseCase)
- Obrigatoriedade: ao criar, pelo menos um de valor_texto ou valor_json deve ser informado; ambos nulos retorna erro
- Serialização JSON: valor_json chega como interface{}, é convertido para bytes via json.Marshal e armazenado na coluna JSON do banco
- Desserialização na saída: mapper deserializa bytes JSON de volta para object; se vazio, retorna null no DTO
- UUID no servidor: ID gerado pelo backend (uuid.New().String()), não enviado pelo cliente
- Timestamp criado_em: definido no UseCase com time.Now(), não controlado pelo cliente
- Atualização parcial (PATCH): registra atual é carregado antes; apenas campos enviados são sobrescritos
- Timestamp alterado_em: definido automaticamente no Repository (tempo.Now()) ao atualizar; nunca nulo se registro foi modificado
- Soft delete: nenhum registro é removido fisicamente; deletado_em e deletado_por são preenchidos
- Filtro de leitura: todas as queries (SELECT, UPDATE, DELETE) incluem WHERE deletado_em IS NULL
- Verificação de existência: Deletar carrega o registro antes para confirmar existência ativa
- Defesa em profundidade: UPDATE/DELETE reforçam WHERE deletado_em IS NULL e checam RowsAffected

**Validações:**

- CriarValorRegistroDTO.registro_id: obrigatório (binding:required), string UUID do registro
- CriarValorRegistroDTO.bloco_id: obrigatório (binding:required), string UUID do bloco de template
- CriarValorRegistroDTO.valor_texto: opcional (binding:omitempty), ponteiro para string
- CriarValorRegistroDTO.valor_json: opcional (binding:omitempty), interface{} (objeto JSON livre)
- AtualizarValorRegistroDTO.valor_texto: opcional (binding:omitempty), ponteiro para string
- AtualizarValorRegistroDTO.valor_json: opcional (binding:omitempty), interface{}
- Validação no UseCase: se ambos valor_texto e valor_json forem nil na criação, retorna erro 'é necessário informar valor_texto ou valor_json'
- Validação no UseCase: json.Marshal(valor_json) falha com 'valor_json inválido' se o objeto não for serializável
- Posse (UseCase): RegistroPertenceAoUsuario(registroID, usuarioID) via registroonebyone.UseCase retorna false → ErrAcessoNegado → Controller responde 404
- Tratamento de erro: ErrAcessoNegado é traduzido para 404 'resposta não encontrada' no controller; demais erros retornam 500

#### 12.11 1:1 ao vivo (WebSocket) (`aovivo`)

Gerencia reuniões 1:1 em tempo real via WebSocket (presença, cursores, quadro colaborativo).

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /api/v1/ws/1a1/:sala` | Upgrade HTTP→WebSocket para entrar em uma sala de 1:1 ao vivo. Retorna id da conexão e lista de participantes. Aceita query params: ?token= (JWT), ?nome= (padrão: 'Alguém'), ?papel= (gestor/LIDER ou colaborador). | Autenticado via JWT em ?token=. O :sala é identificado por colaborador_id (pertence ao 1:1 entre líder e liderado). Gestor e liderado da mesma 1:1 podem entrar. Sem posse explícita no código (segue implícito do fluxo de acesso ao 1:1 pai). |

**Regras de negócio:**

- Hub gerencia múltiplas salas (uma por colaborador_id), criadas sob demanda.
- Cliente conectado recebe UUID próprio e cor por papel (índigo/gestor, coral/liderado).
- Broadcast sem eco: mensagens do remetente não são retransmitidas a si próprio; cliente aplica mudança localmente.
- Estado persistido por sala: último tabuleiro (tipo:'tabuleiro') é guardado e enviado a novos participantes (para sincronizar atrasos).
- Sinal de encerramento (tipo:'encerrado') é persistido: quem entrar depois cai em modo consulta (somente-leitura). Todos (inclusive remetente) recebem o sinal.
- Envio não-bloqueante: se buffer de cliente lento enche, mensagem é descartada (sem backpressure).
- Sala é removida quando última conexão desconecta (garbage collection automático).

**Validações:**

- JWT validado via ?token=; falha → 401 Unauthorized.
- Nome extraído de ?nome=; se vazio, padrão 'Alguém'.
- Papel extraído de ?papel=; mapeia para cor (LIDER/gestor → índigo, outro → coral).
- Upgrade HTTP→WebSocket falha silenciosamente se header/protocolo inválido (padrão gorilla/websocket).
- Mensagens JSON parseadas com struct base {tipo:string} para roteamento; decode falha → descarta mensagem.
- Read limit de 64KB por mensagem (1<<16 bytes).
- Buffer de envio 32 mensagens por cliente; excesso → non-blocking discard (default).

#### 12.12 Tabuleiro da pauta (`tabuleiro`)

Persistência e acesso ao estado do tabuleiro (pauta) do 1:1, compartilhado entre líder e liderado (board colaborativo).

**Tabela(s):** tb_tabuleiros: colaborador_id (VARCHAR 36, PK), estado (JSON), criado_em (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP), alterado_em (DATETIME NULL)

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/tabuleiro` | Obtém o estado salvo do tabuleiro do liderado (JSON com colunas, temas, banco/pauta/conversado). Retorna null se nunca foi salvo. | LÍDER (dono) OU COLABORADOR (o próprio liderado) |
| `PUT /colaboradores/:id/tabuleiro` | Salva/atualiza o estado completo do tabuleiro do liderado (upsert por colaborador_id). Estado deve ser JSON válido. | LÍDER (dono) OU COLABORADOR (o próprio liderado) |

**Regras de negócio:**

- Posse via PodeAcessar (Cadeia B): líder dono OU o próprio liderado (board é colaborativo — ambos manipulam temas ao vivo)
- Upsert: INSERT ON DUPLICATE KEY UPDATE por colaborador_id (uma única linha por liderado)
- Sem estrutura — backend não interpreta o JSON: persiste como-é, devolve como-é; responsabilidade do frontend é a pauta com colunas/temas
- Recurso alheio → 404 (mapeia ErrAcessoNegado)

**Validações:**

- Estado: obrigatório (binding:required) — JSON bruto (json.RawMessage)
- Colaborador_id: UUID recebido via parâmetro de URL (validado como válido antes de rotas)
- Posse: garantida no UseCase via PodeAcessar antes de Buscar/Salvar (defesa em profundidade)

### Conteúdo & desenvolvimento do liderado

#### 12.13 Bloco de tema (`blocotema`)

Conteúdo rico de temas de 1:1 (blocos de texto, link/curso, imagem no S3 e marcos com datas) por liderado, ordenável por tema, com upload de imagem e URLs presignadas

**Tabela(s):** tb_blocos_tema (migration 005): id (UUID, PK), colaborador_id (FK), tema (VARCHAR 150), tipo (ENUM: TEXTO/LINK/IMAGEM/MARCO), texto (TEXT nullable), url (VARCHAR 500 nullable), imagem_key (VARCHAR 255 nullable, chave S3), data_inicio (DATETIME nullable), data_fim (DATETIME nullable), ordem (INT default 0), criado_em (DATETIME). Index: (colaborador_id, tema)

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/blocos` | Lista os blocos de um tema específico do colaborador (filtra via query param ?tema=...) | LIDER dono OU o próprio COLABORADOR (liderado) |
| `GET /colaboradores/:id/blocos/tudo` | Lista TODO o conteúdo (todos os temas) do colaborador, agrupável por tema — usado pela IA | LIDER dono OU o próprio COLABORADOR (liderado) |
| `POST /colaboradores/:id/blocos` | Cria um bloco de tipo TEXTO, LINK ou MARCO (imagem entra por rota separada). Calcula automaticamente a ordem (MAX+1) por tema | LIDER dono OU o próprio COLABORADOR (liderado) |
| `POST /colaboradores/:id/blocos-imagem` | Upload multipart de imagem (JPEG/PNG/WebP, max 5MB) → cria bloco IMAGEM no S3 com URL presignada (~2h). Path separado para não colidir com :blocoId | LIDER dono OU o próprio COLABORADOR (liderado) |
| `DELETE /colaboradores/:id/blocos/:blocoId` | Remove um bloco pelo UUID, validando que pertence ao colaborador da rota (evita deleção cross-colaborador) | LIDER dono OU o próprio COLABORADOR (liderado) |

**Regras de negócio:**

- Posse validada via colaboradorUC.PodeAcessar (líder dono OU liderado próprio, já que ambos editam conteúdo do tema no 1:1 ao vivo)
- Deleção é FÍSICA (DELETE FROM tb_blocos_tema) — NO soft delete
- Imagem → upload S3 em caminho temas/{colaboradorId}/{blocoId}.{ext}, retorna URL presignada (expira ~2h) em vez de chave crua
- Ordem calculada como MAX(ordem)+1 por (colaborador_id, tema) — blocos ordenáveis dentro de cada tema
- Datas de MARCO guardadas ao meio-dia (12h) para conversões de fuso horário não virarem o dia
- Validação dupla em DELETE: (1) posse do colaborador + (2) verificação que bloco.colaborador_id == param:id (evita apagar bloco de outro colaborador passando ID que você possui)

**Validações:**

- GET /blocos: query ?tema= obrigatória (erro 400 se ausente)
- POST /blocos (CriarBlocoDTO): tema obrigatório e max 150 chars; tipo obrigatório e enum (TEXTO|LINK|MARCO); texto/url/data_inicio/data_fim opcionais
- POST /blocos-imagem: tema obrigatório (via PostForm); arquivo 'imagem' obrigatório; Content-Type validado contra whitelist (image/jpeg, image/png, image/webp); corpo limitado a 5MB; legenda opcional
- DELETE /blocos/:blocoId: param blocoId deve ser UUID válido e existente, vínculo com colaborador_id checado no UseCase
- Todas as rotas exigem JWT Bearer Token (authMiddleware global)

#### 12.14 Classificação 9-box (`classificacao`)

Guarda e lista a posição de cada liderado na matriz 9-box (desempenho × potencial), usada pelo Monitor do gestor para acompanhar evolução, destacar talentos e antecipar riscos.

**Tabela(s):** **tb_classificacoes**: colaborador_id (VARCHAR(36), PK), desempenho (VARCHAR(10), BAIXO|MEDIO|ALTO), potencial (VARCHAR(10), BAIXO|MEDIO|ALTO), atualizado_em (DATETIME, DEFAULT CURRENT_TIMESTAMP).

| Rota | O que faz | Acesso |
|---|---|---|
| `PUT /colaboradores/:id/classificacao` | Define/atualiza a posição 9-box de um liderado (upsert desempenho + potencial) | LIDER dono do colaborador (posse validada via colaboradorUC.PertenceAoLider) |
| `GET /organizacoes/:id/classificacoes` | Lista as classificações de todos os liderados ativos de uma organização | LIDER dono da organização (posse validada via colaboradorUC.OrganizacaoPertenceAoLider) |

**Regras de negócio:**

- Upsert: INSERT ... ON DUPLICATE KEY UPDATE — uma classificação por colaborador (colaborador_id é PK)
- Posse cadeia B: classificacao → colaborador_id → equipe.usuario_id OU organizacao.usuario_id; validada no usecase via colaboradorUC.PertenceAoLider (Definir) e colaboradorUC.OrganizacaoPertenceAoLider (ListarPorOrganizacao)
- Listagem filtra colaboradores ativos (deletado_em IS NULL) via JOIN com tb_colaboradores
- Acesso negado (ErrAcessoNegado) mapeado para 404 no controller (segurança: não revela existência de recurso alheio)

**Validações:**

- DefinirClassificacaoDTO: desempenho obrigatório, oneof=BAIXO MEDIO ALTO; potencial obrigatório, oneof=BAIXO MEDIO ALTO (binding:required,oneof)
- Colaborador deve existir e pertencer ao líder (validação de posse no usecase via PertenceAoLider)
- Organização deve pertencer ao líder (validação de posse no usecase via OrganizacaoPertenceAoLider)
- Middleware ApenasLider() em ambas as rotas (defesa em profundidade: barra contas COLABORADOR)

#### 12.15 PDI — Plano de Desenvolvimento Individual (`pdi`)

Gestão de itens de PDI (objetivos/ações) de um liderado com prazos e status de conclusão, registrando quando cada meta foi concluída para desenhar evoluções.

**Tabela(s):** tb_pdi_itens (id UUID, colaborador_id UUID, titulo VARCHAR(255), descricao TEXT NULL, prazo DATE NULL, concluido TINYINT(1) DEFAULT 0, concluido_em DATETIME NULL, criado_em DATETIME, alterado_em DATETIME NULL, deletado_em DATETIME NULL); INDEX idx_pdi_colaborador (colaborador_id); FK: colaborador_id garantida pela aplicação

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/pdi` | Lista todos os itens de PDI de um liderado, ordenados por status (incompletos primeiro), depois por prazo (mais próximos primeiro), depois por criação (mais recentes primeiro). Filtra deletados (soft delete). | LIDER dono (via colaborador.PertenceAoLider) ou próprio liderado (verifica posse no usecase) |
| `POST /colaboradores/:id/pdi` | Cria um novo item de PDI para o liderado (título obrigatório, descrição opcional, prazo opcional em YYYY-MM-DD). Gera UUID, seta concluido=false e criado_em=agora. | LIDER dono (middleware ApenasLider + posse no usecase) |
| `PUT /pdi/:id` | Atualiza campos do item (todos opcionais): título, prazo, status de conclusão. Se marca como concluído, carimbaa concluido_em=agora; se reabre, limpa concluido_em. Atualiza alterado_em. | LIDER dono (middleware ApenasLider + posse no usecase via item.colaborador_id) |
| `DELETE /pdi/:id` | Remove logicamente o item de PDI (soft delete): seta deletado_em=agora. Retorna 404 se já deletado ou não encontrado. | LIDER dono (middleware ApenasLider + posse no usecase) |

**Regras de negócio:**

- Posse: verificada via colaborador.PertenceAoLider(colaboradorID, usuarioID) — garante que o líder JWT é o gestor do liderado (Cadeia B: colaborador → equipe.usuario_id OU organizacao.usuario_id)
- Soft delete: registros não são apagados; preenchem deletado_em. Todas as queries filtram deletado_em IS NULL
- Prazo: parseado como YYYY-MM-DD (time.Local); adicionado 12h para evitar pulo de fuso (meio-dia UTC local)
- Conclusão com carimbo: quando marcado como concluído (concluido=true), concluido_em recebe agora; ao reabrir, limpa. Permite desenhar evolução de burn-up do PDI no tempo
- Recurso alheio: retorna 404 (ErrAcessoNegado mapeado para ErroNaoEncontrado no controller), não 403 (não revela que id existe)
- Descrição opcional: salva como NULL se não informada; nos DTOs, ignorada se string vazia
- Título não atualizável para vazio: PUT com título vazio é ignorado (só atualiza se `dto.Titulo != ""`)

**Validações:**

- CriarItemPDIDTO.Titulo: required, min=2, max=255 (binding tags)
- CriarItemPDIDTO.Descricao: omitempty (optional)
- CriarItemPDIDTO.Prazo: omitempty, formato YYYY-MM-DD (validado no usecase com parsearPrazo, erro: 'prazo inválido — use AAAA-MM-DD')
- AtualizarItemPDIDTO.Titulo: omitempty, min=2, max=255 quando presente
- AtualizarItemPDIDTO.Prazo: omitempty, formato YYYY-MM-DD quando presente
- AtualizarItemPDIDTO.Concluido: omitempty, *bool (update só se != nil e diferente do valor atual)
- Listar: valida posse do colaborador no usecase; se sem acesso, retorna ErrAcessoNegado → 404
- Criar/Atualizar/Deletar: valida posse do colaborador (ou do item via BuscarPorId → colaborador_id) antes de operação

#### 12.16 Acompanhamento (sentimento/entregas/feedbacks/estudos) (`acompanhamento`)

Módulo de acompanhamento unificado: registra sentimento do liderado (humor 1-5), entregas, feedbacks recebidos e estudos, cada um com data de referência, título e detalhe opcional.

**Tabela(s):** tb_acompanhamentos: id (PK, UUID), colaborador_id (índice + índice composto com tipo), tipo (SENTIMENTO|ENTREGA|FEEDBACK|ESTUDO), titulo (VARCHAR 255), detalhe (TEXT nullable), valor (INT 1-5 nullable, só SENTIMENTO), data_ref (DATE), criado_em (DATETIME), alterado_em (DATETIME nullable), deletado_em (DATETIME nullable soft delete).

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/acompanhamento` | Lista acompanhamentos do liderado, filtrável por tipo (SENTIMENTO\|ENTREGA\|FEEDBACK\|ESTUDO) via ?tipo=. Retorna DTO com id, colaborador_id, tipo, titulo, detalhe, valor, data_ref formatada AAAA-MM-DD, criado_em. | LIDER dono do colaborador (posse verificada via PertenceAoLider) |
| `POST /colaboradores/:id/acompanhamento` | Cria acompanhamento. SENTIMENTO exige valor 1-5 (titulo opcional); demais tipos exigem titulo. Data padrão: hoje (meio-dia em timezone local). Gera UUID. | LIDER dono do colaborador (ApenasLider middleware) |
| `PUT /acompanhamento/:id` | Atualiza campos (todos opcionais): titulo, detalhe, valor (1-5), data_ref. Timestamp alterado_em preenchido automaticamente. | LIDER dono do colaborador (ApenasLider middleware, posse validada no UseCase) |
| `DELETE /acompanhamento/:id` | Remove via soft delete (preenchimento de deletado_em). Checagem de RowsAffected garante recurso existed. | LIDER dono do colaborador (ApenasLider middleware, posse validada no UseCase) |

**Regras de negócio:**

- Posse via Cadeia B (colaborador → equipe.usuario_id OU organizacao.usuario_id): PertenceAoLider validado em todo Criar/Listar/Atualizar/Deletar. Recurso alheio → ErrAcessoNegado → 404.
- Soft delete (deletado_em): todo SELECT filtra 'deletado_em IS NULL'; DeletarSoft preenchimento de timestamp e valida RowsAffected == 1.
- SENTIMENTO (humor): Tipo especial que exige valor 1-5, titulo opcional. Demais tipos (ENTREGA, FEEDBACK, ESTUDO) exigem titulo não-vazio.
- Data de referência (data_ref) parseada em timezone local com time.Parse e offset +12h (meio-dia) para evitar pulo de fuso em exibição.
- Listar ordenado por data_ref DESC, criado_em DESC (acompanhamentos mais recentes primeiro).
- Atualização parcial (PATCH semantics com PUT): campos omitidos no DTO são ignorados; detalhe permite null explícito (pointer *string).

**Validações:**

- CriarAcompanhamentoDTO: tipo obrigatório + oneof (SENTIMENTO|ENTREGA|FEEDBACK|ESTUDO); titulo max 255 chars (omitempty); detalhe omitempty; valor omitempty + min=1 max=5 se presente; data_ref omitempty (formato AAAA-MM-DD validado no UseCase, erro retorna 'data inválida — use AAAA-MM-DD').
- Regra negócio no Criar: se tipo==SENTIMENTO && valor==nil → 'informe o humor (de 1 a 5)'; else if titulo vazio (após trim) → 'informe um título'.
- AtualizarAcompanhamentoDTO: todos opcionais (omitempty); titulo max 255; detalhe *string nullable (permite null); valor min=1 max=5; data_ref omitempty.
- Data_ref: string AAAA-MM-DD, parseada via time.ParseInLocation. Se vazio, usa time.Now() + 12h. Erro customizado no message.
- Posse: BuscarPorId falha → ErrAcessoNegado (retorna como 404). garantirPosse checa PertenceAoLider(colaboradorID, usuarioID) → booleano ou erro; falso → ErrAcessoNegado.

### Agenda & notificações

#### 12.17 Agendamento (`agendamento`)

Agenda de 1:1 (com recorrência) entre líder e liderado, com scheduler que avança ocorrências passadas e envia lembretes diários por e-mail.

**Tabela(s):** Tabela **tb_agendamentos**: id (VARCHAR 36, PK), usuario_id (VARCHAR 36, FK lider dono), colaborador_id (VARCHAR 36, FK liderado), data_hora (DATETIME, próxima ocorrência parseada em fuso local), recorrencia (VARCHAR 20, padrão NENHUMA; valores: NENHUMA|SEMANAL|QUINZENAL|MENSAL), ativo (TINYINT 1, padrão 1), criado_em (DATETIME). Índices: idx_agend_usuario (usuario_id), idx_agend_data (data_hora).

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /api/v1/agendamentos` | Cria novo agendamento de 1:1. Valida posse do liderado (BuscarPorId do colaboradorUseCase). Corpo: {colaborador_id (obrigatório), data_hora (obrigatório, formato YYYY-MM-DDTHH:MM ou RFC3339), recorrencia (opcional, padrão NENHUMA)}. Retorna AgendamentoRespostaDTO com id, colaborador_id, liderado_nome, data_hora (formatado), recorrencia. | LIDER (middleware ApenasLider) |
| `GET /api/v1/agendamentos` | Lista todos os agendamentos ativos do gestor logado, com nome do liderado e próxima data/hora. Ordenados por data_hora ascendente. | LIDER (middleware ApenasLider) |
| `PUT /api/v1/agendamentos/{id}` | Reagenda (arrastar no calendário) um 1:1 do gestor. Valida a nova data/hora e atualiza apenas se agendamento pertence ao gestor (WHERE usuario_id). Corpo: {data_hora (obrigatório)}. | LIDER (middleware ApenasLider) |
| `DELETE /api/v1/agendamentos/{id}` | Cancela um agendamento do gestor. Garante posse via WHERE usuario_id e RowsAffected. | LIDER (middleware ApenasLider) |

**Regras de negócio:**

- Posse (Cadeia A): agendamento.usuario_id == JWT do líder logado. Validada no UseCase (Criar via colaboradorUC.BuscarPorId(colabID, usuarioID) que falha se liderado não pertence ao líder) e em todas as operações de escrita/exclusão via WHERE usuario_id no SQL (DELETE, UPDATE).
- Validação de liderado: Criar chama colaboradorUC.BuscarPorId(dto.ColaboradorID, usuarioID) que falha (404) se liderado não existe ou não pertence ao líder dono.
- Parseamento de data/hora no fuso local (time.ParseInLocation): interpreta a data_hora fornecida (YYYY-MM-DDTHH:MM, YYYY-MM-DDTHH:MM:SS ou RFC3339) com time.Local. O DSN MySQL usa loc=Local automaticamente.
- Recorrência: obedece enum NENHUMA|SEMANAL|QUINZENAL|MENSAL. Padrão é NENHUMA. Validada com binding oneof no DTO.
- Scheduler (job de 24h): avança ocorrências passadas pela recorrência (AddDate para cada tipo), desativa agendamentos com recorrência NENHUMA já passados, agrupa por e-mail do gestor e envia um único e-mail com todos os 1:1 de hoje/amanhã (ItemLembrete com liderado_nome + "hoje/amanhã, HH:MM"). Não bloqueante de erro: loga falhas no envio.
- E-mail dormente se SMTP não configurado: o serviço (pkg/email) apenas loga mensagens não-críticas; permite boot sem AWS SES.
- Delete físico: agendamentos removidos via DELETE, não soft delete (sem deletado_em). Listagem filtra ativo=1.
- Reagendamento: valida data/hora com parsearDataHora; atualiza somente se WHERE id AND usuario_id retorna RowsAffected > 0 (posse garantida).

**Validações:**

- DTO CriarAgendamentoDTO: colaborador_id (binding required), data_hora (binding required), recorrencia (binding omitempty, oneof NENHUMA SEMANAL QUINZENAL MENSAL).
- Formato data_hora: parseado com parsearDataHora que tenta YYYY-MM-DDTHH:MM, YYYY-MM-DDTHH:MM:SS, RFC3339 nessa ordem. Erro fixo se nenhum formato coincide: 'data/hora inválida (use AAAA-MM-DDTHH:MM)'.
- Validação de liderado: BuscarPorId (do colaboradorUC) falha com 'liderado não encontrado' se ID não existe, foi deletado (deletado_em IS NOT NULL) ou não pertence ao gestor (verifica posse Cadeia B: equipe.usuario_id OU organizacao.usuario_id).
- Reagendar (PUT): data_hora é obrigatório no corpo (binding required). Retorna 404 'agendamento não encontrado' se reagendar() = (false, nil) via RowsAffected == 0.
- Deletar (DELETE): nenhuma validação no corpo (sem payload). Retorna erro genérico de banco se falhar.

#### 12.18 Notificação (sino + cron) (`notificacao`)

Notificações in-app (sino 🔔) geradas automaticamente pelo cron a partir da agenda de 1:1, com preferências por usuário para ligar/desligar cada tipo de aviso.

**Tabela(s):** **tb_notificacoes**: id (PK, VARCHAR36), usuario_id (FK, destinatário), tipo (VARCHAR30), titulo (VARCHAR160), mensagem (VARCHAR400), link (VARCHAR200, NULL), chave (VARCHAR200, UNIQUE — dedupe usuario|tipo|agendamento|ocorrência), lida (TINYINT, default 0), criado_em (DATETIME).

**tb_pref_notificacoes**: usuario_id (PK, VARCHAR36), agenda_1dia (TINYINT, default 1), agenda_hoje (TINYINT, default 1), agenda_1h (TINYINT, default 1), alterado_em (DATETIME).

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /notificacoes` | Lista as 30 notificações mais recentes do usuário do token (ordenadas por criado_em DESC). | usuário logado (token JWT) — cada um vê só as suas |
| `GET /notificacoes/contagem` | Retorna { "nao_lidas": n } — contagem de notificações não-lidas (badge do sino). | usuário logado |
| `PUT /notificacoes/itens/:id/lida` | Marca uma notificação específica como lida (atualiza lida=1 filtrando por id E usuario_id). | usuário logado — valida posse por usuario_id |
| `PUT /notificacoes/ler-todas` | Marca todas as notificações não-lidas do usuário como lidas em uma operação. | usuário logado |
| `GET /notificacoes/preferencias` | Retorna as preferências de notificação do usuário (agenda_1dia, agenda_hoje, agenda_1h — cada qual true/false). Sem registro = retorna padrão com tudo ligado. | usuário logado |
| `PUT /notificacoes/preferencias` | Salva (upsert via ON DUPLICATE KEY UPDATE) as preferências de notificação do usuário. Body: { "agenda_1dia": bool, "agenda_hoje": bool, "agenda_1h": bool }. | usuário logado |

**Regras de negócio:**

- Cron automático: `Scheduler.Iniciar()` executa uma vez ao boot e depois a cada 30 minutos (robusto a atraso/reinício).
- Faixa de avisos por data/hora da próxima ocorrência: AGENDA_1H (≤90 min), AGENDA_HOJE (mesmo dia e ≥6h da manhã), AGENDA_1DIA (próximo dia).
- Deduplicação: chave = usuario|tipo|agendamento|ocorrência (date only). INSERT IGNORE impede duplicata mesmo se cron rodar múltiplas vezes.
- Preferências: cada destinatário (gestor e liderado) precisa ter o tipo ligado em sua pref para a notificação ser criada. Sem registro = PrefPadrao (tudo ligado).
- Geração para ambos: gestor recebe em todo aviso; liderado recebe se colaborador.usuario_id != NULL e != '' (vínculo de conta estabelecido).
- Leitura da agenda pendente: JOIN agendamento → usuarios (gestor) → colaboradores (liderado). Filtra a.ativo=1 e deletado_em IS NULL.
- Posse simples: usuarioID do token é o único que pode marcar suas próprias notificações como lidas (filtro WHERE usuario_id = ? e WHERE id = ? AND usuario_id = ?).
- Preferências são upsert: insere com ON DUPLICATE KEY UPDATE, registro padrão retornado se não existe ainda.

**Validações:**

- PrefDTO valida via ShouldBindJSON (binding padrão do Gin) — todos os campos bool (agenda_1dia, agenda_hoje, agenda_1h).
- No endpoint PUT /preferencias, qualquer erro de binding retorna 400 'dados inválidos: <erro>'.
- IDs nas rotas (ex.: /itens/:id/lida) são passados como string (UUID), nenhuma validação de formato em controller (delegado ao repositório).
- Nenhum campo requerido (binding omite tags required); valores bool assumem false se ausentes no JSON.
- Marcar lida: id vem do path param, usuario_id vem do token (não é confiável vir do corpo).
- Todos os DTOs de resposta (NotificacaoRespostaDTO) oculta a chave de dedupe (não exposta ao cliente).

### IA & auditoria

#### 12.19 IA plugável (BYOK) (`ia`)

Registra trilha de atividades dos usuários: automaticamente por middleware em escrita (POST/PUT/DELETE) e sob demanda para eventos de UI do frontend.

**Tabela(s):** tb_auditoria: id (VARCHAR 36 PK), usuario_id (VARCHAR 36 NULL), acao (VARCHAR 50), entidade (VARCHAR 100), entidade_id (VARCHAR 36 NULL), ip (VARCHAR 45 NULL), user_agent (VARCHAR 255 NULL), criado_em (DATETIME). Índices: idx_auditoria_usuario, idx_auditoria_entidade (entidade+entidade_id), idx_auditoria_criado.

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /auditoria/eventos` | Frontend registra evento de UI (navegação, clique, visualização). Requer JWT. Payload: acao (obrigatório, max 50), entidade (obrigatório, max 100), entidade_id (opcional). Servidor preenche usuario_id, IP e User-Agent; responde 200 com 'evento registrado'. | Qualquer usuário autenticado |
| `GET /auditoria/minha` | Retorna últimos N eventos do usuário autenticado (mais recentes primeiro). Query param: limite (default 50, máximo 200). Responde com lista de AuditoriaRespostaDTO. | Qualquer usuário autenticado |
| `GET /colaboradores/:id/timeline` | Linha do tempo (eventos) de um liderado específico. Checa posse: colaborador deve pertencer ao líder autenticado via PertenceAoLider. Se não pertencer ou erro, responde 404. Responde com lista de eventos (max 100). | LÍDER dono do colaborador |

**Regras de negócio:**

- Gravação assíncrona em goroutine: Registrar() dispara insert em background, nunca bloqueia resposta HTTP, erros de banco são silenciosos (auditoria nunca quebra operação principal)
- ID gerado na aplicação: uuid.New().String() no momento do registro
- Carimbo de tempo: CriadoEm preenchido com time.Now()
- Normalização: IP e User-Agent vazios convertidos para nil (NULL no banco) via strPtr()
- Limite de paginação seguro: ListarPorUsuario e ListarPorEntidade forçam limite entre 1-200 para padrão 50, evita consultas sem teto
- Middleware global audita automaticamente POST/PUT/DELETE bem-sucedidos (status < 400), mapeia método → CRIAR/ATUALIZAR/DELETAR, derivar entidade e entidade_id do path
- Eventos POST /eventos explicitamente ignorados pelo middleware para evitar auditoria dupla
- Posse: GET /colaboradores/:id/timeline requer colaboradorUseCase.PertenceAoLider(colaboradorID, usuarioID); recurso alheio → 404
- Evento de UI sempre vinculado ao usuário autenticado: RegistrarEvento transforma usuarioID string em ponteiro para reaproveitar Registrar

**Validações:**

- EventoDTO.acao: required, max=50
- EventoDTO.entidade: required, max=100
- EventoDTO.entidade_id: omitempty (opcional)
- Query param limite em /auditoria/minha: convertido com strconv.Atoi, se inválido assume default 50
- Limite reforçado no UseCase: se <= 0 ou > 200, força para 50
- POST /eventos: usuario_id, IP, User-Agent lidos do contexto HTTP, nunca do JSON
- GET /colaboradores/:id/timeline: colaboradorID obrigatório no path; posse checada antes de listar

#### 12.20 Auditoria (`auditoria`)

Registra trilha de atividades dos usuários: automaticamente por middleware em escrita (POST/PUT/DELETE) e sob demanda para eventos de UI do frontend.

**Tabela(s):** tb_auditoria: id (VARCHAR 36 PK), usuario_id (VARCHAR 36 NULL), acao (VARCHAR 50), entidade (VARCHAR 100), entidade_id (VARCHAR 36 NULL), ip (VARCHAR 45 NULL), user_agent (VARCHAR 255 NULL), criado_em (DATETIME). Índices: idx_auditoria_usuario, idx_auditoria_entidade (entidade+entidade_id), idx_auditoria_criado.

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /auditoria/eventos` | Frontend registra evento de UI (navegação, clique, visualização). Requer JWT. Payload: acao (obrigatório, max 50), entidade (obrigatório, max 100), entidade_id (opcional). Servidor preenche usuario_id, IP e User-Agent; responde 200 com 'evento registrado'. | Qualquer usuário autenticado |
| `GET /auditoria/minha` | Retorna últimos N eventos do usuário autenticado (mais recentes primeiro). Query param: limite (default 50, máximo 200). Responde com lista de AuditoriaRespostaDTO. | Qualquer usuário autenticado |
| `GET /colaboradores/:id/timeline` | Linha do tempo (eventos) de um liderado específico. Checa posse: colaborador deve pertencer ao líder autenticado via PertenceAoLider. Se não pertencer ou erro, responde 404. Responde com lista de eventos (max 100). | LÍDER dono do colaborador |

**Regras de negócio:**

- Gravação assíncrona em goroutine: Registrar() dispara insert em background, nunca bloqueia resposta HTTP, erros de banco são silenciosos (auditoria nunca quebra operação principal)
- ID gerado na aplicação: uuid.New().String() no momento do registro
- Carimbo de tempo: CriadoEm preenchido com time.Now()
- Normalização: IP e User-Agent vazios convertidos para nil (NULL no banco) via strPtr()
- Limite de paginação seguro: ListarPorUsuario e ListarPorEntidade forçam limite entre 1-200 para padrão 50, evita consultas sem teto
- Middleware global audita automaticamente POST/PUT/DELETE bem-sucedidos (status < 400), mapeia método → CRIAR/ATUALIZAR/DELETAR, derivar entidade e entidade_id do path
- Eventos POST /eventos explicitamente ignorados pelo middleware para evitar auditoria dupla
- Posse: GET /colaboradores/:id/timeline requer colaboradorUseCase.PertenceAoLider(colaboradorID, usuarioID); recurso alheio → 404
- Evento de UI sempre vinculado ao usuário autenticado: RegistrarEvento transforma usuarioID string em ponteiro para reaproveitar Registrar

**Validações:**

- EventoDTO.acao: required, max=50
- EventoDTO.entidade: required, max=100
- EventoDTO.entidade_id: omitempty (opcional)
- Query param limite em /auditoria/minha: convertido com strconv.Atoi, se inválido assume default 50
- Limite reforçado no UseCase: se <= 0 ou > 200, força para 50
- POST /eventos: usuario_id, IP, User-Agent lidos do contexto HTTP, nunca do JSON
- GET /colaboradores/:id/timeline: colaboradorID obrigatório no path; posse checada antes de listar

### Plataforma, ajuda & recuperação

#### 12.22 Recuperação de senha (`recuperacao`) — atualizado

Fluxo público "esqueci minha senha" (link + código de 6 dígitos). **Validade reduzida para
15 minutos** (`const validadeLink`); o e-mail renderiza o código **uma caixa por dígito**.

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /auth/recuperar-senha` | Pede o link (sempre responde igual — anti-enumeração) | público (rate-limit + reCAPTCHA) |
| `GET /recuperacoes/:token` | Diz se o link ainda é válido (`{valido}`) | público |
| `POST /recuperacoes/:token/redefinir` | Valida token+código e troca a senha | público |

**Regras:** validade 15 min; uso único; código só em hash bcrypt; máx. 5 tentativas de
código erradas invalidam o token; trocar a senha revoga as sessões (token_version).
Detalhe em `internal/recuperacao/README.md`.

#### 12.23 Central de Ajuda (`ajuda`)

Ajuda para todos os usuários: **conteúdo curado** (tópicos + tour, funcionam sem IA) +
**assistente de IA** que resolve a chave em cascata **plataforma → BYOK → curado**.

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /ajuda/topicos` | Tópicos visíveis para o papel do usuário | JWT |
| `GET /ajuda/topicos/:id` | Um tópico (conteúdo em markdown) | JWT |
| `GET /ajuda/tour` | Tour de boas-vindas por papel | JWT |
| `GET /ajuda/ia/status` | `{ia_disponivel}` para mostrar/ocultar o chat | JWT |
| `POST /ajuda/perguntar` | Pergunta livre → resposta da IA | JWT + rate-limit próprio |

**Regras:** chave de plataforma (`IA_PLATAFORMA_*`) atende qualquer usuário; senão usa a IA
BYOK do gestor/RH; senão devolve mensagem amigável. Pergunta máx. 1000 chars; não auditada;
nunca vaza erro técnico nem a chave. Conteúdo curado em `conteudo.go`.

#### 12.24 Painel ADMIN da plataforma (`admin`)

Monitoração global, **só leitura agregada**, exclusiva da conta **ADMIN** (papel novo,
migration 023). Sem tabela nova — agrega `tb_usuarios`, `tb_auditoria`, `tb_onebyone`,
`tb_agendamentos` e a estrutura.

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /admin/visao-geral` | KPIs: contas por papel, estrutura, atividade (DAU/WAU/MAU, logins, 1:1) | JWT + ApenasAdmin |
| `GET /admin/contas` | Lista paginada de contas com resumo de uso (filtros papel/busca) | JWT + ApenasAdmin |
| `GET /admin/acessos` | Série temporal (estilo Google Analytics): logins/ativos/eventos por dia | JWT + ApenasAdmin |
| `GET /admin/uso` | Top funcionalidades + por hora + por dia da semana + por papel | JWT + ApenasAdmin |
| `GET /admin/crescimento` | Novos cadastros por dia/papel + acumulado + 1:1 realizados | JWT + ApenasAdmin |
| `GET /admin/saude` | Engajamento/adoção + ranking de gestores | JWT + ApenasAdmin |

**Regras:** `ApenasAdmin` protege todo o grupo; o login agora é **atribuído ao usuário** na
auditoria (analytics de acessos fiéis); a conta admin é garantida no boot
(`GarantirContaAdmin`, e-mail `ADMIN_EMAIL`); parâmetros `dias`/`limite` são limitados;
séries vêm com buracos preenchidos com zero. Detalhe em `internal/admin/README.md`.

#### 12.25 Feedback dos usuários (`feedback`)

Reações rápidas (curti / não curti / irritado) com contexto e comentário opcionais —
coleta a satisfação dos usuários e leva ao dashboard de gestão. Tabela `tb_feedbacks`
(migration **024**, collation fixada em utf8mb4_unicode_ci para o JOIN com tb_usuarios).

| Rota | O que faz | Acesso |
|---|---|---|
| `POST /feedback` | Registra uma reação (`reacao` + `contexto`/`comentario`/`pagina` opcionais) | JWT (qualquer papel) |
| `GET /admin/feedbacks` | Painel: totais, índice de satisfação, série por reação, por contexto, comentários recentes | JWT + ApenasAdmin |

**Regras:** `usuario_id` vem do JWT (nunca do corpo); `reacao` ∈ {CURTI, NAO_CURTI,
IRRITADO}; log append-only (não é toggle); reação **não** é auditada (tem tabela própria);
série com buracos preenchidos com zero; `recentes` traz só feedbacks com comentário, já com
o autor. Detalhe em `internal/feedback/README.md`.

### Infraestrutura & fiação

#### 12.21 Infraestrutura (pkg/) e fiação (cmd/api/) (`infra`)

Core transversal: autenticação JWT com injeção de usuario_id no contexto, middleware de auditoria automática, envelope padronizado de respostas HTTP, integração com S3 para arquivos privados, envio de e-mail SMTP (dormente), cifragem AES-GCM de segredos, pool de conexões MySQL, e schedulers de lembrete/notificação.

**Tabela(s):** **pkg/config**: Struct `Config` com 20+ variáveis de ambiente (DB_HOST/USER/PASSWORD/NAME, JWT_SECRET/EXPIRACAO_HORAS, AWS_ACCESS_KEY_ID/SECRET_ACCESS_KEY/REGION/BUCKET/PREFIXO, SMTP_HOST/PORT/USER/PASSWORD/REMETENTE, PORTA_API, APP_URL). Defaults: localhost:3306, DB=onebyone, JWT=24h, porta=8080, S3=us-east-1/controleazul/one-by-one, APP_URL=http://localhost:3100.\n\n**pkg/database**: `*sqlx.DB` com DSN charset=utf8mb4, parseTime=true, loc=Local. Conexão validada via ping ao iniciar. Pool reutilizado por todos os repositórios.\n\n**pkg/middleware**: ClaimsJWT (usuario_id, role). ChaveUsuarioID/ChaveUsuarioRole constantes. AutenticarJWT extrai Bearer token, valida HMAC, injeta contexto. ApenasLider checa role==LIDER ou aborta. RegistrarAuditoria (POST/PUT/DELETE) grava ação/entidade/usuario_id/IP/User-Agent.\n\n**pkg/response**: RespostaPadrao (sucesso, dados, erro). Helpers: Sucesso(200), Criado(201), ErroInterno(500), ErroNaoEncontrado(404), ErroRequisicao(400), ErroNaoAutorizado(401), ErroProibido(403), ErroConflito(409).\n\n**pkg/storage**: `Armazenamento` interface (Upload, GerarURLPresignada, Deletar, ChaveCompleta). Impl S3: bucket+prefixo, ExpiracaoURLFoto=2h, acesso via PresignGetObject, sem ACL pública.\n\n**pkg/email**: `Servico` interface (EnviarHTML, Configurado). Impl SMTP: PlainAuth, montarMIME. Dormente se SMTPHost vazio. Templates: BoasVindas (nome, appURL), Convite (nomeLiderado, link, codigo), Lembrete (nomeGestor, []ItemLembrete{Liderado, Quando}).\n\n**pkg/cripto**: Cifrar/Decifrar AES-256-GCM. Chave derivada SHA-256(segredo). Retorno base64(nonce+ciphertext)."

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /health` | Verificação simples de que a API está no ar. Retorna status e versão. | público |
| `GET /swagger/index.html` | Documentação interativa dos endpoints (gerada por swaggo a partir dos comentários // @... do código). | público |
| `Middleware: AutenticarJWT` | Valida o token Bearer no cabeçalho Authorization. Injeita usuario_id e usuario_role no contexto. Tokens inválidos/expirados retornam 401 e interrompem a requisição. | aplicado globalmente em /api/v1 |
| `Middleware: ApenasLider` | Restringe acesso a usuários com role LIDER. Defesa em profundidade para rotas de gestão (criar/editar/deletar). Deve ser usado APÓS AutenticarJWT. Retorna 403 se acesso negado. | LIDER |
| `Middleware: RegistrarAuditoria` | Grava automaticamente requisições POST/PUT/DELETE bem-sucedidas (status < 400) na trilha de auditoria. Extrai ação (CRIAR/ATUALIZAR/DELETAR), entidade, usuario_id do contexto, IP e User-Agent. | aplicado globalmente em /api/v1 |
| `Serviço de e-mail (SMTP)` | Envio de e-mails HTML. Dormente se SMTP_HOST/SMTP_REMETENTE não estiverem no .env (apenas loga). Templates: TemplateBoasVindas, TemplateConvite, TemplateLembrete. Suporta múltiplos destinatários. | interno |
| `Serviço de armazenamento (S3)` | Upload, geração de URL presignada (2h de validade padrão), e deleção de arquivos no AWS S3. Objetos sempre privados, acesso via URL assinada. Chaves prefixadas com AWS_PREFIXO para isolamento no bucket. | interno |
| `Criptografia simétrica (AES-256-GCM)` | Cifra e decifra segredos sensíveis (ex.: chave de IA do gestor BYOK). Chave derivada de SHA-256 do JWT_SECRET. Valor guardado no banco (base64) é inútil sem o segredo do servidor. | interno |
| `Scheduler: agendamento` | Roda a cada 24 horas: avança ocorrências recorrentes que já passaram e envia 1x/dia e-mail com 1:1 de HOJE e AMANHÃ para cada gestor. Tipo de recorrência: NENHUMA, SEMANAL, QUINZENAL, MENSAL. | background |
| `Scheduler: notificação` | Roda a cada 30 minutos: gera avisos in-app por FAIXA (1 dia antes, hoje de manhã, ~1h antes). Deduplicado por chave (usuario+tipo+agendamento+ocorrência). Respeita preferências do usuário. Gera para gestor e liderado. | background |

**Regras de negócio:**

- JWT_SECRET é obrigatório para assinar tokens; se vazio, a API falha ao carregar
- Token JWT contém usuario_id (UUID) e role (LIDER ou COLABORADOR) injetados no contexto Gin via middleware
- Auditoria automática: POST/PUT/DELETE com status < 400 são gravadas com ação/entidade/usuario_id/IP/User-Agent; GET ignorado a menos que seja login
- Cifragem de IA (BYOK): chave de API cifrada com AES-256-GCM, segredo = JWT_SECRET, armazenada em base64 no banco
- S3 URL presignada padrão: 2 horas de validade; frontend deve renovar chamando a API quando expirar
- E-mail dormente: se SMTP não configurado (host vazio), EnviarHTML apenas loga sem falhar; aplicação continua funcionando
- Scheduler agendamento: executa a cada 24h ao subir e depois periodicamente; avança recorrências atrasadas, envia lembrete 1x/dia
- Scheduler notificação: executa a cada 30 min; gera avisos por faixa (1 dia, hoje de manhã, ~1h antes) dedupados por chave única, respeitando preferências
- Posse via JWT: usuario_id do token é a fonte de verdade; nunca confiar em usuario_id do corpo/DTO
- Injeção de dependências manual: Repository → UseCase → Controller → rotas, ordem importa se houver dependência entre módulos (ex.: registroonebyone depende de onebyone)

**Validações:**

- Config.Carregar(): JWT_EXPIRACAO_HORAS convertido para int, default 24 se inválido
- middleware.AutenticarJWT: cabeçalho Authorization ausente → 401; formato não 'Bearer <token>' → 401; token inválido ou expirado → 401; rejeita algo diferente de HMAC para evitar ataque de troca de algoritmo
- middleware.ApenasLider: role != 'LIDER' → 403 'acesso restrito a líderes'
- middleware.RegistrarAuditoria: ignora requisições com status >= 400 (erros não são auditados); entidade extraída do path, fallback para segmento genérico
- email.Configurado(): retorna false se host vazio OU remetente vazio
- cripto.Cifrar/Decifrar: base64 inválido → erro; nonce insuficiente → erro; falha GCM na abertura → erro 'erro ao decifrar'
- storage.Upload: tipo de conteúdo informado, tamanho em bytes, não há ACL pública
- database.NovaConexao: ping obrigatório para validar conectividade ao MySQL

### Frontend (onebyone-app)

#### 12.22 Frontend — telas e fluxos (`frontend`)

Sistema completo de gestão de reuniões one-on-one com interface moderna (tema Encanto), suportando pauta interativa (drag-drop), acompanhamento de liderados (humor, entregas, feedbacks, PDI), matrix 9-box, agenda com recorrência, editor de conteúdo ao vivo via WebSocket, IA contextual (BYOK), timeline de auditoria e sino de notificações.

**Tabela(s):** Sem tabelas de banco de dados (frontend SPA). Dados vêm da API backend Go (/home/ubuntu/onebyone/onebyone-api). Componentes estruturados em 14 subcategorias: pauta, acompanhamento, pdi, matrix, agenda, estrutura, conteúdo, auditoria, notificação, ia, aovivo, painel, ui, marca. Todas as rotas protegidas via JWT (AuthContext).

| Tela/fluxo | O que faz | Acesso |
|---|---|---|
| `/` | Landing page público com hero, recursos, passos onboarding, benefícios (gestor/liderado), 1:1 explicado, FAQ, rodapé — tema Encanto (claro premium), sem cara de IA | público |
| `/entrar` | Login com email/senha, visual encorpado com campos grandes, botão Duo 3D, erro geral, link para criar conta | público |
| `/criar-conta` | Onboarding gestor em 3 passos (conta → equipes → liderados), cada passo grava de verdade na API, comemora com fogos ao fim | público |
| `/convite/:token` | Aceite de convite do liderado: email pré-preenchido, código 6-char segmentado, senha nova, comemora ao aceitar, já entra logado | público |
| `/painel` | Hub principal gestor: onboarding (criar org) → dashboard (pulso do time, lembretes, construtor de estrutura drag-drop equipes+liderados, agenda embutida, matrix9-box, painel de acompanhamento). Visão diferente se liderado logado | LIDER / COLABORADOR |
| `/painel (COLABORADOR)` | Espaço do liderado: boas-vindas, link para abrir seu 1:1 ao vivo (via meu-colaborador da API) | COLABORADOR |
| `/onebyone` | Hub 'Com quem é o 1:1 de hoje?' — tabuleiro com chips clicáveis de cada liderado ativo (rota para /liderado/:id), empty state se sem time | LIDER |
| `/liderado/:id` | O 1:1 ao vivo em 3D: header (dupla + presença + ao vivo), pauta com 3 colunas (banco → pauta → conversado), drag-drop de temas, editor lateral (drawer) por tema (texto/link/imagem/marco), modo apresentação fullscreen, cursores ao vivo, botão encerrar (resumo + próximos passos → histórico) | LIDER / COLABORADOR (mesma sala WebSocket) |
| `/agenda` | Calendário estilo Google: grid mensal, criar/remarcar/cancelar 1:1 com recorrência (semanal/quinzenal/mensal), drag chips entre dias, modal de detalhe, modal de novo agendamento com liderado/hora/recorrência | LIDER |
| `/matrix9-box` | Matriz 9×9 (desempenho × potencial) interativa: arraste liderados entre quadrantes, reposicionamento otimista com transição, filtro por equipe, export PDF, rótulos (⭐ Estrela, ⚠️ Atenção, etc), cores de zona por score | LIDER |
| `/perfil` | Página pessoal: foto (clicável, upload S3 com validação 5MB/imagem), nome/email, ConfigIA (para gestor, BYOK com chave própria) | LIDER / COLABORADOR |
| `ConstrutorEstrutura (componente)` | Painel de gestão de equipes+liderados: equipes em colunas, liderados como chips arrastáveis (reordenação = mover para outra equipe), adicionar equipe/liderado inline, ações (convidar, desligar, reativar, foto, acompanhamento 📊, PDI, IA, timeline), visualização cartões/lista (localStorage), monitor Kanban fullscreen | LIDER |
| `TabuleiroPauta (componente)` | O tabuleiro do 1:1: 3 colunas (banco/pauta/conversado), drag-drop com dnd-kit, feedback ao vivo (tema move entre colunas enquanto arrasta), tema clicável abre editor, sensores mouse/teclado (acessibilidade) | LIDER / COLABORADOR (ao vivo) |
| `TemaEditor (drawer lateral)` | Editor de conteúdo de tema: lista blocos existentes, +TEXTO (multiline), +LINK/CURSO (URL + título), +IMAGEM (upload S3), +MARCO (data início/fim), deletar blocos, botão ▶ Apresentar, recarrega se sinal ao vivo de outro participante | LIDER / COLABORADOR |
| `ApresentacaoTema (fullscreen)` | Modo apresentação tela cheia de um tema: blocos formatados (TEXTO com quebra, LINK expandido, IMAGEM + legenda, MARCO com datas), sincronizado ao vivo (WebSocket), botão voltar | LIDER / COLABORADOR (ao vivo) |
| `EncerrarOneByOne (modal)` | Ritual do encerramento: campo texto resumo + lista dinâmica de 'próximos passos' (+ item), valida se tem algo, salva no tema histórico (📋 Histórico de 1:1), comemora com fogos, sincroniza ao vivo (os dois caem em modo consulta) | LIDER |
| `PainelLiderado (drawer lateral)` | Acompanhamento completo: resumo PDI em anel (concluídos/total %), abas (Sentimento/Entregas/Feedbacks/Estudos), registro inline de cada tipo, gráfico de humor semanal, relatório agregado | LIDER |
| `PdiLiderado (drawer lateral)` | PDI do liderado: anel visual com progresso (%), lista ordenada (pendentes 1º, concluídos 2º), status de prazo (atrasado/vence hoje/faltam Xd), +objetivo, marca concluído (stroke), remove, IA sugere objetivos (chat contextual) | LIDER |
| `TimelineLiderado (drawer lateral)` | Linha do tempo horizontal: eventos de auditoria (criar tema, deletar bloco, 9-box, convite, foto, dados, desligamento, reativação) com emoji, rótulo amigável, data/hora, cor por categoria | LIDER |
| `PainelIALiderado (drawer lateral)` | IA contextual (BYOK): resume o liderado (9-box + todos os blocos de temas), 3 ações (overview, sugerir pauta, rascunho feedback/PDI) via chat, stream de resposta, reusa contexto | LIDER |
| `SinoNotificacoes (header)` | Sino in-app: badge com contador, dropdown com lista de notificações (não-lidas animadas), marca como lida ao clicar (navega pelo link), engrenagem ⚙️ para preferências (desligar tipos), tempo relativo (agora/há 5min/ontem) | LIDER / COLABORADOR |
| `ConfigIA (perfil)` | Entrada de chave OpenAI do gestor (BYOK), salva em segurança, aviso de uso | LIDER |
| `ModalConvite (painel)` | Gerador de link + código 6-char (contra-senha) para convidar liderado, botões copiar com feedback ✓, aviso de expiração 7d | LIDER |
| `CursoresAoVivo (overlay)` | Cursores em tempo real de outros participantes da sala (WebSocket), com avatar + nome, throttle 45ms para não travar | LIDER / COLABORADOR |
| `MatrixNineBox (componente)` | Versão embutida no painel: mesma matriz 9×9, reusa estado/drag/classificações, link para abrir página dedica | LIDER |
| `PulsoTime (painel)` | Resumo do time em 4 cartões: humor médio (ultmos check-ins), atividade recente 30d (entregas/feedbacks/estudos), PDI agregado (concluídos/total), faltam convidar | LIDER |
| `Lembretes (painel)` | O que precisa atenção agora: 1:1 de hoje, convites pendentes, PDI vencendo, liderados sem convidar | LIDER |
| `AgendaPainel (painel)` | Bloco de calendário mini (reutiliza Calendario): clique no 1:1 abre /liderado/:id direto, clique dia → /agenda completa, drag remarcar | LIDER |
| `Calendario (componente)` | Calendário com grid mensal, eventos (1:1 agendados como chips), click dia abre modal novo, click chip abre detalhe (abrir 1:1/cancelar), drag chip remarcar, indicador de recorrência | LIDER |
| `Modal / Drawer / Confirmacao (UI)` | Modal: cartão central com corpo, overlay blur, ESC fecha. Drawer: painel lateral da direita, animação slide, ESC fecha (controla aberto=true). Confirmacao: substitui confirm() nativo, emoji grande, botões estilo Duolingo, perigoso→alerta | global |
| `useConfirmar (hook UI)` | Hook Promise-based para confirmações: não usa alert/confirm nativo, abre modal temático, retorna boolean | global |
| `Campo (componente)` | Input controlado com rótulo, erro em vermelho, ícone opcional, tamanho (normal/grande), somente-leitura, tipos (text/email/password), autoComplete | formulários |
| `Botao / BotaoDuo (componente)` | Botao: simples, variantes (marca/contorno/primario/sucesso), carregando. BotaoDuo: botão 3D estilo Duolingo com borda inferior (afunda ao clicar) | global |
| `CodigoSegmentado (componente)` | Entrada de 6 caracteres para convite: campos separados, navegação automática (preenche → next), visual bonito | convite |
| `AvatarUsuario (componente)` | Avatar com foto URL ou initial do nome, tamanho dinâmico, shadow | global |
| `SeletorTema (flutuante)` | Botão flutuante (canto inferior direito) para trocar entre claro/escuro — global em todas as telas | global |
| `FundoVivo (background)` | Bolhas animadas subindo ao fundo (baixo z-index), acompanha tema claro/escuro | global |
| `LayoutApp (estrutura)` | Moldura das telas internas: header sticky com logo, nav (desktop em linha / mobile menu hambúrguer Drawer), sino notif, assistente IA, usuário logado, sair. Responsivo (px-4 mobile, px-8 desktop) | autenticado |
| `LayoutAuth (estrutura)` | Moldura login/registro/convite: logo, chamada visual à direita, form à esquerda, background decorativo | público |
| `AssistenteIA (header)` | Botão IA flutuante no header (quando configurado), abre chat genérico ou contextual do liderado | LIDER |
| `Ajuda (tooltip)` | Ícone ❓ com tooltip explicativo ao hover (popper posicionado inteligente, pode alinhar esquerda/direita/centro) | global |

**Regras de negócio:**

- Sem alert/confirm nativo — sempre usar useConfirmar() que abre modal temático
- Mobile prioriza botões em vez de drag: Matrix tem botão 📍 no celular (select 3×3) em vez de arrastar
- Visualização (cartões/lista) do construtor persiste em localStorage por sessão do usuário
- Tema 'Encanto' (claro premium): cores suave (tinta, tinta-suave, juncao, sucesso, alerta, gestor, liderado, borda), sem feel de IA/tech
- Tabuleiro ao vivo: throttle drag 120ms (tabela), throttle cursor 45ms (não travar), aplicandoRemoto evita eco WebSocket
- Pauta salva com debounce 800ms ao backend — sobrevive recarregar
- Sinal ao vivo de tema mudado: recarrega blocos do TemaEditor se tema bate
- Apresentação fullscreen: sincronizada ao vivo entre os dois (WebSocket aoApresentacao)
- Encerramento 1:1: grava no tema histórico (📋 Histórico de 1:1), marca os DOIS em modo consulta (somente leitura, sombra no tabuleiro, pointer-events-none)
- 9-box: reposicionamento otimista (sem snap-back), transição suave, filtro por equipe, export PDF
- Humor/sentimento: índice 1-5, emojis mapeados (😞 1, 😕 2, 😐 3, 🙂 4, 😄 5), gráfico semanal
- PDI ordenado: pendentes 1º (prazo mais próximo), concluídos 2º, status visual (atrasado/vence hoje/faltam Xd)
- Timeline: eventos trad. com emoji+cor, mais antigo → mais recente
- Notificação: tempo relativo (agora/há X min/h/d/data), marca lida ao clicar, lista só busca quando painel aberto (lazy)
- Convite: link + código 6-char (contra-senha), expira 7d, já entra logado ao aceitar
- Agendamento: recorrência (nenhuma/semanal/quinzenal/mensal), lembretes por e-mail (quando SMTP ligado)
- E-mail duplicado em colaborador: bloqueia (msg 'já existe um liderado com este e-mail neste time'); e-mail do gestor: bloqueia no convite (anti-sequestro)
- Estrutura drag-drop equipes: Pointer sensor com 6px de distância (cliques não viram arraste)
- Onboarding registro: 3 passos com barra de progresso, cada passo grava de verdade, comemora com fogos
- Convite aceito: comemora com fogos 3s, redireciona /painel
- IA contextual: BYOK (gestor entra chave OpenAI própria), resume tudo (9-box + blocos), 3 ações (overview/pauta/feedback)
- Sem cara de ERP: drawers em vez de modals (preferência do dono), animações suaves, tons acolhedores
- Cache com React Query: invalidações automáticas pós-mutação, queries habilitadas condicionalmente (enabled: Boolean(id))
- Otimismo: não há rollback explícito (mutações assumem sucesso), erros mostram mensagem após falha
- Acessibilidade: useId para label↔input, role/aria, teclado (setas no drag, Esc fecha, Enter submit)

**Validações:**

- Email: regex simples /^[^\s@]+@[^\s@]+\.[^\s@]+$/ em tempo real no input (sinal vermelho)
- Senha: mínimo 6 caracteres (sugestão no placeholder, sem validação hard no frontend)
- Nome: trim, não-vazio (validações no Controller do backend)
- Foto: tipos (JPEG/PNG/WebP) + max 5MB — validado antes do upload (erro exibido)
- Campos obrigatórios: renderizam * ou bloqueiam botão (disabled) se vazio
- URL (link/curso): aceita qualquer URL com http(s):// ou schema relativo
- Data: input type='date' (browser fornece validação), formato YYYY-MM-DD, sem futuro se data de prazo
- Seletor de equipe: desabilitado se só 1 equipe (força a única automaticamente)
- Código segmentado (convite): 6 chars, auto-move se preenchido, valida ao submit
- Recorrência agendamento: enum (NENHUMA/SEMANAL/QUINZENAL/MENSAL), padrão SEMANAL
- Confirmação perigosa (deletar, desligar): modal com emoji ⚠️ e botão em alerta, autoFocus=true no cancelar
- Erro de API: extraído via extrairMensagemErro() — exibido em box alerta/10 border alerta/30
- Carregamento: estado isPending/isLoading nas queries/mutations, botões ganham atributo carregando (visual + disabled)
- Humores: enum [1, 2, 3, 4, 5], exibidos com emoji, não aceitam 0 ou valores fora do intervalo
