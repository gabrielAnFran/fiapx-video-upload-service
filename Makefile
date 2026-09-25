.PHONY: build test test-integration lint cover docker helm-lint run

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags integration ./tests/integration/...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, falling back to go vet + gofmt"; \
		go vet ./...; \
		test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1); \
	fi

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

docker:
	docker build -t fiapx-video-upload-service:server --build-arg TARGET=server .
	docker build -t fiapx-video-upload-service:outbox-dispatcher --build-arg TARGET=outbox-dispatcher .
	docker build -t fiapx-video-upload-service:worker --build-arg TARGET=worker .

helm-lint:
	@if command -v helm >/dev/null 2>&1; then \
		helm lint charts/upload-service; \
	else \
		echo "helm not installed, skipping lint"; \
	fi

run:
	UPLOAD_PORT=$${UPLOAD_PORT:-8081} \
	UPLOAD_DB_DSN=$${UPLOAD_DB_DSN:-"host=localhost user=postgres password=postgres dbname=upload_service port=5432 sslmode=disable"} \
	UPLOAD_AMQP_URL=$${UPLOAD_AMQP_URL:-"amqp://guest:guest@localhost:5672/"} \
	UPLOAD_JWT_SECRET=$${UPLOAD_JWT_SECRET:-"dev-secret-change-me"} \
	MINIO_ENDPOINT=$${MINIO_ENDPOINT:-"localhost:9000"} \
	MINIO_ACCESS_KEY=$${MINIO_ACCESS_KEY:-"minioadmin"} \
	MINIO_SECRET_KEY=$${MINIO_SECRET_KEY:-"minioadmin"} \
	MINIO_BUCKET=$${MINIO_BUCKET:-"fiapx-videos"} \
	go run ./cmd/server
