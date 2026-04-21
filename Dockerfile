# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend

COPY . /app
WORKDIR /app/frontend

RUN npm install -g @angular/cli@latest && \
    npm install -g @ionic/cli@latest

RUN npm ci

RUN ionic build --prod

FROM golang:1.24-alpine AS backend

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
    VERSION_PACKAGE=github.com/Valentin-Kaiser/go-core/version && \
    go mod tidy && \
    go build -ldflags "-s -w -X ${VERSION_PACKAGE}.GitTag=${GIT_TAG} -X ${VERSION_PACKAGE}.GitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.GitShort=${GIT_SHORT} -X ${VERSION_PACKAGE}.BuildDate=${BUILD_TIME}" \
    -o /hdns cmd/main.go

FROM alpine:latest

WORKDIR /
COPY --from=backend /hdns /hdns

RUN apk add --no-cache ca-certificates tzdata curl

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
CMD curl -f http://localhost:8080 || exit 1

EXPOSE 8080/tcp

ENTRYPOINT ["./hdns"]