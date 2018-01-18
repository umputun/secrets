FROM golang:1.9-alpine as build-backend

RUN go version

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN \
    apk add --no-cache --update tzdata git &&\
    cp /usr/share/zoneinfo/America/Chicago /etc/localtime &&\
    go get -u gopkg.in/alecthomas/gometalinter.v1 && \
    ln -s /go/bin/gometalinter.v1 /go/bin/gometalinter && \
    gometalinter --install --force

ADD . /go/src/github.com/umputun/secrets
WORKDIR /go/src/github.com/umputun/secrets

RUN cd app && go test -v $(go list -e ./... | grep -v vendor)

RUN gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode  --enable=gosimple --exclude=test --exclude=mock --exclude=vendor ./...

RUN go build -o secrets -ldflags "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S) -s -w" ./app


FROM node:9.4-alpine as build-frontend

ADD webapp /srv/webapp
RUN apk add --no-cache --update git python make g++
RUN \
    cd /srv/webapp && \
    npm i --production && npm run build


FROM alpine:3.7

COPY --from=build-backend /go/src/github.com/umputun/secrets/secrets /srv/
COPY --from=build-frontend /srv/webapp/public/ /srv/docroot

RUN \
    apk add --update --no-cache tzdata && \
    adduser -s /bin/bash -D -u 1001 secrets && \
    chown -R secrets:secrets /srv

COPY --from=build-frontend /srv/webapp/public/show/index.html /srv/docroot/show/s.html

EXPOSE 8080
USER secrets
WORKDIR /srv
VOLUME ["/srv/docroot"]

ENTRYPOINT ["/srv/secrets"]