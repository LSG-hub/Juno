FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy source code and go mod files
COPY . ./
RUN go mod tidy
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8082

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8082/health || exit 1

CMD ["./main"]