# Production Multi-stage Dockerfile for Gerege Template ERP Platform Backend
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download

# Copy full source
COPY . /app

WORKDIR /app/backend
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrate ./cmd/migrate/main.go

# Production Scratch/Alpine image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Ulaanbaatar

WORKDIR /app

COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/migrate /app/migrate
COPY --from=builder /app/catalog /app/catalog

EXPOSE 8080

CMD ["/app/api"]
