FROM golang:1.23-alpine AS builder
WORKDIR /app

# 1. Copy only what is needed for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy source code
COPY . .

# 3. Build
RUN go build -o /main ./cmd/appointment-service/main.go

# Final lightweight image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /main .

EXPOSE 8082 9092
CMD ["./main"]
