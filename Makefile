.PHONY: run test lint build

# Carga .env si existe; las variables ya exportadas en el shell tienen precedencia.
ifneq (,$(wildcard .env))
  include .env
  export
endif

run:
	go run ./cmd/api/

test:
	go test ./... -covermode=atomic

build:
	go build -o api ./cmd/api/

lint:
	go vet ./...
