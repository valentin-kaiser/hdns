# syntax=docker/dockerfile:1

FROM node:24-alpine AS frontend

COPY . /app
WORKDIR /app/frontend

RUN npm install -g @angular/cli@latest

RUN npm i

RUN ng build

FROM golang:1.26-alpine AS backend

COPY --from=frontend /app /app

WORKDIR /app/backend

ENV CGO_ENABLED=1

RUN apk add --no-cache build-base git
RUN go mod download
RUN set -eux && \
    GIT_TAG=$(git describe --tags || echo "unknown") && \
    GIT_COMMIT=$(git rev-parse HEAD) && \
    GIT_SHORT=$(git rev-parse --short HEAD) && \
    BUILD_TIME=$(date +%FT%T%z) && \
    VERSION_PACKAGE=github.com/valentin-kaiser/go-core/version && \
    go mod tidy && \
    go build -ldflags "-s -w -X ${VERSION_PACKAGE}.GitTag=${GIT_TAG} -X ${VERSION_PACKAGE}.GitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.GitShort=${GIT_SHORT} -X ${VERSION_PACKAGE}.BuildDate=${BUILD_TIME}" \
    -o /app/hdns cmd/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=backend /app/hdns /app/hdns

RUN apk add --no-cache ca-certificates tzdata curl

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
CMD curl -fk https://localhost || exit 1

EXPOSE 443/tcp

ENTRYPOINT ["/app/hdns"]