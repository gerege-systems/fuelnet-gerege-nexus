FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /app/backend
RUN go mod download
COPY backend/ /app/backend/
COPY catalog/ /app/catalog/
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api-server ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/api-server .
COPY catalog/ /app/catalog/
EXPOSE 8080
CMD ["./api-server"]
