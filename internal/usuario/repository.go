// Pacote: internal/usuario
// Arquivo: repository.go
// Descrição: Define a interface Repositorio e sua implementação MySQL para
//            a entidade Usuario. Toda interação com a tabela tb_usuarios
//            passa exclusivamente por este arquivo.
// Autor: OneByOne API
// Criado em: 2025

package usuario

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repositorio define o contrato de acesso ao banco de dados para a entidade Usuario.
// O UseCase depende desta interface, não da implementação concreta, facilitando testes.
type Repositorio interface {
	// Criar insere um novo usuário na tabela tb_usuarios e retorna o registro criado
	Criar(usuario Usuario) (Usuario, error)
	// BuscarPorId retorna um usuário ativo (não deletado) pelo seu UUID
	BuscarPorId(id string) (Usuario, error)
	// BuscarPorEmail retorna um usuário ativo pelo seu endereço de e-mail
	BuscarPorEmail(email string) (Usuario, error)
	// Listar retorna todos os usuários ativos ordenados por nome
	Listar() ([]Usuario, error)
	// Atualizar aplica as alterações em um usuário existente e retorna o registro atualizado
	Atualizar(usuario Usuario) (Usuario, error)
	// DeletarSoft preenche deletado_em e deletado_por sem remover o registro fisicamente
	DeletarSoft(id string, deletadoPor string) error
	// AtualizarFoto persiste a chave S3 da foto no banco de dados
	AtualizarFoto(id string, fotoKey string) error
	// AtualizarSenha troca o hash da senha de um usuário (fluxo de recuperação de senha)
	AtualizarSenha(id string, passwordHash string) error
	// RegistrarFalhaLogin incrementa as falhas e bloqueia ao atingir o limite (atômico)
	RegistrarFalhaLogin(id string, maxFalhas, bloqueioMin int) error
	// ZerarFalhaLogin limpa o contador e o bloqueio (login bem-sucedido)
	ZerarFalhaLogin(id string) error
	// IncrementarVersaoToken invalida todos os tokens atuais do usuário (revogação)
	IncrementarVersaoToken(id string) error
}

// repositorioMySQL é a implementação concreta do Repositorio usando MySQL via sqlx
type repositorioMySQL struct {
	// db é o pool de conexões com o banco de dados
	db *sqlx.DB
}

// NovoRepositorio cria e retorna uma instância do repositório MySQL de usuários
func NovoRepositorio(db *sqlx.DB) Repositorio {
	return &repositorioMySQL{db: db}
}

