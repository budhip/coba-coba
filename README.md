# About The Project
project by finance platform for transaction and accounting purpose
add new create transaction acuan bulking

---

## Tech Stack
- Written in go

### Tools

- Go 1.18 or latest
- Docker

### Web Framework

- https://github.com/gofiber/fiber

### Driver Packages

- Loger driver https://github.com/uber-go/zap wrap using https://bitbucket.org/Amartha/go-x


### Testing Packages

- A toolkit with common assertions https://github.com/stretchr/testify
- Mock https://github.com/golang/mock

### Additional Packages

- Libraries for configuration parsing https://github.com/spf13/viper
- Validator https://github.com/go-playground/validator
- Linter https://github.com/golangci/golangci-lint
- Swagger doc https://github.com/swaggo/swag
- List of go frameworks & libraries https://github.com/avelino/awesome-go

---

## HOW TO RUN

clone the project inside folder `{GO_PROJECT_DIR}/src/bitbucket.org/Amartha`
```bash
git clone git@bitbucket.org:Amartha/go-fp-transaction.git
```

To run this service, you need to add configuration file
```bash
cp config/config.local.example.yaml config/config.local.yaml
```
This service already uses `go.mod`. `make tidy` will simply get all dependencies.

### Run Service

1. Run `make docker-start`
2. Run `make run-api`

### Generate Swagger

1. Install requirement `make swag-prepare`
2. Run generate swagger `make swag-gen`

### Generate Mock

1. Install requirement `make mock-prepare`
2. Run generate mock `make mock-gen`

### Generate Error

Run `make error-gen`

### Run Unit Test

Run `make test`

### Run Linter

1. Install requirement `make lint-prepare`
2. Run linter `make lint`
3. Result in `lint.xml`

---
