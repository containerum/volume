FROM golang:1.10-alpine as builder

WORKDIR /go/src/git.containerum.net/ch/volume-manager
COPY . .
RUN go build -v -ldflags="-w -s" -o /bin/volume-manager ./cmd/volume-manager

FROM alpine:3.7
RUN mkdir -p /app
COPY --from=builder /bin/volume-manager /app

ENV MODE="release" \
    LOG_LEVEL=4 \
    DB_USER="volume-manager" \
    DB_PASSWORD="vTsnHHnI" \
    DB_HOST="postgres:5432" \
    DB_SSLMODE="false" \
    DB_BASE="volume-manager" \
    LISTEN_ADDR=":4343" \
    BILLING_ADDR="" \
    KUBE_API_ADDR="kube-api:1214"

EXPOSE 4343

CMD "/app/volume-manager"
