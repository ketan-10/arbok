FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/server.bin ./cmd/server

FROM alpine:latest

# Install required packages for WireGuard
RUN apk add --no-cache \
    ca-certificates \
    wireguard-tools \
    iptables

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/bin/server.bin ./arbok
COPY --from=builder /app/config.sample.toml ./config.toml

RUN chmod +x ./arbok

EXPOSE 8080
EXPOSE 54321/udp

CMD ["./arbok", "--config", "config.toml"]