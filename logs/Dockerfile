# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY ./logs .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для Linux
RUN GOOS=linux GOARCH=amd64 go build -o logs main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Копируем .env файл
COPY ./logs/.env .env
COPY ./rootca/ca.crt ./sslkeys/ca.crt

# Указываем порт для приложения
EXPOSE ${LOGS_SERVICE_HTTP_PORT}

# Запускаем приложение
CMD ["./logs"]
