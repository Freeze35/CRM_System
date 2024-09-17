# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY . .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для Linux
RUN GOOS=linux GOARCH=amd64 go build -o auth main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Копируем .env файл
COPY .env .env

# Указываем порт для приложения
EXPOSE 8081

# Запускаем приложение
CMD ["./auth"]
