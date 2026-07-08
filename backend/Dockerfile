# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_TARGET=api

RUN if [ "$APP_TARGET" = "api" ]; then \
      go build -o /out/app ./cmd/api; \
    elif [ "$APP_TARGET" = "worker" ]; then \
      go build -o /out/app ./cmd/worker; \
    else \
      echo "APP_TARGET invalido: $APP_TARGET. Use api ou worker." && exit 1; \
    fi

FROM alpine:3.20 AS runner

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

ENV TZ=America/Sao_Paulo
ENV APP_ENV=production

COPY --from=builder /out/app /app/app
COPY --from=builder /app/db/migrations /app/db/migrations

EXPOSE 8080

CMD ["/app/app"]
