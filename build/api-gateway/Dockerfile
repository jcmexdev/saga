# Etapa 1: build
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compilamos el binario del api-gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-gateway ./cmd/api-gateway

# Etapa 2: imagen final
FROM alpine:3.20

WORKDIR /app

ENV TZ=Etc/UTC

COPY --from=builder /app/api-gateway /app/api-gateway

EXPOSE 8080

ENTRYPOINT ["/app/api-gateway"]
