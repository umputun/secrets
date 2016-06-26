FROM nginx:alpine

ADD nginx.conf /etc/nginx/nginx.conf
ADD secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD robots.txt /usr/share/nginx/html/robots.txt

ADD entrypint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
