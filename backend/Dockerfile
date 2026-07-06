FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/worker ./cmd/worker

FROM alpine:latest AS api
WORKDIR /app
COPY --from=builder /app/api /app/api
COPY --from=builder /app/db/migrations /app/db/migrations
EXPOSE 8080
CMD ["/app/api"]

FROM alpine:latest AS worker
WORKDIR /app
COPY --from=builder /app/worker /app/worker
CMD ["/app/worker"]
