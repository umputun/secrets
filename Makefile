B=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(B))
GITREV=$(shell git describe --abbrev=7 --always --tags)
REV=$(GITREV)-$(BRANCH)-$(shell date +%Y%m%d-%H:%M:%S)

.PHONY: build test lint race coverage docker docker-push run-dev clean info help e2e-setup e2e e2e-ui

build: info
	cd app && go build -ldflags "-X main.revision=$(REV) -s -w" -o ../secrets

test:
	go test -v -timeout=60s ./...

lint:
	golangci-lint run

race:
	go test -race -timeout=60s ./...

coverage:
	go test -v -timeout=60s -covermode=count -coverprofile=coverage.out.tmp ./...
	grep -v "mock_" coverage.out.tmp > coverage.out
	rm -f coverage.out.tmp
	go tool cover -html=coverage.out -o coverage.html
	@echo "coverage report: coverage.html"

docker:
	docker build -t secrets:$(BRANCH) .

docker-push:
	docker buildx build -t umputun/secrets:$(BRANCH) --platform linux/amd64,linux/arm64 --push .

run-dev:
	docker compose -f docker-compose-dev.yml up --build

clean:
	rm -f secrets coverage.out coverage.out.tmp coverage.html

info:
	@echo "revision: $(REV)"

e2e-setup:
	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

e2e:
	go test -v -count=1 -timeout=5m -tags=e2e ./e2e/...

e2e-ui:
	E2E_HEADLESS=false go test -v -count=1 -timeout=10m -tags=e2e ./e2e/...

help:
	@echo "targets:"
	@echo "  build       - build binary to ./secrets"
	@echo "  test        - run tests"
	@echo "  lint        - run golangci-lint"
	@echo "  race        - run tests with race detector"
	@echo "  coverage    - generate coverage report"
	@echo "  docker      - build docker image for current platform"
	@echo "  docker-push - build and push multi-arch image (amd64, arm64)"
	@echo "  run-dev     - run local dev with docker compose"
	@echo "  clean       - remove build artifacts"
	@echo "  info        - show version info"
	@echo "  e2e-setup   - install playwright browsers"
	@echo "  e2e         - run e2e tests (headless)"
	@echo "  e2e-ui      - run e2e tests with visible browser"

