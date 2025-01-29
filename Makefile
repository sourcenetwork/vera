IGNITE_RUN = docker run --rm -ti --volume $(PWD):/apps ignitehq/cli:latest
UID := $(shell id --user)
GID := $(shell id --group)
BIN = build/sourcehubd
DEMO_BIN = build/token-protocol-demo

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o ${BIN} ./cmd/sourcehubd

.PHONY: install
install:
	go install ./cmd/sourcehubd

.PHONY: proto
proto:
	ignite generate proto-go

.PHONY: test
test:
	go test ./...

.PHONY: test\:all
test\:all: test_env_generator
	scripts/run-test-matrix

.PHONY: simulate
simulate:
	ignite chain simulate

.PHONY: fmt
fmt:
	gofmt -w .
	buf format --write

.PHONY: run
run: build
	${BIN} start

.PHONY: docs
docs:
	pkgsite -http 0.0.0.0:8080

.PHONY: image
# builds a production docker image in the local system and tags it with
# the ID of the current git HEAD
image:
	scripts/build-docker-image.sh

.PHONY: test_env_generator
test_env_generator:
	go build -o build/test_env_generator cmd/test_env_generator/main.go
