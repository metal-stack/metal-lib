.DEFAULT_GOAL := gotest

.PHONY: gotest
gotest: up
	@sleep 2 || true
	docker logs nsqtest
	@docker-compose down || true

.PHONY: up
up: build
	@docker-compose down || true
	docker-compose up -d

.PHONY: down
down:
	docker-compose down

.PHONY: build
build: gofmt
	go build -tags netgo -o bin/nsqtest
	docker-compose build

.PHONY: gofmt
gofmt:
	GO111MODULE=off go fmt ./...
