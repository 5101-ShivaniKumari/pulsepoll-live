# Multi-stage Docker build for Go Backend
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# Final minimal scratch / alpine runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
