# fiapx-video-upload-service

Microsservico de upload de video, autenticacao e consulta de status do
sistema FIAP X de processamento de video (upload de video -> extracao
assincrona de frames -> zip para download), extraido como um dos quatro
microsservicos do desafio.

Este servico e responsavel por:

- **Autenticacao**: registro (`bcrypt` para hash de senha) e login,
  emitindo um JWT (HS256, 24h de expiracao) que protege os demais endpoints.
- **Upload**: recebe o arquivo de video via `multipart/form-data`, grava o
  objeto no MinIO em streaming (sem bufferizar o arquivo inteiro em
  memoria) e publica o evento `video.uploaded` via outbox.
- **Consulta de status**: lista os videos do usuario autenticado (por
  status, com paginacao por cursor).
- **Download**: emite uma URL presignada (15 min de validade) para o zip de
  frames de um video ja `COMPLETED`, sem proxiar o download por este
  servico.

## Arquitetura

Este servico e um dos participantes de uma saga coordenada por um servico
orquestrador separado, **`fiapx-saga-orchestrator`** (veja aquele
repositorio para os diagramas de sequencia completos entre os quatro
servicos). Este servico nao chama outros servicos diretamente; a
comunicacao acontece exclusivamente via eventos/comandos no RabbitMQ
(exchange topic `video.events`), usando o padrao transactional outbox para
garantir que a escrita no banco e o evento correspondente nunca fiquem
inconsistentes entre si. Veja `docs/adr/0001-orchestrated-saga.md` para o
racional da escolha de uma saga orquestrada mesmo com uma pipeline linear.

Organizacao dos processos (3 binarios, 1 banco de dados, 1 broker AMQP e 1
MinIO compartilhados):

- `cmd/server` — API HTTP em Gin (`/api/v1/auth/...`, `/api/v1/videos...`),
  grava no Postgres e na tabela `outbox` na mesma transacao das escritas de
  dominio, e envia o video para o MinIO antes de confirmar o upload.
- `cmd/outbox-dispatcher` — faz polling da tabela `outbox` e publica as
  linhas ainda nao publicadas no RabbitMQ, com backoff exponencial em caso
  de falha de publicacao.
- `cmd/worker` — consome comandos terminais do RabbitMQ
  (`upload-service.events.q`): `video.status.completed` e
  `video.status.failed`, emitidos pelo orquestrador apos o pipeline de
  processamento terminar.

## Participacao na saga

- **Emite**: `video.uploaded` (apos gravar o video no MinIO e a linha em
  `videos`). Publicado via padrao outbox: o evento e gravado na tabela
  `outbox` na mesma transacao de banco do `INSERT` em `videos`, e o
  `outbox-dispatcher` publica no RabbitMQ de forma assincrona e marca como
  publicado.
- **Consome**: `video.status.completed` / `video.status.failed` (comandos
  terminais vindos do orquestrador, apos o pipeline de extracao de frames
  terminar) — atualiza o status do video e os campos relevantes
  (`zip_bucket`/`zip_object_key`/`frame_count`/`completed_at` ou
  `error_message`/`failed_at`). Este servico e um **consumidor terminal**
  desses dois comandos: nao emite nenhum evento adicional em resposta.
- **Idempotencia**: comandos recebidos sao verificados contra a tabela
  `processed_events` (indexada por `event_id`) antes do processamento, e
  marcados como processados apos uma atualizacao bem sucedida. Combinado com
  a chave primaria da tabela, isso torna o reenvio do mesmo comando uma
  operacao sem efeito (no-op).

## Rodando localmente (standalone)

Requer Postgres, RabbitMQ e MinIO acessiveis pelas variaveis de ambiente
abaixo. Exemplo usando containers Docker locais:

```bash
docker run -d --name upload-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=upload_service -p 5432:5432 postgres:16-alpine
docker run -d --name upload-rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management-alpine
docker run -d --name upload-minio -p 9000:9000 -p 9001:9001 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin minio/minio server /data --console-address ":9001"

export UPLOAD_DB_DSN="host=localhost user=postgres password=postgres dbname=upload_service port=5432 sslmode=disable"
export UPLOAD_AMQP_URL="amqp://guest:guest@localhost:5672/"
export UPLOAD_JWT_SECRET="dev-secret-change-me"
export MINIO_ENDPOINT="localhost:9000"
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
export MINIO_BUCKET="fiapx-videos"

go run ./cmd/server             # API HTTP na porta :8081, roda migrations no boot
go run ./cmd/outbox-dispatcher  # processo separado, faz polling da outbox
go run ./cmd/worker             # processo separado, consome video.status.completed/failed
```

Lembre-se de criar o bucket `fiapx-videos` no MinIO antes do primeiro upload
(via `mc mb` ou pelo console em `http://localhost:9001`).

Ou usando as imagens de container geradas a partir do `Dockerfile` deste
repositorio:

