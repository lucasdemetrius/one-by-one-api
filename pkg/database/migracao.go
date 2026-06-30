// Pacote: pkg/database
// Arquivo: migracao.go
// Descrição: "Mini-Flyway" — aplica as migrations SQL no BOOT da aplicação, igual ao Flyway
//            do Spring Boot. Mantém uma tabela de controle (tb_migracoes) com o que já rodou,
//            então cada arquivo roda UMA vez, em ordem, seja em banco novo ou já existente.
//            Resultado: ao subir a aplicação, as tabelas são criadas/atualizadas sozinhas —
//            não precisa mais aplicar migration na mão nem montar a pasta no container.
// Autor: OneByOne API
// Criado em: 2026

package database

import (
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql" // driver MySQL (side-effect)
	"github.com/jmoiron/sqlx"
	"onebyone-api/pkg/config"
)

// AplicarMigracoes roda as migrations pendentes no boot. `arquivos` é o embed.FS com os .sql.
//
// Abre uma conexão DEDICADA com multiStatements=true (cada arquivo .sql tem vários comandos
// separados por ';'); a conexão normal da aplicação NÃO usa esse modo (mais seguro). É
// idempotente: o que já está registrado em tb_migracoes não roda de novo.
func AplicarMigracoes(cfg *config.Config, arquivos fs.FS) error {
	// multiStatements permite executar um arquivo inteiro (vários comandos) numa só chamada,
	// deixando o próprio MySQL tratar comentários e a separação por ';'.
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("erro ao abrir conexão de migração: %w", err)
	}
	defer db.Close()

	// 1) Tabela de controle (o equivalente ao flyway_schema_history).
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tb_migracoes (
			versao      VARCHAR(255) NOT NULL PRIMARY KEY,
			aplicada_em DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("erro ao criar tb_migracoes: %w", err)
	}

	// 2) O que já foi aplicado.
	var aplicadas []string
	if err := db.Select(&aplicadas, `SELECT versao FROM tb_migracoes`); err != nil {
		return fmt.Errorf("erro ao ler tb_migracoes: %w", err)
	}
	jaAplicada := make(map[string]bool, len(aplicadas))
	for _, v := range aplicadas {
		jaAplicada[v] = true
	}

	// 3) Lista os arquivos .sql em ordem (001_, 002_, ... — o zero-padding garante a ordem).
	entradas, err := fs.ReadDir(arquivos, ".")
	if err != nil {
		return fmt.Errorf("erro ao listar migrations: %w", err)
	}
	var nomes []string
	for _, e := range entradas {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			nomes = append(nomes, e.Name())
		}
	}
	sort.Strings(nomes)

	// 4) BASELINE: banco PRÉ-EXISTENTE (criado pelo modo antigo, via initdb do MySQL) já tem
	//    o schema, mas a tb_migracoes está vazia. Marca tudo como aplicado SEM reexecutar
	//    (evita erro "table already exists"). Só acontece uma vez. Em banco NOVO, a tabela
	//    tb_usuarios ainda não existe → cai no fluxo normal (passo 5) e cria tudo.
	if len(aplicadas) == 0 && tabelaExiste(db, "tb_usuarios") {
		for _, nome := range nomes {
			if _, err := db.Exec(`INSERT INTO tb_migracoes (versao) VALUES (?)`, nome); err != nil {
				return fmt.Errorf("erro ao baselinar %s: %w", nome, err)
			}
		}
		log.Printf("[migracao] baseline: banco pré-existente — %d migration(s) marcada(s) como aplicada(s) sem reexecutar", len(nomes))
		return nil
	}

	// 5) Aplica as pendentes, em ordem.
	novas := 0
	for _, nome := range nomes {
		if jaAplicada[nome] {
			continue
		}
		conteudo, err := fs.ReadFile(arquivos, nome)
		if err != nil {
			return fmt.Errorf("erro ao ler migration %s: %w", nome, err)
		}
		if _, err := db.Exec(string(conteudo)); err != nil {
			return fmt.Errorf("migration %s falhou: %w", nome, err)
		}
		if _, err := db.Exec(`INSERT INTO tb_migracoes (versao) VALUES (?)`, nome); err != nil {
			return fmt.Errorf("erro ao registrar migration %s: %w", nome, err)
		}
		log.Printf("[migracao] aplicada: %s", nome)
		novas++
	}
	if novas == 0 {
		log.Printf("[migracao] banco em dia (%d migration(s))", len(nomes))
	} else {
		log.Printf("[migracao] %d migration(s) nova(s) aplicada(s); %d no total", novas, len(nomes))
	}
	return nil
}

// tabelaExiste diz se uma tabela existe no schema (database) atual.
func tabelaExiste(db *sqlx.DB, tabela string) bool {
	var n int
	err := db.Get(&n, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, tabela)
	return err == nil && n > 0
}
