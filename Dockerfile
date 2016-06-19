FROM golang:alpine

# add user and set TZ
RUN \
 apk add --update tzdata && \
 adduser -s /bin/bash -D -u 1001 secrets && \
 mkdir -p /srv && chown -R secrets:secrets /srv && \
 cp /usr/share/zoneinfo/America/Chicago /etc/localtime && \
 echo "America/Chicago" > /etc/timezone && \
 rm -rf /var/cache/apk/*

ADD app /go/src/github.com/umputun/secrets/app
RUN \
 apk add --update git && \
 cd /go/src/github.com/umputun/secrets/app && \
 go get -v && \
 go build -ldflags "-X main.revision=$(date +%Y%m%d-%H%M%S)" -o /srv/secrets && \
 apk del git && rm -rf /go/src/* && rm -rf /var/cache/apk/*

EXPOSE 8080
USER secrets
WORKDIR /srv

ENTRYPOINT ["/srv/secrets"]
