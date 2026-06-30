// Pacote: migracoes
// Arquivo: embed.go
// Descrição: Embute todos os arquivos .sql desta pasta no binário (//go:embed). Assim a
//            aplicação aplica as migrations no boot (estilo Flyway), sem depender de montar
//            a pasta no container — a imagem final só carrega o binário.
// Autor: OneByOne API
// Criado em: 2026

package migracoes

import "embed"

// Arquivos contém todos os .sql desta pasta, embutidos em tempo de compilação.
//
//go:embed *.sql
var Arquivos embed.FS
