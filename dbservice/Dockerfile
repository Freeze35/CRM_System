# Этап сборки
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY . .

# Устанавливаем зависимости
RUN go mod download

# Собираем статически слинкованный бинарный файл для Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dbservice main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем бинарный файл из этапа сборки
COPY --from=builder /app/dbservice /app/dbservice

# Копируем миграции
COPY --from=builder /app/migrations /app/migrations

# Копируем .env файл
COPY --from=builder /app/.env /app/.env

# Указываем порт для приложения
EXPOSE 8081

# Запускаем приложение
CMD ["/app/dbservice"]
