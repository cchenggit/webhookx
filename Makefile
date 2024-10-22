LDFLAGS = --ldflags "\
		-X github.com/webhookx-io/webhookx/config.COMMIT=`git rev-parse --verify --short HEAD` \
		-X github.com/webhookx-io/webhookx/config.VERSION=`git tag -l --points-at HEAD | head -n 1`"

DIR = $(shell pwd)

.PHONY: clean
clean:
	go clean
	go clean -testcache

.PHONY: build
build:
	CGO_ENABLED=0 go build ${LDFLAGS}

.PHONY: install
install:
	go install ${LDFLAGS}

.PHONY: generate
generate:
	go generate ./...

.PHONY: test
test: clean
	go test $$(go list ./... | grep -v /test/)

test-deps:
	export WEBHOOKX_TEST_OTEL_COLLECTOR_OUTPUT_PATH=$(DIR)/test/output/otel
	mkdir -p test/output/otel
	docker compose -f test/docker-compose.yml up -d

.PHONY: test-coverage
test-coverage: clean
	go test $$(go list ./... | grep -v /test/) -coverprofile=coverage.txt

.PHONY: test-integration
test-integration: clean
	go test -p 1 -v ./test/...

.PHONY: test-integration-coverage
test-integration-coverage: clean
	go test -p 1 -v ./test/... --coverpkg ./... -coverprofile=coverage.txt

.PHONY: goreleaser
goreleaser:
	goreleaser release --snapshot --clean

migrate-create:
	migrate create -ext sql -dir db/migrations -seq -digits 1 $(message)
