
export GOPRIVATE := bitbucket.org/Amartha
export GOOGLE_APPLICATION_CREDENTIALS := credentials.json

docker-start:
	docker-compose up -d

docker-stop:
	docker-compose down

run-api: tidy swag-gen 
	CGO_ENABLED=0 go run ./cmd/api/main.go
.PHONY: run-api

run-consumer-transaction: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=transaction
.PHONY: run-consumer-transaction

run-consumer-dlq_notification: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=dlq_notification
.PHONY: run-consumer-dlq_notification

run-consumer-dlq_retrier: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=dlq_retrier
.PHONY: run-consumer-dlq_retrier

run-consumer-account_mutation: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=account_mutation
.PHONY: run-consumer-account_mutation

run-consumer-recon_task_queue: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=recon_task_queue
.PHONY: run-consumer-recon_task_queue

run-consumer-hvt_balance_update: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=hvt_balance_update
.PHONY: run-consumer-hvt_balance_update

run-consumer-money_flow_calc: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=money_flow_calc
.PHONY: run-consumer-money_flow_calc

run-consumer-transaction_stream: tidy swag-gen
	CGO_ENABLED=0 go run ./cmd/consumer/main.go run -n=transaction_stream
.PHONY: run-consumer-transaction_stream

error-gen: 
	CGO_ENABLED=0 go run ./cmd/errorgen/main.go

tidy:
	go mod tidy
	go mod download
.PHONY: tidy

prepare-release: swag-gen error-gen test

swag-prepare:
	go install github.com/swaggo/swag/cmd/swag@latest

swag-gen:
	swag init --parseDependency --parseInternal -g internal/deliveries/http/router.go --output docs/
.PHONY: swag

mock-prepare:
	go install go.uber.org/mock/mockgen@latest

mock-gen:
	@./scripts/generate-mock.sh services
	@./scripts/generate-mock.sh repositories
	@./scripts/generate-mock.sh common/idgenerator
	@./scripts/generate-mock.sh common/metrics
	@./scripts/generate-mock.sh common/retry
	@./scripts/generate-mock.sh common/acuanclient
	@./scripts/generate-mock.sh common/dlq_publisher
	@./scripts/generate-mock.sh common/queueunicorn
	@./scripts/generate-mock.sh common/publisher
	@./scripts/generate-mock.sh common/transaction_notification
	@./scripts/generate-mock.sh common/accounting
	@./scripts/generate-mock.sh common/flag
	@./scripts/generate-mock.sh internal/deliveries/consumer/kafka_recon
	@./scripts/generate-mock.sh internal/deliveries/consumer/kafka
	mockgen -source=./internal/common/safeaccess/gcs.go -destination=./internal/common/safeaccess/mock/gcs_mock.go -package=mock
	mockgen -destination=internal/deliveries/consumer/kafka/mock/consumer_group_mock.go -package=mock github.com/Shopify/sarama ConsumerGroupSession,ConsumerGroup,ConsumerGroupClaim


test_files:= $(shell go list ./internal/... | grep -v /mock)

test:
	CGO_ENABLED=1 GOPRIVATE=bitbucket.org/Amartha go test --count=1 -short -race -cover $(test_files)

test-cover:
	CGO_ENABLED=1 GOPRIVATE=bitbucket.org/Amartha go test --count=1 -short -race -coverprofile=./cov.out $(test_files)

test-cover-display: mock-prepare test-cover
	go tool cover -html=cov.out

lint-prepare:
	$(eval GOPATH := $(shell go env GOPATH))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.53.1

lint:
	golangci-lint run --out-format checkstyle > lint.xml