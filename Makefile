.PHONY: tidy setup lint test_short test_short_coverage test_coverage test

tidy:
	@go mod tidy

lint:
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

test_short:
	@go run gotest.tools/gotestsum@latest -- --short ./...

test_short_coverage:
	@go run gotest.tools/gotestsum@latest -- --short ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

docker_up:
	@docker compose -f test/docker-compose.yaml up -d

docker_down:
	@docker compose -f test/docker-compose.yaml down

# Run fist docker_up and then docker_down. We can't automatize this without adding more scripts
test:
	go run gotest.tools/gotestsum@latest

# Run fist docker_up and then docker_down. We can't automatize this without adding more scripts
test_coverage:
	@go run gotest.tools/gotestsum@latest -- ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
