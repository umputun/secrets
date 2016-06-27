FROM nginx:stable-alpine

ADD nginx.conf /etc/nginx/nginx.conf
ADD secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD robots.txt /robots.txt

ADD entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

#add acme for letsencrypt
ADD https://github.com/hlandau/acme/releases/download/v0.0.54/acmetool-v0.0.54-linux_amd64.tar.gz /tmp/acmetool.tar.gz
RUN \
 cd /tmp && tar zxf /tmp/acmetool.tar.gz && \
 mv /tmp/acmetool-v0.0.54-linux_amd64/bin/acmetool /usr/bin/acmetool && \
 chmod +x /usr/bin/acmetool && \
 rm -rf  /tmp/acmetool-v0.0.54-linux_amd64

ENTRYPOINT ["/entrypoint.sh"]
