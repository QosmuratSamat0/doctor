FROM golang:1.25-alpine AS builder
WORKDIR /app

# 1. Copy only what is needed for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy source code
COPY . .

# 3. Build
RUN go build -o /main ./cmd/doctor-service/main.go

# Final lightweight image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /main .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8081 9091
CMD ["./main"]
