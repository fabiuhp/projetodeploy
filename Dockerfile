FROM golang:1.24.3-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/main .

RUN chown appuser:appgroup main

USER appuser

ENV PORT=8080

EXPOSE 8080

# Add a health check endpoint
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/weather/01310100 || exit 1

# Add debugging information
CMD echo "Starting application on port ${PORT}" && \
    echo "Current directory: $(pwd)" && \
    echo "Files in directory: $(ls -la)" && \
    ./main 