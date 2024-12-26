# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY ./timer .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для Linux
RUN GOOS=linux GOARCH=amd64 go build -o timer main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Копируем .env файл
COPY ./timer/.env .env
COPY ./rootca/ca.crt ./sslkeys/ca.crt

# Указываем порт для приложения
EXPOSE ${TIMER_SERVICE_PORT}

# Запускаем приложение
CMD ["./timer"]
