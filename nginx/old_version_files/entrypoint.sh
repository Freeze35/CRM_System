#!/bin/sh

# Загрузка переменных из .env файла
export $(grep -v '^#' /etc/nginx/.env | xargs)

# Вывод переменной для отладки
echo "JWT_SECRET_KEY = $JWT_SECRET_KEY"

# Генерация конфигурации NGINX из шаблона
envsubst < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

# Запуск NGINX
exec nginx -g 'daemon off;'