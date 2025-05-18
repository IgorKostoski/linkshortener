FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /app/linkshortener .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/linkshortener /app/linkshortener


EXPOSE 8080

ENTRYPOINT ["/app/linkshortener"]