# AI Integration Copilot

Ferramenta para gerar um projeto Go de integração a partir de uma especificação OpenAPI usando Gemini via API.

## Estado atual

Hoje o projeto implementa o backend do pipeline:

- recebe uma spec OpenAPI por upload multipart ou por caminho local
- faz parse da spec com `kin-openapi`
- monta um prompt enxuto para geração de código
- chama o Gemini via HTTP
- persiste os arquivos gerados em `/tmp`
- empacota o resultado em `.zip`
- expõe status e download por job

O frontend ainda não está implementado.

## Fluxo

1. `POST /generate`
2. backend salva ou lê a spec
3. parser extrai metadados, endpoints e autenticação
4. prompt builder pede um projeto Go mínimo e pronto para usar
5. runner envia o prompt para o Gemini
6. o modelo retorna JSON com arquivos
7. backend grava os arquivos e gera um archive `.zip`
8. cliente consulta `GET /generate/:id` e baixa em `GET /generate/:id/download`

## Estrutura

```text
backend/
  cmd/server/              entrypoint HTTP
  internal/api/            rotas e handlers
  internal/ai/             montagem do prompt
  internal/config/         config por ambiente
  internal/generator/      jobs, runner Gemini e persistência de artefatos
  internal/parser/         parse de OpenAPI
```

## Requisitos

- Go `1.25.1`
- chave de API do Gemini
- um modelo Gemini compatível, por padrão `gemini-3.1-flash-lite-preview`

## Variáveis de ambiente

- `PORT`: porta do backend. Padrão: `8080`
- `GEMINI_API_KEY`: chave da API Gemini. Obrigatória
- `GEMINI_MODEL`: modelo usado na geração. Padrão: `gemini-3.1-flash-lite-preview`
- `GEMINI_BASE_URL`: URL base da API Gemini. Padrão: `https://generativelanguage.googleapis.com`
- `GEMINI_TIMEOUT`: timeout da chamada ao Gemini. Padrão: `2m`

Exemplo de `.env`:

```dotenv
PORT=8080
GEMINI_API_KEY=sua_chave_aqui
GEMINI_MODEL=gemini-3.1-flash-lite-preview
GEMINI_BASE_URL=https://generativelanguage.googleapis.com
GEMINI_TIMEOUT=2m
```

O backend autentica no Gemini usando o header `x-goog-api-key`.

## Rodando localmente

```bash
export GEMINI_API_KEY=sua_chave_aqui
cd backend
go run ./cmd/server
```

Servidor padrão: `http://localhost:8080`

## Rodando com Docker Compose

```bash
docker compose up --build
```

Se você usar um arquivo `.env` na raiz do projeto, o Compose lê `GEMINI_API_KEY` automaticamente.

O `docker-compose.yml` já sobe o backend com:

- porta `8080`
- modelo `gemini-3.1-flash-lite-preview`
- timeout `5m`

## Endpoints

### `GET /health`

Resposta de saúde do serviço.

### `POST /generate`

Aceita:

- `multipart/form-data` com `specFile`
- `application/json` com `specPath`

Exemplo com arquivo:

```bash
curl -X POST http://localhost:8080/generate \
  -F "specFile=@/caminho/para/openapi.yaml"
```

Exemplo com JSON:

```bash
curl -X POST http://localhost:8080/generate \
  -H "Content-Type: application/json" \
  -d '{"specSource":"file","specPath":"/caminho/para/openapi.yaml"}'
```

Resposta esperada:

- `jobId`
- `status`
- metadados da spec
- preview do prompt

### `GET /generate/:id`

Retorna o status do job, erro se houver, métricas, arquivos gerados e caminhos de saída.

Exemplo:

```bash
curl http://localhost:8080/generate/job-123
```

### `GET /generate/:id/download`

Baixa o `.zip` do job quando o status for `succeeded`.

Exemplo:

```bash
curl -O -J http://localhost:8080/generate/job-123/download
```

## Formato esperado do modelo

O backend espera que o modelo retorne somente JSON neste formato:

```json
{
  "files": [
    {
      "path": "go.mod",
      "content": "module generated/integration"
    }
  ]
}
```

O prompt atual pede exatamente estes arquivos:

- `go.mod`
- `client/client.go`
- `client/models.go`
- `client/auth.go`
- `cmd/example/main.go`
- `README.md`

## Saída gerada

Os arquivos gerados são gravados em um diretório temporário por job:

- specs enviadas: `/tmp/ai-integration-specs`
- artefatos gerados: `/tmp/ai-integration-output/<job-id>`
- archives: `/tmp/ai-integration-output/archives/<job-id>.zip`

## Limites atuais

- `specUrl` ainda não está implementado
- jobs ficam só em memória
- não há persistência de histórico após restart
- frontend ainda não existe
- diagramas Mermaid ainda não são gerados
