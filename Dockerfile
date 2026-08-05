FROM docker.arvancloud.ir/golang:1.25.7-alpine AS builder
WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN GOPROXY="https://goproxy.io,direct" go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bot ./cmd/bot

# Final stage
FROM docker.arvancloud.ir/alpine:latest

# Use ArvanCloud alpine mirror instead of dl-cdn.alpinelinux.org
# (dl-cdn is blocked/filtered on this server, causing TLS handshake failures)
RUN ALPINE_VERSION=$(cat /etc/alpine-release | cut -d'.' -f1,2) && \
    sed -i "s#https://dl-cdn.alpinelinux.org/alpine/v[0-9.]*#https://mirror.arvancloud.ir/alpine/v${ALPINE_VERSION}#g" /etc/apk/repositories && \
    cat /etc/apk/repositories && \
    apk update && \
    apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bot .
COPY --from=builder /build/migrations ./migrations

# Create logs directory
RUN mkdir -p /app/logs

EXPOSE 8080
CMD ["./bot"]