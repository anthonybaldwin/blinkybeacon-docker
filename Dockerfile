FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev linux-headers hidapi-dev libusb-dev eudev-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o fsbeacon ./cmd/fsbeacon && \
    go build -o beacon-api ./cmd/beacon-api

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache hidapi libusb eudev-libs

# Copy binaries
COPY --from=builder /build/fsbeacon /usr/local/bin/fsbeacon
COPY --from=builder /build/beacon-api /usr/local/bin/beacon-api

# Expose API port
EXPOSE 9100

# Run API server
CMD ["/usr/local/bin/beacon-api"]
