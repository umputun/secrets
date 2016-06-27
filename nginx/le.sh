#!/bin/sh

if [ "$LETSENCRYPT" = "true" ]; then
    letsencrypt certonly -t -n --agree-tos --email "${LE_EMAIL}" --webroot -w /usr/share/nginx/html -d $LE_FQDN -v
    cp -fv /etc/letsencrypt/live/$LE_FQDN/privkey.pem /etc/nginx/ssl/secrets-key.pem
    cp -fv /etc/letsencrypt/live/$LE_FQDN/fullchain.pem /etc/nginx/ssl/secrets-cert.pem
else
    echo "letsencrypt disabled"
fi
