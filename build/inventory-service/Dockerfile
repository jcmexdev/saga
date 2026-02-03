# Etapa 1: build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copiamos módulos primero para cache
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el resto del código
COPY . .

# Compilamos el binario del order-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o inventory-service ./cmd/inventory-service

# Etapa 2: imagen final
FROM alpine:3.20

WORKDIR /app

# Opcional: timezone
ENV TZ=Etc/UTC

COPY --from=builder /app/inventory-service /app/inventory-service

EXPOSE 9092

ENTRYPOINT ["/app/inventory-service"]
