
FROM golang:1.21-alpine AS builder


ARG TARGETPLATFORM
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .


RUN echo "Building for TARGETPLATFORM: $TARGETPLATFORM, TARGETARCH: $TARGETARCH" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-s -w" -v -o /app/linkshortener .

# Stage 2: Create a small, production-ready image
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/linkshortener /app/linkshortener

EXPOSE 8080
ENTRYPOINT ["/app/linkshortener"]