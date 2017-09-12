FROM golang:alpine

ARG TZ=America/Chicago

# add user and set TZ
RUN \
 apk add --update tzdata && \
 adduser -s /bin/bash -D -u 1001 secrets && \
 mkdir -p /srv && chown -R secrets:secrets /srv && \
 cp /usr/share/zoneinfo/$TZ /etc/localtime && \
 echo $TZ > /etc/timezone && \
 rm -rf /var/cache/apk/*

# build service
ADD app /go/src/github.com/umputun/secrets/app
RUN \
 apk add --update --no-progress git && \
 cd /go/src/github.com/umputun/secrets/app && \
 go get -v && \
 go build -ldflags "-X main.revision=$(date +%Y%m%d-%H%M%S)" -o /srv/secrets && \
 apk del git && rm -rf /go/src/* && rm -rf /var/cache/apk/*

# build webapp
ADD webapp /srv/webapp
RUN \
 apk --update --no-progress add nodejs-current nodejs-current-npm git python make g++ && \
 cd /srv/webapp && \
 npm i --production && npm run build && \
 mv -fv /srv/webapp/public/ /srv/docroot && \
 rm -rf  /srv/webapp && \
 apk del nodejs-current nodejs-current-npm git python make g++ && rm -rf /var/cache/apk/*

RUN \
 echo "#!/bin/sh" > /srv/exec.sh && \
 echo "tail -F /srv/logs/secrets.log &" /srv/exec.sh && \
 echo "/srv/secrets >> /srv/logs/secrets.log" >> /srv/exec.sh && \
 chmod +x /srv/exec.sh


EXPOSE 8080
USER secrets
WORKDIR /srv
VOLUME ["/srv/docroot", "/srv/logs"]
ENTRYPOINT ["/srv/exec.sh"]
