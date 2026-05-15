FROM golang:1.23-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/buhpro-api ./cmd/api

FROM alpine:3.21
WORKDIR /app
RUN adduser -D -g '' appuser
RUN mkdir -p /app/uploads && chown -R appuser:appuser /app/uploads

COPY --from=builder /bin/buhpro-api /app/buhpro-api
COPY migrations /app/migrations

USER appuser
EXPOSE 8080
CMD ["/app/buhpro-api"]
