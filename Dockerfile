# Используем официальный образ Go для сборки
FROM golang:1.25-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение (modernc.org/sqlite - pure Go, CGO не требуется)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o license-service ./cmd/license-service

# Финальный образ
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов (если понадобится)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Копируем бинарный файл из builder
COPY --from=builder /app/license-service .

# Создаем директорию для базы данных
RUN mkdir -p /app/storage

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./license-service"]

