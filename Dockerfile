ARG TZ=America/Chicago

FROM umputun/baseimage:buildgo-latest as build-backend

ARG CI
ARG GIT_BRANCH
ARG SKIP_TEST
ARG GITHUB_SHA

ENV GOFLAGS="-mod=vendor"

ADD . /build/secrets
WORKDIR /build/secrets

# run tests and linters
RUN \
    if [ -z "$SKIP_TEST" ] ; then \
    go test -timeout=30s  ./... && \
    golangci-lint run ; \
    else echo "skip tests and linter" ; fi

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(git rev-parse --abbrev-ref HEAD)-$(git log -1 --format=%h)-$(date +%Y%m%dT%H:%M:%S); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    cd app && go build -o /build/secrets.bin -ldflags "-X main.revision=${version} -s -w"


FROM umputun/baseimage:app-latest

COPY --from=build-backend /build/secrets.bin /srv/secrets
COPY --from=build-backend /build/secrets/ui/static /srv/ui/static/

WORKDIR /srv
EXPOSE 8080


CMD ["/srv/secrets"]
ENTRYPOINT ["/init.sh"]
