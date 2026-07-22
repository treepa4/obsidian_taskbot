# 1. Stage сборки
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o obsidian-bot ./cmd/bot/main.go

# 2. Финальный образ
FROM alpine:latest

# Ставим git и ssh-клиент (нужно для работы гита на VPS)
RUN apk --no-cache add ca-certificates git openssh-client tzdata

# Ставим часовой пояс (например, Europe/Moscow, чтобы время 08:00 совпадало)
ENV TZ=Europe/Moscow

WORKDIR /root/

# Копируем скомпилированный бинарник
COPY --from=builder /app/obsidian-bot .

CMD ["./obsidian-bot"]