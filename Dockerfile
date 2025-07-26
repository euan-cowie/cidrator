# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o cidrator .

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the binary
COPY --from=builder /build/cidrator /usr/local/bin/cidrator

# Create non-root user
USER nobody:nobody

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/cidrator"]

# Default command
CMD ["--help"]

# Metadata
LABEL org.opencontainers.image.title="Cidrator"
LABEL org.opencontainers.image.description="Comprehensive network analysis and manipulation toolkit built with Go"
LABEL org.opencontainers.image.vendor="Euan Cowie"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/euan-cowie/cidrator"
LABEL org.opencontainers.image.documentation="https://github.com/euan-cowie/cidrator/wiki" 