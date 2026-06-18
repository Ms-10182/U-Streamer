# Build stage
FROM golang:1.20-alpine AS builder
WORKDIR /src

# Cache and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/app

# Final image
FROM alpine:3.18
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/app .
USER appuser

EXPOSE 8080
CMD ["./app"]
