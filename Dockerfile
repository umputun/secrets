ARG TZ=America/Chicago

FROM umputun/baseimage:buildgo-latest AS build-backend

ARG CI
ARG GIT_BRANCH
ARG SKIP_TEST
ARG GITHUB_SHA

ADD . /build/secrets
WORKDIR /build/secrets

# run tests and linters
RUN \
    if [ -z "$SKIP_TEST" ] ; then \
    go test -timeout=30s  ./... && \
    golangci-lint run ; \
    else echo "skip tests and linter" ; fi

RUN \
    version=$(/script/version.sh) && \
    echo "version=$version" && \
    cd app && go build -o /build/secrets.bin -ldflags "-X main.revision=${version} -s -w"


FROM umputun/baseimage:app-latest

# enables automatic changelog generation by tools like Dependabot
LABEL org.opencontainers.image.source="https://github.com/umputun/secrets"

COPY --from=build-backend /build/secrets.bin /srv/secrets

WORKDIR /srv
EXPOSE 8080


CMD ["/srv/secrets"]
ENTRYPOINT ["/init.sh"]
