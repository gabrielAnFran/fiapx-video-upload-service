# tests/integration

Testes de integracao de verdade (Postgres + RabbitMQ + MinIO reais via
`testcontainers-go`, no mesmo espirito do que `pos-os-service/tests/integration`
faz para o servico de Ordem de Servico) sao um **objetivo desejavel (stretch
goal)** para este servico, ainda nao implementados.

O que existe hoje neste diretorio e apenas um stub (`stub_test.go`) atras da
build tag `integration`, para que:

- `go test ./...` (sem tags) continue rapido e sem dependencias externas,
  cobrindo `internal/domain`, `internal/application/usecases` e
  `internal/presentation` via fakes em memoria.
- `go test -tags=integration ./tests/integration/...` tenha algo para rodar
  e nao quebre o pipeline por falta de arquivos, mesmo antes dos testes reais
  existirem.

## Proximos passos (nao implementados aqui)

- Subir Postgres via testcontainers, rodar as migrations de
  `migrations/000001_init.up.sql` e testar `UserRepository`/
  `VideoRepository`/`OutboxRepository`/`ProcessedEventRepository` contra o
  banco real (unique constraint de email, upsert por `id`, paginacao por
  cursor, fetch/mark de outbox, idempotencia de `processed_events`).
- Subir RabbitMQ via testcontainers e testar `messaging.Conn` (publish,
  consume, retry ate `MaxRetries`, DLQ) do mesmo jeito que
  `pos-os-service/tests/integration/messaging_test.go` faz.
- Subir MinIO via testcontainers e testar `storage.S3Client` (upload
  multipart-safe + presigned GET) contra um bucket real.
