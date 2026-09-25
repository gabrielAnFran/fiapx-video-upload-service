//go:build integration

package integration

import "testing"

// TestPlaceholder is a trivial passing test behind the `integration` build
// tag. Full testcontainers-go integration tests (real Postgres, RabbitMQ,
// and MinIO containers exercising VideoRepository/UserRepository/
// OutboxRepository/ProcessedEventRepository and the messaging.Conn helper)
// are a stretch goal deferred as polish — see README.md in this directory.
func TestPlaceholder(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("arithmetic is broken")
	}
}
