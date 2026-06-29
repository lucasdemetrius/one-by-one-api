// Pacote: internal/blocotema
// Arquivo: entity.go
// Descrição: Entidade BlocoTema — um bloco de conteúdo de um tema de 1:1 de um
//            liderado (texto, link/curso, imagem no S3 ou marco com datas).
//            Mapeia a tabela tb_blocos_tema.
// Autor: OneByOne API
// Criado em: 2025

package blocotema

import "time"

// Tipos de bloco suportados.
const (
	TipoTexto  = "TEXTO"  // parágrafo de texto livre
	TipoLink   = "LINK"   // link/curso: url + rótulo (texto)
	TipoImagem = "IMAGEM" // imagem no S3 + legenda (texto)
	TipoMarco  = "MARCO"  // etapa com data de início/fim + descrição (texto)
)

// BlocoTema é um item de conteúdo dentro de um tema, para um colaborador.
type BlocoTema struct {
	ID            string     `db:"id"`
	ColaboradorID string     `db:"colaborador_id"`
	Tema          string     `db:"tema"`
	Tipo          string     `db:"tipo"`
	Texto         *string    `db:"texto"`
	URL           *string    `db:"url"`
	ImagemKey     *string    `db:"imagem_key"`
	DataInicio    *time.Time `db:"data_inicio"`
	DataFim       *time.Time `db:"data_fim"`
	Ordem         int        `db:"ordem"`
	CriadoEm      time.Time  `db:"criado_em"`
}
