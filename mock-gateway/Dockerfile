FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
# If go.sum exists, copy it too.
COPY . .
RUN go build -o mock-gateway .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mock-gateway .
EXPOSE 8080
CMD ["./mock-gateway"]
