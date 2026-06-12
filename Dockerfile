# syntax=docker/dockerfile:1

FROM node:24-alpine AS frontend-builder

COPY . /app
WORKDIR /app/frontend

RUN npm i
RUN npx ng build

FROM golang:1.26-alpine AS backend-builder

COPY --from=frontend-builder /app /app

WORKDIR /app/backend

ENV CGO_ENABLED=1

RUN apk add --no-cache build-base git
RUN go mod download
RUN set -eux && \
    GIT_TAG=$(git -C /app describe --tags 2>/dev/null || echo "unknown") && \
    GIT_COMMIT=$(git -C /app rev-parse HEAD 2>/dev/null || echo "unknown") && \
    GIT_SHORT=$(git -C /app rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date +%FT%T%z) && \
    VERSION_PACKAGE=github.com/valentin-kaiser/go-core/version && \
    go mod tidy && \
    go build -ldflags "-s -w -X ${VERSION_PACKAGE}.GitTag=${GIT_TAG} -X ${VERSION_PACKAGE}.GitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.GitShort=${GIT_SHORT} -X ${VERSION_PACKAGE}.BuildDate=${BUILD_TIME}" \
    -o /app/hdns ./cmd/main.go

FROM golang:1.26-alpine AS worker-builder

COPY . /app
WORKDIR /app/worker

ENV CGO_ENABLED=0

RUN apk add --no-cache git
RUN go mod download
RUN set -eux && \
    GIT_TAG=$(git -C /app describe --tags 2>/dev/null || echo "unknown") && \
    GIT_COMMIT=$(git -C /app rev-parse HEAD 2>/dev/null || echo "unknown") && \
    GIT_SHORT=$(git -C /app rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date +%FT%T%z) && \
    VERSION_PACKAGE=github.com/valentin-kaiser/go-core/version && \
    go build -ldflags "-s -w -X ${VERSION_PACKAGE}.GitTag=${GIT_TAG} -X ${VERSION_PACKAGE}.GitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.GitShort=${GIT_SHORT} -X ${VERSION_PACKAGE}.BuildDate=${BUILD_TIME}" \
    -o /app/hdns-worker ./cmd/main.go

FROM alpine:latest AS hdns-worker

WORKDIR /app
COPY --from=worker-builder /app/hdns-worker /app/hdns-worker

RUN apk add --no-cache ca-certificates tzdata

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/ || exit 1

EXPOSE 8080/tcp

ENTRYPOINT ["/app/hdns-worker"]

FROM alpine:latest AS hdns

WORKDIR /app
COPY --from=backend-builder /app/hdns /app/hdns

RUN apk add --no-cache ca-certificates tzdata curl

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
CMD curl -fk https://localhost || exit 1

EXPOSE 443/tcp

ENTRYPOINT ["/app/hdns"]