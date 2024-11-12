# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY ./dbservice/ .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для Linux
RUN GOOS=linux GOARCH=amd64 go build -o dbservice main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Копируем .env файл
COPY ./dbservice/.env .env
COPY ./rootca/ca.crt ./rootca/ca.crt

# Указываем порт для приложения
EXPOSE ${DB_SERVICE_PORT}

# Запускаем приложение
CMD ["./dbservice"]
