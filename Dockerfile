# Multi-stage Dockerfile for Go Microservices
FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/service ./cmd/${SERVICE_NAME}

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/
COPY --from=builder /app/service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8000 8001 8002 8003 8004 8005

CMD ["./service"]
