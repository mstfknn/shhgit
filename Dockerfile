FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /shhgit .

FROM alpine:3.19 AS runtime
WORKDIR /app

RUN apk update && apk add --no-cache git ca-certificates && \
    addgroup -S shhgit && adduser -S -G shhgit shhgit && \
    mkdir -p /tmp/shhgit && chown shhgit:shhgit /tmp/shhgit

COPY --from=builder /shhgit /app/shhgit

USER shhgit

ENTRYPOINT [ "/app/shhgit" ]