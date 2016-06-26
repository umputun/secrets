FROM nginx:stable-alpine

ADD nginx.conf /etc/nginx/nginx.conf
ADD secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD robots.txt /robots.txt

ADD entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
