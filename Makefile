#!make
include ./.env

GO=go

.PHONY: run
run:
	$(GO) run main.go

.PHONY: build
build:
	$(GO) build -mod=mod -o bin/example cmd/example/main.go

.PHONY: build-docker-image
build-docker-image:
	docker build -t finsupport/hot-wallet-service-eth . --no-cache

.PHONY: migrateup
migrateup:
	migrate -path internal/db/migrations -database "postgresql://${POSTGRES_PASSWORD}:${POSTGRES_USER}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSL}" -verbose up

.PHONY: migratedown
migratedown:
	migrate -path internal/db/migrations -database "postgresql://${POSTGRES_PASSWORD}:${POSTGRES_USER}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSL}" -verbose down

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-coverage
test-coverage:
	mkdir -p coverage/
	go test -v ./... -covermode=set -coverpkg=./... -coverprofile coverage/coverage.out
	go tool cover -html coverage/coverage.out -o coverage/coverage.html
	open coverage/coverage.html

.PHONY: check-style
check-style:
	$(GO) fmt ./...

# Add database connections below

.PHONY: verify-tor
verify-tor:
	@echo "🔍 Verifying TOR connection..."
	@go run test_utils/verify_tor_connection.go

.PHONY: test-tor
test-tor:
	@echo "🧪 Testing with TOR proxy..."
	@HTTP_PROXY=socks5://127.0.0.1:9050 HTTPS_PROXY=socks5://127.0.0.1:9050 go run test_utils/etherum/ether.go

.PHONY: start-tor
start-tor:
	@echo "🚀 Starting TOR proxy..."
	@docker-compose up -d tor1

.PHONY: stop-tor
stop-tor:
	@echo "🛑 Stopping TOR proxy..."
	@docker-compose stop tor1

.PHONY: tor-logs
tor-logs:
	@echo "📋 TOR proxy logs..."
	@docker logs tor-proxy

.PHONY: run-example
run-example:
	./bin/example
