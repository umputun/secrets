#!/bin/sh
echo "start nginx"

echo "ssl_key=${SSL_KEY:=/etc/nginx/ssl/secrets-key.pem}, ssl_cert=${SSL_CERT:=/etc/nginx/ssl/secrets-crt.pem}"

sed -i "s/SECRETS_KEY/${SSL_KEY}/g" /etc/nginx/conf.d/secrets.conf
sed -i "s/SECRETS_CERT/${SSL_CERT}/g" /etc/nginx/conf.d/secrets.conf

nginx -g "daemon off;"
