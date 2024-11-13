# Используйте базовый образ NGINX для сборки
FROM nginx:alpine AS builder

# Установите необходимые пакеты для сборки
RUN apk add --no-cache \
    git \
    gcc \
    g++ \
    make \
    pcre-dev \
    zlib-dev \
    linux-headers \
    openssl-dev \
    libjwt-dev \
    jansson-dev  # Добавлено для установки Jansson

# Установите переменную окружения для версии NGINX
ENV NGINX_VERSION=1.27.2

# Скачайте и соберите NGINX
RUN cd /tmp \
    # Скачиваем и распаковываем NGINX
    && wget https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz \
    && tar -xzvf nginx-${NGINX_VERSION}.tar.gz \
    # Скачиваем и распаковываем модуль ngx-http-auth-jwt-module
    && wget https://github.com/TeslaGov/ngx-http-auth-jwt-module/archive/refs/heads/master.tar.gz -O ngx-http-auth-jwt-module.tar.gz \
    && tar -xzvf ngx-http-auth-jwt-module.tar.gz \
    # Конфигурируем NGINX с добавлением модуля и флага --with-compat
    && cd nginx-${NGINX_VERSION} \
    && ./configure --add-dynamic-module=/tmp/ngx-http-auth-jwt-module-master --with-compat --with-cc-opt='-DNGX_LINKED_LIST_COOKIES=1' \
    # Собираем только динамические модули
    && make modules \
    # Собираем и устанавливаем NGINX
    && make && make install

# Переместите файл модуля из /tmp в /etc/nginx/modules с добавлением суффикса .so
RUN mv /tmp/nginx-${NGINX_VERSION}/objs/ngx_http_auth_jwt_module.so /etc/nginx/modules/ngx_http_auth_jwt_module.so

# Копируем ключ в сборку для проверки ключа авторизации.
COPY ./nginx/opensslkeys/public_key.pem /etc/nginx/opensslkeys/public_key.pem

# Копируем сертификаты в контейнер
COPY ./nginx/sslkeys/server.pem /etc/nginx/certs/server.pem
COPY ./nginx/sslkeys/server.key /etc/nginx/certs/server.key
COPY ./rootca/ca.crt /etc/nginx/certs/ca.crt

# Копируйте конфигурационный файл NGINX
COPY ./nginx/nginx.conf /etc/nginx/nginx.conf

EXPOSE 80
EXPOSE 443

# Команда для запуска NGINX
CMD ["nginx", "-g", "daemon off;"]
