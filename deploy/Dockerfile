# Production Multi-stage Dockerfile for the Gerege Nexus backend
# Build context is the repository root: docker build -f deploy/Dockerfile .
#
# The Go toolchain version must satisfy backend/go.mod. Official golang images
# pin GOTOOLCHAIN=local, so an image older than the go directive fails the
# build outright.
FROM golang:1.26-alpine AS builder

# The upstream AI client currently requires Go 1.26. Keep auto enabled so
# patch-level toolchain requirements do not make an otherwise valid build fail.
ENV GOTOOLCHAIN=auto

WORKDIR /app/backend

# Copy dependency manifests first so `go mod download` stays cached.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the module source
COPY backend/ ./

# Which platform this build claims to be. Manifests declare the platform they
# need ("platform": ">=1.1.0"), and once the catalogue comes from a registry
# outside this repository, that constraint is checked against whatever is
# stamped here — an unstamped build would tell every store it is 1.0.0 for ever.
# The default is the version the constant carries, so nothing changes for a
# build that does not pass it. It must be valid semver: manifest validation
# parses it.
ARG VERSION=1.0.0

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-w -s -X github.com/gerege-systems/open-gerege-nexus/backend/internal/platform.PlatformVersion=${VERSION}" \
    -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/migrate ./cmd/migrate

# The App Store registry ships in the same image as the platform it serves.
#
# It is a separate service — its own database, its own domain, its own compose
# stack — but it is built from the same module so that it and the client that
# consumes it share one definition of the catalogue format rather than two
# descriptions of it. One image means one build to keep them in step.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/appstore ./cmd/appstore
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /bin/catalog-sign ./cmd/catalog-sign

# Production Alpine image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Ulaanbaatar

# Run as an unprivileged user.
RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/migrate /app/migrate
COPY --from=builder /bin/appstore /app/appstore
COPY --from=builder /bin/catalog-sign /app/catalog-sign
# App-store catalog + goose migrations are read from disk at runtime.
COPY catalog/ /app/catalog/
COPY backend/db/migrations/ /app/db/migrations/

USER app

EXPOSE 8080

CMD ["/app/api"]
