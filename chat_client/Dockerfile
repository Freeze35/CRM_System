# Используем базовый образ Go
FROM golang:1.23 AS builder

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем файлы в контейнер
COPY ./chat_client .


# Копируем сертификат
COPY ./rootca/ca.crt ./sslkeys/ca.crt

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарный файл для chat_client
RUN GOOS=linux GOARCH=amd64 go build -o chat_client main.go

# Указываем рабочую директорию (необязательно)
WORKDIR /app

# Запускаем приложение
CMD ["./chat_client"]
