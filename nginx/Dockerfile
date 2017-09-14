FROM nginx:stable-alpine

ARG TZ=America/Chicago

ADD conf/nginx.conf /etc/nginx/nginx.conf
ADD conf/secrets-nginx.conf /etc/nginx/conf.d/secrets.conf
ADD conf/robots.txt /robots.txt

ADD script/entrypoint.sh /entrypoint.sh
ADD script/le.sh /le.sh

RUN \
 chmod +x /entrypoint.sh && \
 chmod +x /le.sh && \
 apk add  --update certbot tzdata openssl && \
 cp /usr/share/zoneinfo/$TZ /etc/localtime && \
 echo $TZ > /etc/timezone && \
 rm -rf /var/cache/apk/*

CMD ["/entrypoint.sh"]
