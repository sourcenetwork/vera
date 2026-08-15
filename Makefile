IGNITE_RUN = docker run --rm -ti --volume $(PWD):/apps ignitehq/cli:latest
UID := $(shell id -u)
GID := $(shell id -g)
BIN = build/verad
DEMO_BIN = build/token-protocol-demo

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o ${BIN} ./cmd/verad

.PHONY: build-mac
build-mac:
	GOOS=darwin GOARCH=arm64 go build -o ${BIN} ./cmd/verad

.PHONY: install
install:
	go install ./cmd/verad

.PHONY: proto
proto:
	ignite generate proto-go --yes
	PATH="$$(go env GOPATH)/bin:$$PATH" buf generate proto --template proto/buf.gen.pulsar.yaml \
		--exclude-path proto/vera/orbis/document.proto \
		--exclude-path proto/vera/orbis/ring.proto \
		--exclude-path proto/vera/orbis/tx.proto
	PATH="$$(go env GOPATH)/bin:$$PATH" buf generate proto --template proto/buf.gen.orbis.optional.yaml \
		--path proto/vera/orbis/document.proto \
		--path proto/vera/orbis/ring.proto \
		--path proto/vera/orbis/tx.proto
	rm -f api/vera/orbis/document.pulsar.go \
		api/vera/orbis/ring.pulsar.go \
		api/vera/orbis/tx.pulsar.go

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

.PHONY: test_env_generator
test_env_generator:
	go build -o build/test_env_generator cmd/test_env_generator/main.go

.PHONY: docker
docker:
	docker image build -t ghcr.io/sourcenetwork/vera:dev .
