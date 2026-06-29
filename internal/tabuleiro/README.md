# Módulo `tabuleiro`

Persistência e acesso ao estado do tabuleiro (pauta) do 1:1, compartilhado entre líder e liderado (board colaborativo).

> Documentação gerada a partir do código. O **catálogo geral** (com todos os
> módulos) está na seção 12 do [CLAUDE.md](../../CLAUDE.md). Regras de posse/IDOR
> na seção 7.1. **Ao mudar rota/regra/validação, atualize este README.**

## Endpoints

| Rota | O que faz | Acesso |
|---|---|---|
| `GET /colaboradores/:id/tabuleiro` | Obtém o estado salvo do tabuleiro do liderado (JSON com colunas, temas, banco/pauta/conversado). Retorna null se nunca foi salvo. | LÍDER (dono) OU COLABORADOR (o próprio liderado) |
| `PUT /colaboradores/:id/tabuleiro` | Salva/atualiza o estado completo do tabuleiro do liderado (upsert por colaborador_id). Estado deve ser JSON válido. | LÍDER (dono) OU COLABORADOR (o próprio liderado) |

## Tabela(s)

tb_tabuleiros: colaborador_id (VARCHAR 36, PK), estado (JSON), criado_em (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP), alterado_em (DATETIME NULL)

## Regras de negócio

- Posse via PodeAcessar (Cadeia B): líder dono OU o próprio liderado (board é colaborativo — ambos manipulam temas ao vivo)
- Upsert: INSERT ON DUPLICATE KEY UPDATE por colaborador_id (uma única linha por liderado)
- Sem estrutura — backend não interpreta o JSON: persiste como-é, devolve como-é; responsabilidade do frontend é a pauta com colunas/temas
- Recurso alheio → 404 (mapeia ErrAcessoNegado)

## Validações

- Estado: obrigatório (binding:required) — JSON bruto (json.RawMessage)
- Colaborador_id: UUID recebido via parâmetro de URL (validado como válido antes de rotas)
- Posse: garantida no UseCase via PodeAcessar antes de Buscar/Salvar (defesa em profundidade)
