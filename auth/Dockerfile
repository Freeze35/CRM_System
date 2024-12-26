# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY ./auth .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для Linux
RUN GOOS=linux GOARCH=amd64 go build -o auth main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Копируем .env файл
COPY ./auth/.env .env
COPY ./rootca/ca.crt ./sslkeys/ca.crt

# Указываем порт для приложения
EXPOSE ${AUTH_SERVICE_GRPC_PORT}
EXPOSE ${AUTH_SERVICE_HTTP_PORT}

# Запускаем приложение
CMD ["./auth"]