// AtualizarSenha troca o hash da senha de um usuário ativo (fluxo de recuperação).
func (r *repositorioMySQL) AtualizarSenha(id string, passwordHash string) error {
	res, err := r.db.Exec(`UPDATE tb_usuarios SET password = ? WHERE id = ? AND deletado_em IS NULL`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("erro ao atualizar senha: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("usuário não encontrado")
	}
	return nil
}

// RegistrarFalhaLogin incrementa as falhas e, ao cruzar o limite, zera o contador e bloqueia
// a conta — tudo numa única instrução atômica (sem read-modify-write, à prova de corrida sob
// requisições paralelas). Ao bloquear, zera o contador para dar uma janela limpa após a espera.
func (r *repositorioMySQL) RegistrarFalhaLogin(id string, maxFalhas, bloqueioMin int) error {
	const q = `
		UPDATE tb_usuarios
		SET tentativas_login = CASE WHEN tentativas_login + 1 >= ? THEN 0 ELSE tentativas_login + 1 END,
		    bloqueado_ate     = CASE WHEN tentativas_login + 1 >= ? THEN DATE_ADD(NOW(), INTERVAL ? MINUTE) ELSE bloqueado_ate END
		WHERE id = ?`
	if _, err := r.db.Exec(q, maxFalhas, maxFalhas, bloqueioMin, id); err != nil {
		return fmt.Errorf("erro ao registrar falha de login: %w", err)
	}
	return nil
}

// ZerarFalhaLogin limpa o contador de falhas e o bloqueio (login bem-sucedido).
func (r *repositorioMySQL) ZerarFalhaLogin(id string) error {
	if _, err := r.db.Exec(`UPDATE tb_usuarios SET tentativas_login = 0, bloqueado_ate = NULL WHERE id = ?`, id); err != nil {
		return fmt.Errorf("erro ao zerar falhas de login: %w", err)
	}
	return nil
}

// IncrementarVersaoToken soma 1 à versão do token, invalidando todas as sessões anteriores.
func (r *repositorioMySQL) IncrementarVersaoToken(id string) error {
	if _, err := r.db.Exec(`UPDATE tb_usuarios SET token_version = token_version + 1 WHERE id = ?`, id); err != nil {
		return fmt.Errorf("erro ao incrementar versão do token: %w", err)
	}
	return nil
}

// Criar insere um novo registro em tb_usuarios e retorna o usuário persistido.
// O UUID e o hash da senha devem ser gerados antes de chamar este método.
func (r *repositorioMySQL) Criar(usuario Usuario) (Usuario, error) {
	// rh_id é nil para gestor solo / RH raiz / liderado; preenchido só quando um RH
	// cadastra um gestor (CriarGestorParaRH). NamedExec mapeia :rh_id a partir do
	// campo RhID (nulo → NULL). Requer a coluna rh_id (migration 017).
	query := `
		INSERT INTO tb_usuarios (id, nome, email, password, role, rh_id, criado_em)
		VALUES (:id, :nome, :email, :password, :role, :rh_id, :criado_em)
	`

	if _, err := r.db.NamedExec(query, usuario); err != nil {
		return Usuario{}, fmt.Errorf("erro ao inserir usuário no banco: %w", err)
	}

	// Busca o registro recém-criado para retornar os dados completos com defaults do banco
	return r.BuscarPorId(usuario.ID)
}

// BuscarPorId retorna o usuário ativo correspondente ao UUID informado.
// Retorna erro se o usuário não existir ou estiver deletado logicamente.
func (r *repositorioMySQL) BuscarPorId(id string) (Usuario, error) {
	var u Usuario

	// A cláusula deletado_em IS NULL garante que registros deletados não sejam retornados
	query := `
		SELECT id, nome, email, password, role, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_usuarios
		WHERE id = ? AND deletado_em IS NULL
	`

	if err := r.db.Get(&u, query, id); err != nil {
		return Usuario{}, fmt.Errorf("usuário não encontrado com id '%s': %w", id, err)
	}

	return u, nil
}

// BuscarPorEmail retorna o usuário ativo com o e-mail informado.
// Usado tanto no login quanto na verificação de duplicidade ao criar/atualizar.
func (r *repositorioMySQL) BuscarPorEmail(email string) (Usuario, error) {
	var u Usuario

	// A cláusula deletado_em IS NULL garante que e-mails de usuários deletados
	// possam ser reutilizados por novos cadastros
	query := `
		SELECT id, nome, email, password, token_version, tentativas_login, bloqueado_ate, role, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_usuarios
		WHERE email = ? AND deletado_em IS NULL
	`

	if err := r.db.Get(&u, query, email); err != nil {
		return Usuario{}, fmt.Errorf("usuário não encontrado com email '%s': %w", email, err)
	}

	return u, nil
}

// Listar retorna todos os usuários ativos do sistema, ordenados por nome para facilitar exibição.
func (r *repositorioMySQL) Listar() ([]Usuario, error) {
	var usuarios []Usuario

	query := `
		SELECT id, nome, email, password, role, foto_key, criado_em, alterado_em, deletado_em, deletado_por
		FROM tb_usuarios
		WHERE deletado_em IS NULL
		ORDER BY nome ASC
	`

	if err := r.db.Select(&usuarios, query); err != nil {
		return nil, fmt.Errorf("erro ao listar usuários: %w", err)
	}

	return usuarios, nil
}

// Atualizar aplica as modificações no registro do usuário informado e retorna os dados atualizados.
// O campo alterado_em é preenchido automaticamente com o timestamp atual.
func (r *repositorioMySQL) Atualizar(usuario Usuario) (Usuario, error) {
	// Registra o momento exato da alteração antes de persistir
	agora := time.Now()
	usuario.AlteradoEm = &agora

	query := `
		UPDATE tb_usuarios
		SET nome = :nome, email = :email, role = :role, alterado_em = :alterado_em
		WHERE id = :id AND deletado_em IS NULL
	`

	if _, err := r.db.NamedExec(query, usuario); err != nil {
		return Usuario{}, fmt.Errorf("erro ao atualizar usuário: %w", err)
	}

	return r.BuscarPorId(usuario.ID)
}

// DeletarSoft realiza a exclusão lógica preenchendo deletado_em com o timestamp atual
// e deletado_por com o ID do usuário responsável pela ação. O registro permanece no banco.
func (r *repositorioMySQL) DeletarSoft(id string, deletadoPor string) error {
	agora := time.Now()

	query := `
		UPDATE tb_usuarios
		SET deletado_em = ?, deletado_por = ?
		WHERE id = ? AND deletado_em IS NULL
	`

	resultado, err := r.db.Exec(query, agora, deletadoPor, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar usuário '%s': %w", id, err)
	}

	// Verifica se alguma linha foi afetada — se não, o usuário não existe ou já foi deletado
	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("usuário não encontrado ou já deletado")
	}

	return nil
}

// AtualizarFoto persiste a chave do objeto S3 na coluna foto_key do usuário informado.
func (r *repositorioMySQL) AtualizarFoto(id string, fotoKey string) error {
	query := `UPDATE tb_usuarios SET foto_key = ?, alterado_em = ? WHERE id = ? AND deletado_em IS NULL`

	resultado, err := r.db.Exec(query, fotoKey, time.Now(), id)
	if err != nil {
		return fmt.Errorf("erro ao atualizar foto do usuário '%s': %w", id, err)
	}

	linhasAfetadas, _ := resultado.RowsAffected()
	if linhasAfetadas == 0 {
		return fmt.Errorf("usuário não encontrado")
	}

	return nil
}
