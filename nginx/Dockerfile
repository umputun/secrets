FROM nginx:stable-alpine

ARG TZ=America/Chicago

ADD nginx.conf /etc/nginx/nginx.conf
ADD secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD robots.txt /robots.txt

ADD entrypoint.sh /entrypoint.sh
ADD le.sh /le.sh

RUN \
 chmod +x /entrypoint.sh && \
 chmod +x /le.sh && \
 apk add  --update letsencrypt tzdata openssl && \
 cp /usr/share/zoneinfo/$TZ /etc/localtime && \
 echo $TZ > /etc/timezone && \
 rm -rf /var/cache/apk/*

CMD ["/entrypoint.sh"]
