# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY ./admin_control .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для Linux
RUN GOOS=linux GOARCH=amd64 go build -o admin_control main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Копируем .env файл
COPY ./admin_control/.env .env
COPY ./rootca/ca.crt ./sslkeys/ca.crt

# Указываем порт для приложения
EXPOSE ${ADMIN_SERVICE_HTTP_PORT}

# Запускаем приложение
CMD ["./admin_control"]
