# Шаг 1: Сборка gRPC-сервиса
FROM golang:1.23 AS builder

WORKDIR /app

# Копируем файлы для сборки gRPC-сервиса
COPY ./redis .
RUN go mod download
RUN go build -o redis_service ./main.go

# Шаг 2: Установка Redis
FROM redis:6.2 AS redis

# Шаг 3: Финальный образ с gRPC-сервисом и Redis
FROM ubuntu:22.04

WORKDIR /app

# Устанавливаем Redis
COPY --from=redis /usr/local/bin/redis-server /usr/local/bin/
COPY --from=redis /usr/local/bin/redis-cli /usr/local/bin/
COPY ./redis/redis.conf /usr/local/etc/redis.conf
COPY ./rootca/ca.crt ./sslkeys/ca.crt
COPY ./redis/sslkeys ./sslkeys
COPY ./redis/.env .env

# Копируем бинарный файл gRPC-сервиса
COPY --from=builder /app/redis_service /app/redis_service

# Запускаем Redis и gRPC-сервис
CMD ["sh", "-c", "redis-server --daemonize yes && /app/redis_service"]
