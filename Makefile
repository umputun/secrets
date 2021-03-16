B=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(B))
GITREV=$(shell git describe --abbrev=7 --always --tags)
REV=$(GITREV)-$(BRANCH)-$(shell date +%Y%m%d-%H:%M:%S)

all: build docker frontend

build: info
	- cd backend/app; GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.revision=$(REV)" -o ../target/secrets

docker:
	- docker build -t secrets:$(BRANCH) .

frontend:
	- docker run --rm -it --name=secrets.tmp -d secrets:$(BRANCH) /bin/sh
	- docker cp secrets.tmp:/srv/docroot/ ./var/docroot
	- docker rm -f secrets.tmp

push:
	- docker secrets:${BRANCH}

check:
	- cd backend/app; golangci-lint run --out-format=tab --tests=false ./...

info:
	- @echo "revision $(REV)"

.PHONY: bin info frontend docker
