FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/server.bin ./cmd/server

FROM alpine:latest

# Install required packages
RUN apk add --no-cache \
    ca-certificates \
    wireguard-tools \
    iptables \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 arbok && \
    adduser -D -s /bin/sh -u 1000 -G arbok arbok

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/bin/server.bin ./arbok
COPY --from=builder /app/config.sample.toml ./config.toml

# Set permissions
RUN chmod +x ./arbok && \
    chown arbok:arbok ./arbok ./config.toml

# Switch to non-root user
USER arbok

EXPOSE 8080
EXPOSE 54321/udp

CMD ["./arbok", "--config", "config.toml"]