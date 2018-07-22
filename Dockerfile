ARG TZ=America/Chicago

FROM umputun/baseimage:buildgo-latest as build-backend

ADD . /go/src/github.com/umputun/secrets
WORKDIR /go/src/github.com/umputun/secrets/backend

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

ARG TZ

COPY --from=build-backend /go/src/github.com/umputun/secrets/backend/secrets /srv/
COPY --from=build-frontend /srv/webapp/public/ /srv/docroot

RUN \
    apk add --update --no-cache tzdata && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime &&\
    adduser -s /bin/bash -D -u 1001 secrets && \
    chown -R secrets:secrets /srv

EXPOSE 8080
USER secrets
WORKDIR /srv
VOLUME ["/srv/docroot"]

ENTRYPOINT ["/srv/secrets"]