# Production Multi-stage Dockerfile for Gerege Template ERP Platform Backend
# Build context is the repository root: docker build -f deploy/Dockerfile .
#
# The Go toolchain version must satisfy backend/go.mod. Official golang images
# pin GOTOOLCHAIN=local, so an image older than the go directive fails the
# build outright.
FROM golang:1.25-alpine AS builder

# go.mod pins an exact patch (go 1.25.7). Official golang images set
# GOTOOLCHAIN=local, which hard-fails if the base image is an older patch —
# "auto" lets Go fetch the exact toolchain the module asks for.
ENV GOTOOLCHAIN=auto

WORKDIR /app/backend

# Copy dependency manifests first so `go mod download` stays cached.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the module source
COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/migrate ./cmd/migrate

# Production Alpine image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Ulaanbaatar

# Run as an unprivileged user.
RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/migrate /app/migrate
# App-store catalog + goose migrations are read from disk at runtime.
COPY catalog/ /app/catalog/
COPY backend/db/migrations/ /app/db/migrations/

USER app

EXPOSE 8080

CMD ["/app/api"]