```bash
docker build --build-arg TARGET=server -t fiapx-video-upload-service:server .
docker run --rm -p 8081:8081 \
  -e UPLOAD_DB_DSN="host.docker.internal:..." \
  -e UPLOAD_AMQP_URL="amqp://guest:guest@host.docker.internal:5672/" \
  fiapx-video-upload-service:server
```

## Rodando como parte da stack completa

Este servico e composto como parte da stack completa via
`fiapx-saga-orchestrator/deploy/local/docker-compose.yml` (nesse repositorio
irmao — nao alterado aqui), que orquestra Postgres, RabbitMQ, MinIO, este
servico e os outros tres microsservicos juntos.

## Variaveis de ambiente

| Variavel                       | Padrao                                                                                            | Descricao                                                  |
|---------------------------------|-----------------------------------------------------------------------------------------------------|--------------------------------------------------------------|
| `UPLOAD_PORT`                   | `8081`                                                                                               | Porta HTTP do `cmd/server`                                    |
| `UPLOAD_DB_DSN`                 | `host=localhost user=postgres password=postgres dbname=upload_service port=5432 sslmode=disable`    | DSN do Postgres (formato GORM/`lib/pq`)                       |
| `UPLOAD_AMQP_URL`                | `amqp://guest:guest@localhost:5672/`                                                                 | URL de conexao com o RabbitMQ                                 |
| `UPLOAD_DISPATCH_INTERVAL_MS`    | `500`                                                                                                 | Intervalo de polling da outbox no `cmd/outbox-dispatcher`     |
| `UPLOAD_JWT_SECRET`              | `dev-secret-change-me`                                                                               | Segredo de assinatura HS256 dos JWTs                           |
| `MINIO_ENDPOINT`                 | `localhost:9000`                                                                                      | Endpoint host:port do MinIO                                    |
| `MINIO_ACCESS_KEY`               | `minioadmin`                                                                                          | Access key do MinIO                                            |
| `MINIO_SECRET_KEY`               | `minioadmin`                                                                                          | Secret key do MinIO                                            |
| `MINIO_BUCKET`                   | `fiapx-videos`                                                                                        | Bucket usado para videos originais e zips de frames             |
| `MINIO_USE_SSL`                  | `false`                                                                                               | Se `true`, conecta ao MinIO via HTTPS                           |

## API

Veja `docs/openapi.yaml`. Resumo:

- `POST /api/v1/auth/register` -> `201`, `409` se o email ja existir
- `POST /api/v1/auth/login` -> `200 {token}`, `401` em credenciais invalidas
- `POST /api/v1/videos` (autenticado, `multipart/form-data`, campo `video`)
  -> `202 {video_id, status}`
- `GET /api/v1/videos?status=&cursor=&limit=` (autenticado) -> lista os
  videos do usuario autenticado (nunca de outro usuario)
- `GET /api/v1/videos/:id/download` (autenticado) -> `200 {download_url}`,
  `403` se o video for de outro usuario, `400` se ainda nao estiver
  `COMPLETED`, `404` se nao existir
- `GET /healthz`, `GET /readyz`

Todos os erros seguem o formato `application/problem+json` (RFC 7807).

## Testes

```bash
go test ./...                                     # testes unitarios, sem dependencias externas, rapido
go test -tags=integration ./tests/integration/...  # stub atras da build tag `integration` (testes reais com testcontainers-go sao um stretch goal, ver tests/integration/README.md)
```

`internal/application/usecases` e `internal/infrastructure/auth` sao
cobertos por testes unitarios com fakes em memoria (sem framework de
mocking), incluindo `RegisterUser`, `LoginUser`, `UploadVideo`,
`GetDownloadURL` e os dois handlers de comando idempotentes
(`HandleStatusCompleted`/`HandleStatusFailed`, com casos de sucesso, replay
idempotente e video nao encontrado).

## Notas de desenvolvimento / desvios da especificacao

- O `go.mod` fixa `go 1.26.0` (nao `1.23.2` como nos servicos irmaos)
  porque este servico e o unico dos quatro que depende do
  `aws-sdk-go-v2` (cliente MinIO/S3) e de versoes recentes de
  `golang-migrate`/`jackc/pgx`/`gorm.io/driver/postgres`; mesmo fixando esses
  pacotes nas versoes mais antigas compativeis com Go 1.23 disponiveis
  (`golang-migrate v4.17.1`, `jackc/pgx/v5 v5.6.0`,
  `gorm.io/driver/postgres v1.6.0`, `aws-sdk-go-v2 v1.36.0`), o
  `go mod tidy` ainda recalculava a diretiva `go` para cima a partir de
  dependencias transitivas (ex: `golang.org/x/crypto`) que por sua vez exigem
  Go mais novo em suas ultimas versoes patch. Como o toolchain instalado
  neste ambiente e o 1.26.5 (mais novo que qualquer requisito encontrado),
  isso nao afeta build nem testes — `go build ./...`, `go vet ./...` e
  `go test ./...` estao todos verdes — apenas diverge do numero exato "1.23"
  citado como convencao a copiar dos servicos irmaos.
