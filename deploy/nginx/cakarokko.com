# HTTP -> HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name cakarokko.com www.cakarokko.com;
    location ^~ /.well-known/acme-challenge/ { allow all; }
    return 301 https://$host$request_uri;
}

# HTTPS (uygulama)
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name cakarokko.com www.cakarokko.com;

    # --- SSL ---
    ssl_certificate     /etc/letsencrypt/live/cakarokko.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cakarokko.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # --- Güvenlik header'ları ---
    add_header Content-Security-Policy "default-src 'self'; img-src 'self' data: https:; script-src 'self'; style-src 'self';" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

    # NOT: Burada root/index YOK; her şey uygulamaya proxy edilecek.

    # --- API ---
    location ^~ /api/ {
        limit_req zone=perip burst=20 nodelay;

        proxy_pass         http://127.0.0.1:8080;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;

        proxy_http_version 1.1;
        proxy_set_header   Connection "";
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
        client_max_body_size 2m;
        proxy_request_buffering on;
    }

    # --- Uygulama (tüm diğer yollar) ---
    location / {
        proxy_pass         http://127.0.0.1:8080;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;

        proxy_http_version 1.1;
        proxy_set_header   Connection "";
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;

        # Teşhis için (geçici):
        add_header X-Vhost "cakarokko-443" always;
    }
}
