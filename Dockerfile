ARG TZ=America/Chicago

FROM umputun/baseimage:buildgo-latest as build-backend

ARG CI
ARG GIT_BRANCH
ARG SKIP_TEST

ENV GOFLAGS="-mod=vendor"

ADD backend /build/secrets
ADD .git /build/secrets/.git
WORKDIR /build/secrets

# run tests and linters
RUN \
    if [ -z "$SKIP_TEST" ] ; then \
    go test -timeout=30s  ./... && \
    golangci-lint run ; \
    else echo "skip tests and linter" ; fi

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(/script/git-rev.sh); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o secrets -ldflags "-X main.revision=${version} -s -w" ./app


FROM node:10.19.0-alpine3.11 as build-frontend

ADD frontend /srv/frontend
RUN apk add --no-cache --update git python make g++
RUN \
    cd /srv/frontend && \
    npm i gulp && \
    npm i --production && npm run build


FROM umputun/baseimage:app-latest

COPY --from=build-backend /build/secrets/secrets /srv/secrets
COPY --from=build-frontend /srv/frontend/public/ /srv/docroot

WORKDIR /srv
EXPOSE 8080


CMD ["/srv/secrets"]
ENTRYPOINT ["/init.sh"]
