FROM nginx:stable-alpine

ADD nginx.conf /etc/nginx/nginx.conf
ADD secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD robots.txt /robots.txt

ADD entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
RUN apk add  --update letsencrypt

CMD ["/entrypoint.sh"]
