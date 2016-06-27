FROM nginx:stable-alpine

ADD nginx.conf /etc/nginx/nginx.conf
ADD secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD robots.txt /robots.txt

ADD entrypoint.sh /entrypoint.sh
ADD le.sh /le.sh

RUN \
 chmod +x /entrypoint.sh && \
 chmod +x /le.sh && \
 apk add  --update letsencrypt && \
 rm -rf /var/cache/apk/*
 
CMD ["/entrypoint.sh"]
